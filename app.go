package fusionsolar

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-rod/rod"
	"os"
	"os/exec"
	"path/filepath"

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
	page := a.loginToFusionSolar(browser)

	// Get Stations
	stationsData, err := a.getStations(page)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	// Collect Data
	emailText, attachments := a.collectStationData(page, stationsData)

	fmt.Println(strings.Join(emailText, "\n"))

	if emailEnabled {
		if a.SmtpFrom == "" {
			a.SmtpFrom = a.SmtpUser
		}

		subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

		a.sendEmail(
			subject,
			strings.Join(emailText, "\n"),
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
	l := launcher.New()

	chromiumPath := os.Getenv("CHROMIUM")
	if chromiumPath == "" {
		chromiumPath = "chromium"
	}

	if filepath.IsAbs(chromiumPath) {
		l = l.Bin(chromiumPath)
	} else {
		path, err := exec.LookPath(chromiumPath)
		if err == nil {
			l = l.Bin(path)
		} else {
			slog.Debug("Chromium executable not found in PATH, falling back to default launcher logic", "path", chromiumPath)
		}
	}

	if a.Headless {
		l = l.Headless(true).
			Set("disable-gpu").
			Set("disable-dev-shm-usage").
			Set("no-sandbox")
	} else {
		l = l.Headless(false)
	}

	if a.Proxy != "" {
		l = l.Proxy(a.Proxy)
	}

	l = l.Set("window-size", "1280,720")

	u := l.MustLaunch()

	browser := rod.New().ControlURL(u)

	if a.Verbose {
		browser = browser.Trace(true)
	}

	return browser.MustConnect()
}

func (a *App) loginToFusionSolar(browser *rod.Browser) *rod.Page {
	for {
		slog.Info("[*] Login")
		page := browser.MustPage("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")

		page.MustWaitLoad()
		time.Sleep(5 * time.Second)

		page.MustElement("div#username input").MustInput(a.User)
		passwordInput := page.MustElement("div#password input")
		passwordInput.MustInput(a.Password)
		passwordInput.MustType(input.Enter)

		time.Sleep(10 * time.Second)

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
				time.Sleep(1 * time.Second)
				approveBtn.MustClick()
				time.Sleep(10 * time.Second)
			}
		}

		// Check if logged in
		url := page.MustInfo().URL
		if !strings.Contains(url, "login") {
			return page
		}
		slog.Info("[*] Reiniciando processo de login")
	}
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
		time.Sleep(10 * time.Second)

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

func (a *App) collectStationData(page *rod.Page, stationsData []StationData) ([]string, []Attachment) {
	emailText := []string{"Quantidade de energia produzida em cada base", ""}
	var attachments []Attachment

	for _, station := range stationsData {
		slog.Info(fmt.Sprintf("[*] Coletando dados da estação \"%s\"", station.Name))
		page.MustNavigate(station.URL)
		time.Sleep(10 * time.Second)

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
		amountProduced, _ := strconv.ParseFloat(valText, 64)

		emailText = append(emailText, fmt.Sprintf("%s: %vkWh", station.Name, amountProduced))
		slog.Info(fmt.Sprintf("[*] Produzido hoje: %vkWh", amountProduced))
	}

	emailText = append(emailText,
		"",
		"Os gráficos de geração estão em anexo.",
		"",
		fmt.Sprintf("Dados obtidos em: %s", time.Now().Format("2006-01-02 15:04:05")),
	)

	return emailText, attachments
}

func (a *App) sendEmail(subject, body string, attachments []Attachment) {
	slog.Info("[*] Enviando emails")
	server := a.SmtpServer

	// Ensure port is present
	if !strings.Contains(server, ":") {
		server = server + ":587"
	}

	// Split server host:port
	host, _, _ := net.SplitHostPort(server)

	// Set up authentication information.
	auth := smtp.PlainAuth("", a.SmtpUser, a.SmtpPasswd, host)

	// Create the message
	boundary := "fusionsolar-boundary"
	header := make(map[string]string)
	header["From"] = a.SmtpFrom
	toAddrs := strings.Split(a.SmtpDestinations, " ")
	header["To"] = strings.Join(toAddrs, ", ")
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "multipart/mixed; boundary=" + boundary

	var msgBuilder strings.Builder
	for k, v := range header {
		msgBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msgBuilder.WriteString("\r\n")

	// Body
	msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(body)
	msgBuilder.WriteString("\r\n")

	// Attachments
	for _, att := range attachments {
		msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msgBuilder.WriteString("Content-Type: image/png\r\n")
		msgBuilder.WriteString("Content-Transfer-Encoding: base64\r\n")
		msgBuilder.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Name))
		msgBuilder.WriteString("\r\n")

		b64 := base64.StdEncoding.EncodeToString(att.Content)
		// Split lines at 76 chars
		for i := 0; i < len(b64); i += 76 {
			end := i + 76
			if end > len(b64) {
				end = len(b64)
			}
			msgBuilder.WriteString(b64[i:end] + "\r\n")
		}
		msgBuilder.WriteString("\r\n")
	}
	msgBuilder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	err := smtp.SendMail(server, auth, a.SmtpFrom, toAddrs, []byte(msgBuilder.String()))
	if err != nil {
		slog.Error("Error sending email", "error", err)
	}
}
