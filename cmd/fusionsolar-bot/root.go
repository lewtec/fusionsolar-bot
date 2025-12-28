package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
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
	// Values from Viper (Flag > Env > Default)
	userVal := viper.GetString("user")
	passVal := viper.GetString("password")
	sentryDSNVal := viper.GetString("sentry-dsn")

	// Setup Sentry
	setupSentry(sentryDSNVal)
	defer sentry.Flush(2 * time.Second)

	// Validate credentials
	if userVal == "" || passVal == "" {
		log.Fatal("[!] Usuário e senha fusionsolar não fornecidos")
	}

    // Check if email is enabled
    smtpUserVal := viper.GetString("smtp-user")
    smtpPasswdVal := viper.GetString("smtp-passwd")
    smtpServerVal := viper.GetString("smtp-server")
    smtpDestinationsVal := viper.GetString("smtp-destinations")

    emailEnabled := smtpUserVal != "" && smtpPasswdVal != "" && smtpServerVal != "" && smtpDestinationsVal != ""
    if !emailEnabled {
        log.Println("[!] Funcionalidade de email desativada")
    }

	// Setup Browser
	browser := setupBrowser(viper.GetString("proxy"), headless, verbose)
	defer browser.MustClose()

    // Login
    page := loginToFusionSolar(browser, userVal, passVal)

    // Get Stations
    stationsData := getStations(page)

    // Collect Data
    emailText, attachments := collectStationData(page, stationsData)

    fmt.Println(strings.Join(emailText, "\n"))

    if emailEnabled {
        smtpFromVal := viper.GetString("smtp-from")
        if smtpFromVal == "" {
            smtpFromVal = smtpUserVal
        }

        subject := fmt.Sprintf("Relatório do dia %s FusionSolar", time.Now().Format("2006-01-02"))

        sendEmail(
            smtpUserVal,
            smtpFromVal,
            smtpPasswdVal,
            smtpServerVal,
            smtpDestinationsVal,
            subject,
            strings.Join(emailText, "\n"),
            attachments,
        )
    }
}

func setupSentry(dsn string) {
	if dsn == "" {
		log.Println("[!] Sentry: DSN não especificado")
		return
	}
	log.Println("[*] Configurando sentry")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Printf("Sentry initialization failed: %v\n", err)
	}
}

func setupBrowser(proxyURL string, headless bool, verbose bool) *rod.Browser {
	l := launcher.New()

    if headless {
        l = l.Headless(true).
            Set("disable-gpu").
            Set("disable-dev-shm-usage").
            Set("no-sandbox")
    } else {
        l = l.Headless(false)
    }

	if proxyURL != "" {
		l = l.Proxy(proxyURL)
	}

    // Verbose logging in Rod typically involves setting a logger on the browser
    // Launcher args for verbose might differ from Selenium's
    // Python script uses --verbose for chromedriver logs and --log-level=DEBUG
    // For Rod, we can just use default launcher unless we need specific flags.

    // Setting window size
    l = l.Set("window-size", "1280,720")
    l = l.Set("remote-debugging-pipe")

	u := l.MustLaunch()

	browser := rod.New().ControlURL(u)

    if verbose {
        browser = browser.Trace(true)
    }

	return browser.MustConnect()
}

