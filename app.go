package fusionsolar

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
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
	SmtpDestinations []string
	SentryDsn        string
	BrowserCDP       string
	Verbose          bool
	MaxLoginRetries  int
}

func sleepContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
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
	if len(a.SmtpDestinations) == 0 {
		missingEmailParams = append(missingEmailParams, "SMTP_DESTINATIONS")
	}

	emailEnabled := len(missingEmailParams) == 0
	if !emailEnabled {
		slog.Warn("[!] Funcionalidade de email desativada", "motivo", "Variáveis de ambiente/flags faltando", "missing", strings.Join(missingEmailParams, ", "))
	}

	scraper := &Scraper{
		User:            a.User,
		Password:        a.Password,
		BrowserCDP:      a.BrowserCDP,
		Verbose:         a.Verbose,
		MaxLoginRetries: a.MaxLoginRetries,
	}

	// Setup Browser
	browser, err := scraper.SetupBrowser(ctx)
	if err != nil {
		return err
	}
	defer browser.MustClose()

	// Login
	page, err := scraper.LoginToFusionSolar(ctx, browser)
	if err != nil {
		ReportError("Failed to login to FusionSolar", err)
		return err
	}

	// Get Stations
	stationsData, err := scraper.GetStations(ctx, page)
	if err != nil {
		ReportError("Failed to get stations", err)
		return err
	}

	// Collect Data
	emailBody, attachments, err := scraper.CollectStationData(ctx, page, stationsData)
	if err != nil {
		ReportError("Failed to collect station data", err)
		return err
	}

	fmt.Println(emailBody)

	if emailEnabled {
		if a.SmtpFrom == "" {
			a.SmtpFrom = a.SmtpUser
		}

		mailer := &Mailer{
			SmtpUser:         a.SmtpUser,
			SmtpFrom:         a.SmtpFrom,
			SmtpPasswd:       a.SmtpPasswd,
			SmtpServer:       a.SmtpServer,
			SmtpDestinations: a.SmtpDestinations,
		}

		subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

		mailer.SendEmail(
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
		ReportError("Sentry initialization failed", err)
	}
}
