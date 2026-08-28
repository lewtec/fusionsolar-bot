package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	fusionsolar "github.com/lewtec/fusionsolar-bot"
	appversion "github.com/lewtec/fusionsolar-bot/internal/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultMaxLoginRetries = 5

var (
	cfgFile          string
	user             string
	password         string
	smtpUser         string
	smtpFrom         string
	smtpPasswd       string
	smtpServer       string
	smtpDestinations string
	sentryDsn        string
	browserCDP       string
	verbose          bool
	timeout          time.Duration
	maxLoginRetries  int
)

var rootCmd = &cobra.Command{
	Use:     "fusionsolar-bot",
	Short:   "FusionSolar data collector",
	Long:    `A bot to collect data from FusionSolar and send reports via email using an external CDP browser.`,
	Version: appversion.Get(),
	Run: func(cmd *cobra.Command, args []string) {
		runBot()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fusionsolar.ReportError("Failed to execute root command", err)
		os.Exit(1)
	}
}

// mustBind panics on BindPFlag/BindEnv failure: those are programming errors
// (wrong flag name or key) and must not be ignored at process startup.
func mustBind(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fusionsolar-bot.yaml)")

	rootCmd.Flags().StringVar(&user, "user", "", "FusionSolar username")
	rootCmd.Flags().StringVar(&password, "password", "", "FusionSolar password")
	rootCmd.Flags().StringVar(&smtpUser, "smtp-user", "", "SMTP username")
	rootCmd.Flags().StringVar(&smtpFrom, "smtp-from", "", "SMTP From address (defaults to smtp-user if empty)")
	rootCmd.Flags().StringVar(&smtpPasswd, "smtp-passwd", "", "SMTP password")
	rootCmd.Flags().StringVar(&smtpServer, "smtp-server", "", "SMTP server (format: server:port)")
	rootCmd.Flags().StringVar(&smtpDestinations, "smtp-destinations", "", "Email recipients (space separated)")
	rootCmd.Flags().StringVar(&sentryDsn, "sentry-dsn", "", "Sentry DSN for error tracking")
	rootCmd.Flags().StringVar(&browserCDP, "browser-cdp", "", "CDP endpoint for the browser")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum total runtime before cancellation")
	rootCmd.Flags().IntVar(&maxLoginRetries, "max-login-retries", defaultMaxLoginRetries, "Maximum number of login retries")

	bindFlags()
}

func bindFlags() {
	mustBind(viper.BindPFlag("user", rootCmd.Flags().Lookup("user")))
	mustBind(viper.BindPFlag("password", rootCmd.Flags().Lookup("password")))
	mustBind(viper.BindPFlag("smtp-user", rootCmd.Flags().Lookup("smtp-user")))
	mustBind(viper.BindPFlag("smtp-from", rootCmd.Flags().Lookup("smtp-from")))
	mustBind(viper.BindPFlag("smtp-passwd", rootCmd.Flags().Lookup("smtp-passwd")))
	mustBind(viper.BindPFlag("smtp-server", rootCmd.Flags().Lookup("smtp-server")))
	mustBind(viper.BindPFlag("smtp-destinations", rootCmd.Flags().Lookup("smtp-destinations")))
	mustBind(viper.BindPFlag("sentry-dsn", rootCmd.Flags().Lookup("sentry-dsn")))
	mustBind(viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout")))
	mustBind(viper.BindPFlag("browser-cdp", rootCmd.Flags().Lookup("browser-cdp")))
	mustBind(viper.BindPFlag("max-login-retries", rootCmd.Flags().Lookup("max-login-retries")))
	mustBind(viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose")))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fusionsolar.ReportError("Failed to get user home directory", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".fusionsolar-bot")
	}

	// Map environment variables to match Python script's expectations
	mustBind(viper.BindEnv("user", "FUSIONSOLAR_USER"))
	mustBind(viper.BindEnv("password", "FUSIONSOLAR_PASSWORD"))
	mustBind(viper.BindEnv("smtp-user", "SMTP_USER"))
	mustBind(viper.BindEnv("smtp-from", "SMTP_FROM")) // If SMTP_FROM is not set, fallback to SMTP_USER in app logic
	mustBind(viper.BindEnv("smtp-passwd", "SMTP_PASSWD"))
	mustBind(viper.BindEnv("smtp-server", "SMTP_SERVER"))
	mustBind(viper.BindEnv("smtp-destinations", "SMTP_DESTINATIONS"))
	mustBind(viper.BindEnv("sentry-dsn", "SENTRY_DSN"))
	mustBind(viper.BindEnv("timeout", "TIMEOUT"))
	mustBind(viper.BindEnv("browser-cdp", "BROWSER_CDP"))
	mustBind(viper.BindEnv("max-login-retries", "MAX_LOGIN_RETRIES"))
	mustBind(viper.BindEnv("verbose", "VERBOSE"))

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "path", viper.ConfigFileUsed())
	}
}

func runBot() {
	runTimeout := viper.GetDuration("timeout")
	if runTimeout <= 0 {
		runTimeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	destinationsStr := viper.GetString("smtp-destinations")
	destinations := strings.FieldsFunc(destinationsStr, func(r rune) bool {
		return r == ',' || r == ' '
	})

	retries := viper.GetInt("max-login-retries")
	if retries <= 0 {
		// Empty/invalid env or config can surface as 0; keep the flag default.
		retries = defaultMaxLoginRetries
	}

	app := fusionsolar.App{
		User:             viper.GetString("user"),
		Password:         viper.GetString("password"),
		SmtpUser:         viper.GetString("smtp-user"),
		SmtpFrom:         viper.GetString("smtp-from"),
		SmtpPasswd:       viper.GetString("smtp-passwd"),
		SmtpServer:       viper.GetString("smtp-server"),
		SmtpDestinations: destinations,
		SentryDsn:        viper.GetString("sentry-dsn"),
		BrowserCDP:       viper.GetString("browser-cdp"),
		Verbose:          viper.GetBool("verbose"),
		MaxLoginRetries:  retries,
	}

	if err := app.Run(ctx); err != nil {
		fusionsolar.ReportError("Application failed", err)
		fusionsolar.FlushSentry(2 * time.Second)
		os.Exit(1)
	}
}
