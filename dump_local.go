package fusionsolar

import (
	"github.com/go-rod/rod"
	"log/slog"
	"os"
	"path/filepath"
)

// dumpLocal writes subject/body/attachments to $OUT_DIR when OUT_DIR is set.
func dumpLocal(subject, body string, atts []Attachment) {
	dir := os.Getenv("OUT_DIR")
	if dir == "" {
		return
	}
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "subject.txt"), []byte(subject), 0o644)
	os.WriteFile(filepath.Join(dir, "body.txt"), []byte(body), 0o644)
	for _, a := range atts {
		os.WriteFile(filepath.Join(dir, filepath.Base(a.Name)), a.Content, 0o644)
	}
}

// singleStationFallback handles accounts that land directly on a single station overview.
// The station name comes from STATION_NAME, the page, or a generic default.
func singleStationFallback(page *rod.Page, out *[]StationData) bool {
	has, err := page.Eval(`() => !!document.querySelector(".nco-single-energy-body canvas")`)
	if err != nil || !has.Value.Bool() {
		return false
	}
	name := os.Getenv("STATION_NAME")
	if name == "" {
		r, _ := page.Eval(`() => { for (const s of [".nco-station-name",".station-name","[class*=stationName]","[class*=station-name]"]) { const e=document.querySelector(s); if (e && e.innerText.trim()) return e.innerText.trim(); } return ""; }`)
		if r != nil {
			name = r.Value.Str()
		}
	}
	if name == "" {
		name = "Station"
	}
	*out = append(*out, StationData{URL: page.MustInfo().URL, Name: name})
	slog.Info("[*] Estação única (fallback)", "name", name)
	return true
}
