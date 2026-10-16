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
	"github.com/go-rod/rod/lib/proto"
	"gopkg.in/gomail.v2"

	appversion "github.com/lewtec/fusionsolar-bot/internal/version"
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
// so callers do not treat unexpected panics as a successful Run.
// The result is marked already reported so the CLI does not capture it again.
func reportPanic(r any) error {
	sentry.CurrentHub().Recover(r)
	var err error
	if panicErr, ok := r.(error); ok {
		err = fmt.Errorf("panic: %w", panicErr)
	} else {
		err = fmt.Errorf("panic: %v", r)
	}
	return alreadyReportedError{err}
}

func (a *App) Run(ctx context.Context) (err error) {
	// Setup Sentry
	a.setupSentry()
	defer sentry.Flush(2 * time.Second)
	// Browser helpers return errors. This only covers an unexpected panic so the
	// CLI still exits non-zero instead of looking successful after recovery.
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
		return fmt.Errorf("login to FusionSolar: %w", err)
	}

	// Get Stations
	stationsData, err := a.getStations(ctx, page)
	if err != nil {
		return fmt.Errorf("get stations: %w", err)
	}

	// Collect Data
	emailBody, attachments, err := a.collectStationData(ctx, page, stationsData)
	if err != nil {
		return fmt.Errorf("collect station data: %w", err)
	}

	fmt.Println(emailBody)

	if emailEnabled {
		if a.SmtpFrom == "" {
			a.SmtpFrom = a.SmtpUser
		}

		subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

		if err := a.sendEmail(subject, emailBody, attachments); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
	}
	return nil
}

// sentryOptions builds ClientOptions for this batch CLI.
// Release tags events with the binary version; TracesSampleRate stays 0 because
// short-lived scrape/report runs do not benefit from performance transactions.
func sentryOptions(dsn, release string) sentry.ClientOptions {
	return sentry.ClientOptions{
		Dsn:              dsn,
		Release:          release,
		TracesSampleRate: 0,
	}
}

func (a *App) setupSentry() {
	if a.SentryDsn == "" {
		slog.Warn("[!] Sentry: DSN não especificado. Variável de ambiente/flag faltando: SENTRY_DSN")
		return
	}
	slog.Info("[*] Configurando sentry")
	err := sentry.Init(sentryOptions(a.SentryDsn, appversion.Get()))
	if err != nil {
		ReportError("Sentry initialization failed", err)
	}
}

func (a *App) setupBrowser(ctx context.Context) (*rod.Browser, error) {
	if a.BrowserCDP == "" {
		return nil, fmt.Errorf("[!] BROWSER_CDP não fornecido (defina BROWSER_CDP ou --browser-cdp)")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("setup browser: %w", err)
	}

	browser := rod.New().Context(ctx).ControlURL(a.BrowserCDP)

	if a.Verbose {
		browser = browser.Trace(true)
	}

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("BROWSER_CDP inacessível: %w", err)
	}
	return browser, nil
}

