package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile           string
	user              string
	password          string
	smtpUser          string
	smtpFrom          string
	smtpPasswd        string
	smtpServer        string
	smtpDestinations  string
	sentryDsn         string
	proxy             string
	headless          bool
	verbose           bool
)

var rootCmd = &cobra.Command{
	Use:   "fusionsolar-bot",
	Short: "FusionSolar data collector",
	Long:  `A bot to collect data from FusionSolar and send reports via email.`,
	Run: func(cmd *cobra.Command, args []string) {
		runBot()
	},
}

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
	rootCmd.Flags().StringVar(&proxy, "proxy", "", "Proxy server for Selenium")
	rootCmd.Flags().BoolVar(&headless, "headless", false, "Run Chrome in headless mode")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	bindFlags()
}

func bindFlags() {
	viper.BindPFlag("user", rootCmd.Flags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("smtp-user", rootCmd.Flags().Lookup("smtp-user"))
	viper.BindPFlag("smtp-from", rootCmd.Flags().Lookup("smtp-from"))
	viper.BindPFlag("smtp-passwd", rootCmd.Flags().Lookup("smtp-passwd"))
	viper.BindPFlag("smtp-server", rootCmd.Flags().Lookup("smtp-server"))
	viper.BindPFlag("smtp-destinations", rootCmd.Flags().Lookup("smtp-destinations"))
	viper.BindPFlag("sentry-dsn", rootCmd.Flags().Lookup("sentry-dsn"))
	viper.BindPFlag("proxy", rootCmd.Flags().Lookup("proxy"))
	// Headless and verbose are flags only, usually, but we can bind them if we want env vars support for them too.
	// The python script didn't seem to use env vars for headless/verbose, but `args = parser.parse_args()` was used.
	// The python script had `default=os.getenv(...)` for some args.
	// Cobra+Viper handles this by checking flag -> env -> config -> default.
}

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

	viper.AutomaticEnv()

	// Map environment variables to match Python script's expectations
	viper.BindEnv("user", "FUSIONSOLAR_USER")
	viper.BindEnv("password", "FUSIONSOLAR_PASSWORD")
	viper.BindEnv("smtp-user", "SMTP_USER")
	viper.BindEnv("smtp-from", "SMTP_FROM") // If SMTP_FROM is not set, we'll handle fallback to SMTP_USER in logic
	viper.BindEnv("smtp-passwd", "SMTP_PASSWD")
	viper.BindEnv("smtp-server", "SMTP_SERVER")
	viper.BindEnv("smtp-destinations", "SMTP_DESTINATIONS")
	viper.BindEnv("sentry-dsn", "SENTRY_DSN")
	viper.BindEnv("proxy", "SELENIUM_PROXY_SERVER")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runBot() {
	app := App{
		User:             viper.GetString("user"),
		Password:         viper.GetString("password"),
		SmtpUser:         viper.GetString("smtp-user"),
		SmtpFrom:         viper.GetString("smtp-from"),
		SmtpPasswd:       viper.GetString("smtp-passwd"),
		SmtpServer:       viper.GetString("smtp-server"),
		SmtpDestinations: viper.GetString("smtp-destinations"),
		SentryDsn:        viper.GetString("sentry-dsn"),
		Proxy:            viper.GetString("proxy"),
		Headless:         headless,
		Verbose:          verbose,
	}

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
