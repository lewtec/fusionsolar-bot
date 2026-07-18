package fusionsolar

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"gopkg.in/gomail.v2"
)

type App struct {
	User             string
	Password         string
	SmtpUser         string
	SmtpFrom         string
	SmtpPasswd       string
	SmtpServer       string
	SmtpDestinations []string
	SentryDsn        string
	BrowserCDP       string
	Verbose          bool
	MaxLoginRetries  int
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// reportPanic sends a recovered panic value to Sentry and returns it as an error
// so callers do not treat Must* (or other) panics as a successful Run.
func reportPanic(r any) error {
	sentry.CurrentHub().Recover(r)
	if err, ok := r.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", r)
}

func (a *App) Run(ctx context.Context) (err error) {
	// Setup Sentry
	a.setupSentry()
	defer sentry.Flush(2 * time.Second)
	// Must* rod helpers panic on failure. Convert panics into a non-nil error so
	// the CLI exits non-zero instead of looking successful after sentry.Recover().
	defer func() {
		if r := recover(); r != nil {
			err = reportPanic(r)
		}
	}()

	// Validate credentials
	if a.User == "" || a.Password == "" {
		return fmt.Errorf("[!] Usuário e senha fusionsolar não fornecidos")
	}

	// Check if email is enabled
	var missingEmailParams []string
	if a.SmtpUser == "" {
		missingEmailParams = append(missingEmailParams, "SMTP_USER")
	}
	if a.SmtpPasswd == "" {
		missingEmailParams = append(missingEmailParams, "SMTP_PASSWD")
	}
	if a.SmtpServer == "" {
		missingEmailParams = append(missingEmailParams, "SMTP_SERVER")
	}
	if len(a.SmtpDestinations) == 0 {
		missingEmailParams = append(missingEmailParams, "SMTP_DESTINATIONS")
	}

	emailEnabled := len(missingEmailParams) == 0
	if !emailEnabled {
		slog.Warn("[!] Funcionalidade de email desativada", "motivo", "Variáveis de ambiente/flags faltando", "missing", strings.Join(missingEmailParams, ", "))
	}

	// Setup Browser
	browser, err := a.setupBrowser(ctx)
	if err != nil {
		return err
	}
	defer browser.MustClose()

	// Login
	page, err := a.loginToFusionSolar(ctx, browser)
	if err != nil {
		return fmt.Errorf("failed to login to FusionSolar: %w", err)
	}

	// Get Stations
	stationsData, err := a.getStations(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to get stations: %w", err)
	}

	// Collect Data
	emailBody, attachments, err := a.collectStationData(ctx, page, stationsData)
	if err != nil {
		return fmt.Errorf("failed to collect station data: %w", err)
	}

	fmt.Println(emailBody)

	if emailEnabled {
		if a.SmtpFrom == "" {
			a.SmtpFrom = a.SmtpUser
		}

		subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

		if err := a.sendEmail(subject, emailBody, attachments); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}
	return nil
}

func (a *App) setupSentry() {
	if a.SentryDsn == "" {
		slog.Warn("[!] Sentry: DSN não especificado. Variável de ambiente/flag faltando: SENTRY_DSN")
		return
	}
	slog.Info("[*] Configurando sentry")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              a.SentryDsn,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		ReportError("Sentry initialization failed", err)
	}
}

func (a *App) setupBrowser(ctx context.Context) (*rod.Browser, error) {
	if a.BrowserCDP == "" {
		return nil, fmt.Errorf("[!] BROWSER_CDP não fornecido")
	}

	browser := rod.New().Context(ctx).ControlURL(a.BrowserCDP)

	if a.Verbose {
		browser = browser.Trace(true)
	}

	return browser, browser.Connect()
}

// loginToFusionSolar navigates to the FusionSolar login page, enters credentials,
// and dynamically handles common UI modals such as cookie policies and privacy agreements.
// It incorporates a retry mechanism bound by MaxLoginRetries to mitigate intermittent
// network instability or slow SPA rendering. It yields a logged-in rod.Page or an
// error if all retries are exhausted.
func (a *App) loginToFusionSolar(ctx context.Context, browser *rod.Browser) (*rod.Page, error) {
	for i := 0; i < a.MaxLoginRetries; i++ {
		slog.Info("[*] Login attempt", "attempt", i+1, "maxRetries", a.MaxLoginRetries)
		page := browser.MustPage("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")

		page.MustWaitLoad()
		if err := sleepContext(ctx, 5*time.Second); err != nil {
			return nil, err
		}

		page.MustElement("div#username input").MustInput(a.User)
		passwordInput := page.MustElement("div#password input")
		passwordInput.MustInput(a.Password)
		passwordInput.MustType(input.Enter)

		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return nil, err
		}

		// Handle cookie dialog
		cookieButtons := page.MustElements("i.cookiePolicy-icon")
		for _, button := range cookieButtons {
			slog.Info("[*] Fechando diálogo de cookie")
			button.MustClick()
		}

		// Handle privacy modal
		modals := page.MustElements("div.nco-privacy-confirm-modal")
		for _, modal := range modals {
			if has, _, _ := modal.Has("div.nco-privacy-content"); !has {
				continue
			}

			approveBtn, err := modal.Element("button.dpdesign-btn-primary")
			if err == nil {
				if err := sleepContext(ctx, 1*time.Second); err != nil {
					return nil, err
				}
				approveBtn.MustClick()
				if err := sleepContext(ctx, 10*time.Second); err != nil {
					return nil, err
				}
			}
		}

		// Check if logged in
		url := page.MustInfo().URL
		if !strings.Contains(url, "login") {
			return page, nil
		}
		slog.Info("[*] Reiniciando processo de login")
	}
	return nil, fmt.Errorf("failed to login after %d attempts", a.MaxLoginRetries)
}