// loginToFusionSolar navigates to the FusionSolar login page, enters credentials,
// and dynamically handles common UI modals such as cookie policies and privacy agreements.
// It incorporates a retry mechanism bound by MaxLoginRetries to mitigate intermittent
// network instability or slow SPA rendering. It yields a logged-in rod.Page or an
// error if all retries are exhausted.
func (a *App) loginToFusionSolar(ctx context.Context, browser *rod.Browser) (*rod.Page, error) {
	const loginURL = "https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN"
	var lastErr error
	for i := 0; i < a.MaxLoginRetries; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
		slog.Info("[*] Login attempt", "attempt", i+1, "maxRetries", a.MaxLoginRetries)
		page, err := browser.Page(proto.TargetCreateTarget{URL: loginURL})
		if err != nil {
			lastErr = err
			slog.Info("[*] Falha ao abrir página de login", "attempt", i+1, "error", err)
			if !isRetryableBrowserErr(err) {
				return nil, fmt.Errorf("open login page: %w", err)
			}
			continue
		}

		if err := waitPageLoad(ctx, page); err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("wait login page: %w", err)
			}
			continue
		}
		if err := sleepContext(ctx, 5*time.Second); err != nil {
			return nil, err
		}

		userEl, err := waitElement(ctx, page, "div#username input")
		if err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("login username field: %w", err)
			}
			continue
		}
		if err := inputText(ctx, userEl, a.User); err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("fill username: %w", err)
			}
			continue
		}
		passwordInput, err := waitElement(ctx, page, "div#password input")
		if err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("login password field: %w", err)
			}
			continue
		}
		if err := inputText(ctx, passwordInput, a.Password); err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("fill password: %w", err)
			}
			continue
		}
		if err := pressKey(ctx, passwordInput, input.Enter); err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("submit login: %w", err)
			}
			continue
		}

		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return nil, err
		}

		if err := a.dismissBlockingModals(ctx, page); err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("dismiss blocking modals: %w", err)
			}
			continue
		}

		url, err := pageURL(ctx, page)
		if err != nil {
			lastErr = err
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("read page url: %w", err)
			}
			continue
		}
		if !strings.Contains(url, "login") {
			return page, nil
		}
		slog.Info("[*] Reiniciando processo de login")
	}
	if lastErr != nil {
		return nil, fmt.Errorf("failed to login after %d attempts: %w", a.MaxLoginRetries, lastErr)
	}
	return nil, fmt.Errorf("failed to login after %d attempts", a.MaxLoginRetries)
}

// stopBrowserRetry reports whether another attempt would hide a finished run
// or a non-transient browser failure.
func stopBrowserRetry(ctx context.Context, err error) bool {
	return ctx.Err() != nil || !isRetryableBrowserErr(err)
}

// dismissBlockingModals closes the cookie banner and the post-login
// "Aviso de atualização" / privacy modal. Concordar is a default solid
// button, not dpdesign-btn-primary; match by label. The dialog can appear
// on #/home/list after login, not only on the login page.
func (a *App) dismissBlockingModals(ctx context.Context, page *rod.Page) error {
	cookies, err := pageElements(ctx, page, "i.cookiePolicy-icon")
	if err != nil {
		return err
	}
	for _, button := range cookies {
		slog.Info("[*] Fechando diálogo de cookie")
		if err := elementClick(ctx, button); err != nil {
			return err
		}
	}

	for _, sel := range []string{"div.nco-privacy-confirm-modal", ".privacy-confirm-popup"} {
		modals, err := pageElements(ctx, page, sel)
		if err != nil {
			return err
		}
		for _, modal := range modals {
			if err := acceptPrivacyModal(ctx, modal); err != nil {
				return err
			}
		}
	}
	return nil
}

func acceptPrivacyModal(ctx context.Context, modal *rod.Element) error {
	buttons, err := elementElements(ctx, modal, "div.ant-modal-footer button")
	if err != nil {
		return err
	}
	var agree *rod.Element
	for _, btn := range buttons {
		text, err := elementText(ctx, btn)
		if err != nil {
			return err
		}
		if isAgreeButtonText(text) {
			agree = btn
			break
		}
	}
	if agree == nil {
		// Optional button: Elements returns immediately when it is absent.
		fallbacks, err := elementElements(ctx, modal, "button.dpdesign-btn-primary")
		if err != nil {
			return err
		}
		if len(fallbacks) > 0 {
			agree = fallbacks[0]
		}
	}
	if agree == nil {
		return nil
	}

	slog.Info("[*] Aceitando aviso de atualização / privacidade")
	if err := sleepContext(ctx, time.Second); err != nil {
		return err
	}
	if err := elementClick(ctx, agree); err != nil {
		return err
	}
	return sleepContext(ctx, 3*time.Second)
}