func loginToFusionSolar(browser *rod.Browser, username, password string) *rod.Page {
	for {
		log.Println("[*] Login")
		page := browser.MustPage("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")

		// Wait for load - Python script waits fixed 10s. Rod is smarter, but let's be safe.
		// page.MustWaitLoad() might be enough, but let's wait for selector
		page.MustWaitLoad()
		time.Sleep(5 * time.Second) // Mimicking the script's waiting behavior partially

		page.MustElement("div#username input").MustInput(username)
		passwordInput := page.MustElement("div#password input")
		passwordInput.MustInput(password)
		passwordInput.MustType(input.Enter)

		time.Sleep(10 * time.Second)

		// Handle cookie dialog
		// Rod doesn't throw if element not found when using Elements + loop
		cookieButtons := page.MustElements("i.cookiePolicy-icon")
		for _, button := range cookieButtons {
			log.Println("[*] Fechando diálogo de cookie")
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
		log.Println("[*] Reiniciando processo de login")
	}
}

type StationData struct {
    URL  string
    Name string
}

func getStations(page *rod.Page) []StationData {
    maxAttempts := 5
    attempts := 0
    var stationsData []StationData

    for len(stationsData) == 0 && attempts < maxAttempts {
        log.Println("[*] Acessando homepage")
        page.MustNavigate("https://intl.fusionsolar.huawei.com")
        time.Sleep(10 * time.Second)

        log.Println("[*] Tentando listar estações")
        // stations = driver.find_elements(By.CSS_SELECTOR, "a.nco-home-list-text-ellipsis")
        stations, err := page.Elements("a.nco-home-list-text-ellipsis")
        if err != nil || len(stations) == 0 {
             attempts++
             log.Printf("[*] Tentativa %d/%d: Zero estações encontradas\n", attempts, maxAttempts)
             log.Printf("[*] URL atual: %s\n", page.MustInfo().URL)
             if attempts >= maxAttempts {
                 log.Println("[*] Sem estações, desistindo...")
                 os.Exit(1)
             }
        } else {
            for _, station := range stations {
                href := station.MustAttribute("href")
                name := station.MustText()
                sData := StationData{
                    URL:  *href,
                    Name: name,
                }
                log.Println(sData)
                stationsData = append(stationsData, sData)
            }
        }
    }
    return stationsData
}

type Attachment struct {
    Name    string
    Content []byte
}

func collectStationData(page *rod.Page, stationsData []StationData) ([]string, []Attachment) {
    emailText := []string{"Quantidade de energia produzida em cada base", ""}
    var attachments []Attachment

    for _, station := range stationsData {
        log.Printf("[*] Coletando dados da estação \"%s\"\n", station.Name)
        page.MustNavigate(station.URL)
        time.Sleep(10 * time.Second)

        // Get canvas chart
        canvas := page.MustElement(".nco-single-energy-body canvas")
        // In python: return arguments[0].toDataURL('image/png').substring(21);
        // In Rod: Eval
        res, err := canvas.Eval(`function() { return this.toDataURL('image/png').substring(21) }`)
        if err != nil {
            log.Printf("Error getting canvas data: %v", err)
            continue
        }
        b64 := res.Value.String()

        imgBytes, _ := base64.StdEncoding.DecodeString(b64)
        attachments = append(attachments, Attachment{
            Name:    station.Name + ".png",
            Content: imgBytes,
        })

        // Get production amount
        // span.value
        valEl := page.MustElement("span.value")
        valText := valEl.MustText()
        valText = strings.ReplaceAll(valText, ",", ".")
        amountProduced, _ := strconv.ParseFloat(valText, 64)

        emailText = append(emailText, fmt.Sprintf("%s: %vkWh", station.Name, amountProduced))
        log.Printf("[*] Produzido hoje: %vkWh\n", amountProduced)
    }

    emailText = append(emailText,
        "",
        "Os gráficos de geração estão em anexo.",
        "",
        fmt.Sprintf("Dados obtidos em: %s", time.Now().Format("2006-01-02 15:04:05")),
    )

    return emailText, attachments
}

func sendEmail(user, from, password, server string, destinations string, subject, body string, attachments []Attachment) {
	log.Println("[*] Enviando emails")

	// Ensure port is present
	if !strings.Contains(server, ":") {
		server = server + ":587"
	}

	// Split server host:port
	host, _, _ := net.SplitHostPort(server)

	// Set up authentication information.
	auth := smtp.PlainAuth("", user, password, host)

	// Create the message
	boundary := "fusionsolar-boundary"
	header := make(map[string]string)
	header["From"] = from
	toAddrs := strings.Split(destinations, " ")
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

	err := smtp.SendMail(server, auth, from, toAddrs, []byte(msgBuilder.String()))
	if err != nil {
		log.Printf("Error sending email: %v", err)
	}
}