type StationData struct {
	URL  string
	Name string
}

// getStations navigates to the logged-in homepage and scrapes DOM elements
// to compile a list of available solar stations. It features a built-in retry
// mechanism (up to 5 attempts) to account for the page's SPA architecture,
// which may delay rendering the stations list.
func (a *App) getStations(ctx context.Context, page *rod.Page) ([]StationData, error) {
	maxAttempts := 5
	attempts := 0
	var stationsData []StationData

	for len(stationsData) == 0 && attempts < maxAttempts {
		slog.Info("[*] Acessando homepage")
		page.MustNavigate("https://intl.fusionsolar.huawei.com")
		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return nil, err
		}

		slog.Info("[*] Tentando listar estações")
		stations, err := page.Elements("a.nco-home-list-text-ellipsis")
		if err != nil || len(stations) == 0 {
			attempts++
			slog.Info("[*] Zero estações encontradas", "attempt", attempts, "maxAttempts", maxAttempts)
			slog.Info("[*] URL atual", "url", page.MustInfo().URL)
			if attempts >= maxAttempts {
				err := fmt.Errorf("failed to find stations after %d attempts", maxAttempts)
				// We don't exit here anymore, return empty or error
				return nil, err
			}
		} else {
			for _, station := range stations {
				href := station.MustProperty("href").String()
				name := station.MustText()
				sData := StationData{
					URL:  href,
					Name: name,
				}
				slog.Info("[*] Estação encontrada", "name", sData.Name, "url", sData.URL)
				stationsData = append(stationsData, sData)
			}
		}
	}
	return stationsData, nil
}

type Attachment struct {
	Name    string
	Content []byte
}

// collectStationData iterates over the provided stations to extract power generation data
// and export the frontend charting canvas as a base64-encoded PNG screenshot. It fails
// gracefully per station if parsing errors occur (such as image decoding or float parsing),
// accumulating results rather than aborting the entire collection process.
func (a *App) collectStationData(ctx context.Context, page *rod.Page, stationsData []StationData) (string, []Attachment, error) {
	var emailBody strings.Builder
	emailBody.WriteString("Quantidade de energia produzida em cada base\n\n")
	var attachments []Attachment

	for _, station := range stationsData {
		slog.Info("[*] Coletando dados da estação", "station", station.Name)
		page.MustNavigate(station.URL)
		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return emailBody.String(), attachments, err
		}

		// Get canvas chart
		canvas := page.MustElement(".nco-single-energy-body canvas")
		res, err := canvas.Eval(`function() { return this.toDataURL('image/png') }`)
		if err != nil {
			ReportError("Error getting canvas data", err)
			continue
		}
		dataURL := res.Value.Str()
		b64 := strings.TrimPrefix(dataURL, "data:image/png;base64,")

		imgBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			ReportError("Error decoding base64 image", err, "station", station.Name)
			continue
		}
		attachments = append(attachments, Attachment{
			Name:    station.Name + ".png",
			Content: imgBytes,
		})

		// Get production amount
		valEl := page.MustElement("span.value")
		valText := valEl.MustText()
		valText = strings.ReplaceAll(valText, ",", ".")
		amountProduced, err := strconv.ParseFloat(valText, 64)
		if err != nil {
			ReportError("Error parsing production amount", err, "station", station.Name, "value", valText)
			fmt.Fprintf(&emailBody, "%s: Falha ao obter dados\n", station.Name)
		} else {
			fmt.Fprintf(&emailBody, "%s: %vkWh\n", station.Name, amountProduced)
			slog.Info("[*] Produzido hoje", "amount_kWh", amountProduced, "station", station.Name)
		}
	}

	fmt.Fprintf(&emailBody, "\nOs gráficos de geração estão em anexo.\n\n")
	fmt.Fprintf(&emailBody, "Dados obtidos em: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return emailBody.String(), attachments, nil
}

func (a *App) sendEmail(subject, body string, attachments []Attachment) error {
	slog.Info("[*] Enviando emails")

	m := gomail.NewMessage()
	m.SetHeader("From", a.SmtpFrom)
	m.SetHeader("To", a.SmtpDestinations...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	for _, att := range attachments {
		m.Attach(att.Name, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(att.Content)
			return err
		}))
	}

	host, portStr, err := net.SplitHostPort(a.SmtpServer)
	if err != nil {
		// a.SmtpServer might not have a port, try to append default
		host = a.SmtpServer
		portStr = "587"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid smtp port: %w", err)
	}

	d := gomail.NewDialer(host, port, a.SmtpUser, a.SmtpPasswd)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}
	return nil
}
