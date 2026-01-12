package fusionsolar

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"gopkg.in/gomail.v2"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-rod/rod"

	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
)

//go:embed version.txt
var Version string

type App struct {
	User             string
	Password         string
	SmtpUser         string
	SmtpFrom         string
	SmtpPasswd       string
	SmtpServer       string
	SmtpDestinations string
	SentryDsn        string
	Proxy            string
	Headless         bool
	Verbose          bool
	MaxLoginRetries  int
}

func (a *App) Run(ctx context.Context) error {
	// Setup Sentry
	a.setupSentry()
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

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
	if a.SmtpDestinations == "" {
		missingEmailParams = append(missingEmailParams, "SMTP_DESTINATIONS")
	}

	emailEnabled := len(missingEmailParams) == 0
	if !emailEnabled {
		slog.Warn(fmt.Sprintf("[!] Funcionalidade de email desativada. Motivo: Variáveis de ambiente/flags faltando: %s", strings.Join(missingEmailParams, ", ")))
	}

	// Setup Browser
	browser := a.setupBrowser().Context(ctx)
	defer browser.MustClose()

	// Login
	page, err := a.loginToFusionSolar(browser)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	// Get Stations
	stationsData, err := a.getStations(page)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	// Collect Data
	emailBody, attachments := a.collectStationData(page, stationsData)

	fmt.Println(emailBody)

	if emailEnabled {
		if a.SmtpFrom == "" {
			a.SmtpFrom = a.SmtpUser
		}

		subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

		a.sendEmail(
			subject,
			emailBody,
			attachments,
		)
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
		slog.Error("Sentry initialization failed", "error", err)
	}
}

func (a *App) setupBrowser() *rod.Browser {
	browser := NewBrowser(a.Headless, a.Proxy)

	if a.Verbose {
		browser = browser.Trace(true)
	}

	return browser.MustConnect()
}

func NewBrowser(headless bool, proxy string) *rod.Browser {
	l := launcher.New()

	if chromiumPath := os.Getenv("CHROMIUM"); chromiumPath != "" {
		l.Bin(chromiumPath)
	}

	if headless {
		l = l.Headless(true).
			Set("disable-gpu").
			Set("disable-dev-shm-usage").
			Set("no-sandbox")
	} else {
		l = l.Headless(false)
	}

	if proxy != "" {
		l = l.Proxy(proxy)
	}

	l = l.Set("window-size", "1280,720")

	u := l.MustLaunch()

	return rod.New().ControlURL(u)
}

func (a *App) loginToFusionSolar(browser *rod.Browser) (*rod.Page, error) {
	for i := 0; i < a.MaxLoginRetries; i++ {
		slog.Info(fmt.Sprintf("[*] Login attempt %d/%d", i+1, a.MaxLoginRetries))
		page := browser.MustPage("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")

		page.MustWaitLoad()
		page.MustElement("div#username input").MustWaitVisible().MustInput(a.User)
		passwordInput := page.MustElement("div#password input").MustWaitVisible()
		passwordInput.MustInput(a.Password)
		wait := page.MustWaitNavigation()
		passwordInput.MustType(input.Enter)
		wait()

		// Handle cookie dialog
		cookieButtons, err := page.Elements("i.cookiePolicy-icon")
		if err != nil {
			slog.Debug("Cookie dialog not found, skipping...")
		}
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
				approveBtn.MustClick()
				modal.MustWaitInvisible()
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

func (a *App) getStations(page *rod.Page) ([]StationData, error) {
	maxAttempts := 5
	attempts := 0
	var stationsData []StationData

	for len(stationsData) == 0 && attempts < maxAttempts {
		slog.Info("[*] Acessando homepage")
		page.MustNavigate("https://intl.fusionsolar.huawei.com")
		page.MustWaitLoad()
		// Wait for the station list to appear, which signals the page is ready.
		page.MustElementR("div.nco-home-list", "Estação")

		slog.Info("[*] Tentando listar estações")
		stations, err := page.Elements("a.nco-home-list-text-ellipsis")
		if err != nil || len(stations) == 0 {
			attempts++
			slog.Info(fmt.Sprintf("[*] Tentativa %d/%d: Zero estações encontradas", attempts, maxAttempts))
			slog.Info(fmt.Sprintf("[*] URL atual: %s", page.MustInfo().URL))
			if attempts >= maxAttempts {
				slog.Error("[*] Sem estações, desistindo...")
				// We don't exit here anymore, return empty or error
				return nil, fmt.Errorf("failed to find stations after %d attempts", maxAttempts)
			}
		} else {
			for _, station := range stations {
				href := station.MustProperty("href").String()
				name := station.MustText()
				sData := StationData{
					URL:  href,
					Name: name,
				}
				slog.Info(fmt.Sprintf("%v", sData))
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

func (a *App) collectStationData(page *rod.Page, stationsData []StationData) (string, []Attachment) {
	var emailBody strings.Builder
	emailBody.WriteString("Quantidade de energia produzida em cada base\n\n")
	var attachments []Attachment

	for _, station := range stationsData {
		slog.Info(fmt.Sprintf("[*] Coletando dados da estação \"%s\"", station.Name))
		page.MustNavigate(station.URL)
		page.MustElement(".nco-single-energy-body canvas").MustWaitVisible()

		// Get canvas chart
		canvas := page.MustElement(".nco-single-energy-body canvas")
		res, err := canvas.Eval(`function() { return this.toDataURL('image/png') }`)
		if err != nil {
			slog.Error("Error getting canvas data", "error", err)
			continue
		}
		dataURL := res.Value.Str()
		b64 := strings.TrimPrefix(dataURL, "data:image/png;base64,")

		imgBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			slog.Error(fmt.Sprintf("Error decoding base64 image for station %s", station.Name), "error", err)
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
			slog.Error(fmt.Sprintf("Error parsing production amount for station %s", station.Name), "error", err, "value", valText)
			fmt.Fprintf(&emailBody, "%s: Falha ao obter dados\n", station.Name)
		} else {
			fmt.Fprintf(&emailBody, "%s: %vkWh\n", station.Name, amountProduced)
			slog.Info(fmt.Sprintf("[*] Produzido hoje: %vkWh", amountProduced))
		}
	}

	fmt.Fprintf(&emailBody, "\nOs gráficos de geração estão em anexo.\n\n")
	fmt.Fprintf(&emailBody, "Dados obtidos em: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return emailBody.String(), attachments
}

func (a *App) sendEmail(subject, body string, attachments []Attachment) {
	slog.Info("[*] Enviando emails")

	m := gomail.NewMessage()
	m.SetHeader("From", a.SmtpFrom)
	m.SetHeader("To", strings.Split(a.SmtpDestinations, " ")...)
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
		slog.Error("invalid smtp port", "error", err)
		return
	}

	d := gomail.NewDialer(host, port, a.SmtpUser, a.SmtpPasswd)

	if err := d.DialAndSend(m); err != nil {
		slog.Error("Error sending email", "error", err)
	}
}
