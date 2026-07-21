package version

import (
	"runtime/debug"
	"strings"
	"unicode"
)

// version is set via ldflags at build time (e.g. -X ...version.version=1.2.3).
// Defaults to "dev" for local builds without injection.
var version = "dev"

// Get returns the build version string.
//
// Preference order:
//  1. ldflags-injected version (GoReleaser release builds)
//  2. stable module version from debug.ReadBuildInfo (go install @tag)
//  3. "dev-<shortsha>" when Go stamps vcs.revision (local/CI checkouts,
//     including module pseudo-versions that already embed a long commit id)
//  4. "dev" when nothing else is available
//
// Sentry Release (setupSentry) and --version both use this string; local
// binaries should not collapse to a generic "dev" when a commit is known.
func Get() string {
	return resolve(version, mainModuleInfo)
}

// mainModule holds optional build-info fields used when ldflags are unset.
type mainModule struct {
	Version  string
	Revision string
	OK       bool
}

func mainModuleInfo() mainModule {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return mainModule{}
	}
	m := mainModule{Version: info.Main.Version, OK: true}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			m.Revision = s.Value
			break
		}
	}
	return m
}

// shortRevision returns a compact commit id for display (GitHub-style 7 chars).
func shortRevision(rev string) string {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return ""
	}
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

// isPseudoVersion reports whether v looks like a Go module pseudo-version
// (timestamp + commit) rather than a release tag. Avoids depending on
// golang.org/x/mod just for this check.
func isPseudoVersion(v string) bool {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" || v == "(devel)" {
		return false
	}
	if strings.Contains(v, "+dirty") {
		return true
	}
	// Pseudo-versions embed a 14-digit UTC timestamp (yyyymmddhhmmss),
	// optionally after a "0." base-version segment.
	for _, part := range strings.Split(v, "-") {
		p := part
		if strings.HasPrefix(p, "0.") && len(p) > 2 {
			p = p[2:]
		}
		if len(p) == 14 && isAllDigits(p) {
			return true
		}
	}
	return false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// resolve picks a display version from ldflags injection and optional module metadata.
func resolve(injected string, main func() mainModule) string {
	if v := strings.TrimSpace(injected); v != "" && v != "dev" {
		return v
	}
	if main != nil {
		m := main()
		if m.OK {
			mv := strings.TrimSpace(m.Version)
			// Prefer a real release tag over VCS; skip (devel) and pseudo-versions.
			if mv != "" && mv != "(devel)" && !isPseudoVersion(mv) {
				// Repo tags are unprefixed (svu); strip a leading v so
				// go install @vX.Y.Z and ldflags both surface X.Y.Z.
				return strings.TrimPrefix(mv, "v")
			}
			if rev := shortRevision(m.Revision); rev != "" {
				// Unique id for --version and Sentry Release when ldflags are unset.
				return "dev-" + rev
			}
			// Pseudo-version without a separate revision stamp: keep stripped form.
			if mv != "" && mv != "(devel)" {
				return strings.TrimPrefix(mv, "v")
			}
		}
	}
	if v := strings.TrimSpace(injected); v != "" {
		return v
	}
	return "dev"
}
