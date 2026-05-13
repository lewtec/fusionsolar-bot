package fusionsolar

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

type Scraper struct {
	User            string
	Password        string
	BrowserCDP      string
	Verbose         bool
	MaxLoginRetries int
}

type StationData struct {
	URL  string
	Name string
}

func (s *Scraper) SetupBrowser(ctx context.Context) (*rod.Browser, error) {
	if s.BrowserCDP == "" {
		return nil, fmt.Errorf("[!] BROWSER_CDP não fornecido")
	}

	browser := rod.New().Context(ctx).ControlURL(s.BrowserCDP)

	if s.Verbose {
		browser = browser.Trace(true)
	}

	return browser, browser.Connect()
}

// LoginToFusionSolar navigates to the FusionSolar login page, enters credentials,
// and dynamically handles common UI modals such as cookie policies and privacy agreements.
// It incorporates a retry mechanism bound by MaxLoginRetries to mitigate intermittent
// network instability or slow SPA rendering. It yields a logged-in rod.Page or an
// error if all retries are exhausted.
func (s *Scraper) LoginToFusionSolar(ctx context.Context, browser *rod.Browser) (*rod.Page, error) {
	for i := 0; i < s.MaxLoginRetries; i++ {
		slog.Info("[*] Login attempt", "attempt", i+1, "maxRetries", s.MaxLoginRetries)
		page := browser.MustPage("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")

		page.MustWaitLoad()
		sleepContext(ctx, 5*time.Second)
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		page.MustElement("div#username input").MustInput(s.User)
		passwordInput := page.MustElement("div#password input")
		passwordInput.MustInput(s.Password)
		passwordInput.MustType(input.Enter)

		sleepContext(ctx, 10*time.Second)
		if err := ctx.Err(); err != nil {
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
				sleepContext(ctx, 1*time.Second)
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				approveBtn.MustClick()
				sleepContext(ctx, 10*time.Second)
				if err := ctx.Err(); err != nil {
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
	return nil, fmt.Errorf("failed to login after %d attempts", s.MaxLoginRetries)
}

// GetStations navigates to the logged-in homepage and scrapes DOM elements
// to compile a list of available solar stations. It features a built-in retry
// mechanism (up to 5 attempts) to account for the page's SPA architecture,
// which may delay rendering the stations list.
func (s *Scraper) GetStations(ctx context.Context, page *rod.Page) ([]StationData, error) {
	maxAttempts := 5
	attempts := 0
	var stationsData []StationData

	for len(stationsData) == 0 && attempts < maxAttempts {
		slog.Info("[*] Acessando homepage")
		page.MustNavigate("https://intl.fusionsolar.huawei.com")
		sleepContext(ctx, 10*time.Second)
		if err := ctx.Err(); err != nil {
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
				ReportError("[*] Sem estações, desistindo...", err)
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

// CollectStationData iterates over the provided stations to extract power generation data
// and export the frontend charting canvas as a base64-encoded PNG screenshot. It fails
// gracefully per station if parsing errors occur (such as image decoding or float parsing),
// accumulating results rather than aborting the entire collection process.
func (s *Scraper) CollectStationData(ctx context.Context, page *rod.Page, stationsData []StationData) (string, []Attachment, error) {
	var emailBody strings.Builder
	emailBody.WriteString("Quantidade de energia produzida em cada base\n\n")
	var attachments []Attachment

	for _, station := range stationsData {
		slog.Info("[*] Coletando dados da estação", "station", station.Name)
		page.MustNavigate(station.URL)
		sleepContext(ctx, 10*time.Second)
		if err := ctx.Err(); err != nil {
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