func isAgreeButtonText(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "concordar", "agree", "aceptar", "aceitar", "同意":
		return true
	default:
		return false
	}
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
	const maxAttempts = 5
	const homeURL = "https://intl.fusionsolar.huawei.com"
	var stationsData []StationData
	var lastErr error

	for attempt := 1; attempt <= maxAttempts && len(stationsData) == 0; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("list stations: %w", err)
		}
		slog.Info("[*] Acessando homepage")
		if err := navigatePage(ctx, page, homeURL); err != nil {
			lastErr = err
			slog.Info("[*] Falha ao abrir homepage", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("open homepage: %w", err)
			}
			continue
		}
		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return nil, err
		}
		if err := a.dismissBlockingModals(ctx, page); err != nil {
			lastErr = err
			slog.Info("[*] Falha ao fechar modais", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("dismiss blocking modals: %w", err)
			}
			continue
		}

		slog.Info("[*] Tentando listar estações")
		stations, err := pageElements(ctx, page, "a.nco-home-list-text-ellipsis")
		if err != nil {
			lastErr = err
			slog.Info("[*] Falha ao listar estações", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
			if stopBrowserRetry(ctx, err) {
				return nil, fmt.Errorf("list stations: %w", err)
			}
			continue
		}
		if len(stations) == 0 {
			slog.Info("[*] Zero estações encontradas", "attempt", attempt, "maxAttempts", maxAttempts)
			if url, urlErr := pageURL(ctx, page); urlErr != nil {
				if ctx.Err() != nil {
					return nil, fmt.Errorf("list stations: %w", ctx.Err())
				}
				slog.Info("[*] URL atual indisponível", "error", urlErr)
			} else {
				slog.Info("[*] URL atual", "url", url)
			}
			continue
		}

		readErr := readStationLinks(ctx, stations, &stationsData)
		if readErr != nil {
			stationsData = nil
			lastErr = readErr
			slog.Info("[*] Falha ao ler estações", "attempt", attempt, "maxAttempts", maxAttempts, "error", readErr)
			if stopBrowserRetry(ctx, readErr) {
				return nil, fmt.Errorf("read station: %w", readErr)
			}
		}
	}
	if len(stationsData) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("find stations after %d attempts: %w", maxAttempts, lastErr)
		}
		return nil, fmt.Errorf("find stations after %d attempts", maxAttempts)
	}
	return stationsData, nil
}

func readStationLinks(ctx context.Context, stations rod.Elements, dest *[]StationData) error {
	for _, station := range stations {
		href, err := elementProperty(ctx, station, "href")
		if err != nil {
			return err
		}
		name, err := elementText(ctx, station)
		if err != nil {
			return err
		}
		sData := StationData{URL: href, Name: name}
		slog.Info("[*] Estação encontrada", "name", sData.Name, "url", sData.URL)
		*dest = append(*dest, sData)
	}
	return nil
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
		if err := navigatePage(ctx, page, station.URL); err != nil {
			return emailBody.String(), attachments, fmt.Errorf("open station: %w", err)
		}
		if err := sleepContext(ctx, 10*time.Second); err != nil {
			return emailBody.String(), attachments, err
		}
		if err := a.dismissBlockingModals(ctx, page); err != nil {
			return emailBody.String(), attachments, err
		}

		canvas, err := waitElement(ctx, page, ".nco-single-energy-body canvas")
		if err != nil {
			return emailBody.String(), attachments, fmt.Errorf("station canvas: %w", err)
		}
		dataURL, err := readCanvasDataURL(ctx, canvas)
		if err != nil {
			return emailBody.String(), attachments, fmt.Errorf("read canvas: %w", err)
		}
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

		valEl, err := waitElement(ctx, page, "span.value")
		if err != nil {
			return emailBody.String(), attachments, fmt.Errorf("read production amount: %w", err)
		}
		valText, err := elementText(ctx, valEl)
		if err != nil {
			return emailBody.String(), attachments, fmt.Errorf("read production amount: %w", err)
		}
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
