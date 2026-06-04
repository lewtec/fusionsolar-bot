package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	fusionsolar "fusionsolar-bot"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	version          bool
	timeout          time.Duration
	maxLoginRetries  int
)

// rootCmd represents the base command when called without any subcommands.
// It intercepts the version flag directly, otherwise delegating to runBot
// to execute the standard scraping pipeline.
var rootCmd = &cobra.Command{
	Use:   "fusionsolar-bot",
	Short: "FusionSolar data collector",
	Long:  `A bot to collect data from FusionSolar and send reports via email using an external CDP browser.`,
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			fmt.Println(fusionsolar.Version)
			os.Exit(0)
		}
		runBot()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fusionsolar-bot.yaml)")

	rootCmd.Flags().StringVar(&user, "user", "", "FusionSolar username")
	rootCmd.Flags().StringVar(&password, "password", "", "FusionSolar password")
	rootCmd.Flags().StringVar(&smtpUser, "smtp-user", "", "SMTP username")
	rootCmd.Flags().StringVar(&smtpFrom, "smtp-from", "", "SMTP username (default to smtp-user if not provided)")
	rootCmd.Flags().StringVar(&smtpPasswd, "smtp-passwd", "", "SMTP password")
	rootCmd.Flags().StringVar(&smtpServer, "smtp-server", "", "SMTP server (format: server:port)")
	rootCmd.Flags().StringVar(&smtpDestinations, "smtp-destinations", "", "Email recipients (space separated)")
	rootCmd.Flags().StringVar(&sentryDsn, "sentry-dsn", "", "Sentry DSN for error tracking")
	rootCmd.Flags().StringVar(&browserCDP, "browser-cdp", "", "CDP endpoint for the browser")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolVar(&version, "version", false, "Print version and exit")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum total runtime before cancellation")
	rootCmd.Flags().IntVar(&maxLoginRetries, "max-login-retries", 5, "Maximum number of login retries")

	bindFlags()
}

// bindFlags synchronizes Cobra command line flags with Viper's configuration map.
// This allows the application logic to read from Viper uniformly, regardless of
// whether a value was provided via a CLI flag, an environment variable, or a config file.
func bindFlags() {
	viper.BindPFlag("user", rootCmd.Flags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("smtp-user", rootCmd.Flags().Lookup("smtp-user"))
	viper.BindPFlag("smtp-from", rootCmd.Flags().Lookup("smtp-from"))
	viper.BindPFlag("smtp-passwd", rootCmd.Flags().Lookup("smtp-passwd"))
	viper.BindPFlag("smtp-server", rootCmd.Flags().Lookup("smtp-server"))
	viper.BindPFlag("smtp-destinations", rootCmd.Flags().Lookup("smtp-destinations"))
	viper.BindPFlag("sentry-dsn", rootCmd.Flags().Lookup("sentry-dsn"))
	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("browser-cdp", rootCmd.Flags().Lookup("browser-cdp"))
	viper.BindPFlag("max-login-retries", rootCmd.Flags().Lookup("max-login-retries"))
}

// initConfig reads in config file and ENV variables if set.
// It prioritizes explicit config files over $HOME defaults, and explicitly
// maps legacy/Python script-compatible environment variable names to internal viper keys.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".fusionsolar-bot")
	}

	// Map environment variables to match Python script's expectations
	viper.BindEnv("user", "FUSIONSOLAR_USER")
	viper.BindEnv("password", "FUSIONSOLAR_PASSWORD")
	viper.BindEnv("smtp-user", "SMTP_USER")
	viper.BindEnv("smtp-from", "SMTP_FROM") // If SMTP_FROM is not set, we'll handle fallback to SMTP_USER in logic
	viper.BindEnv("smtp-passwd", "SMTP_PASSWD")
	viper.BindEnv("smtp-server", "SMTP_SERVER")
	viper.BindEnv("smtp-destinations", "SMTP_DESTINATIONS")
	viper.BindEnv("sentry-dsn", "SENTRY_DSN")
	viper.BindEnv("timeout", "TIMEOUT")
	viper.BindEnv("browser-cdp", "BROWSER_CDP")
	viper.BindEnv("max-login-retries", "MAX_LOGIN_RETRIES")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// runBot bridges the Viper configuration context to the fusionsolar.App domain object.
// It parses raw string inputs (like comma-separated destination emails) into slices,
// enforces the global execution timeout, and invokes the central App.Run method.
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
		Verbose:          verbose,
		MaxLoginRetries:  viper.GetInt("max-login-retries"),
	}

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
