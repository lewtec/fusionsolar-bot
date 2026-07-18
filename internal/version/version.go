package version

import (
	"runtime/debug"
	"strings"
)

// version is set via ldflags at build time (e.g. -X ...version.version=1.2.3).
// Defaults to "dev" for local builds without injection.
var version = "dev"

// Get returns the build version string.
//
// Preference order:
//  1. ldflags-injected version (GoReleaser release builds)
//  2. module version from debug.ReadBuildInfo (go install of a tagged module)
//  3. "dev" for plain local checkouts
func Get() string {
	return resolve(version, mainModuleVersion)
}

func mainModuleVersion() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	return info.Main.Version, true
}

// resolve picks a display version from ldflags injection and optional module metadata.
// mainVersion returns the module version and whether build info was available.
func resolve(injected string, mainVersion func() (string, bool)) string {
	if v := strings.TrimSpace(injected); v != "" && v != "dev" {
		return v
	}
	if mainVersion != nil {
		if mv, ok := mainVersion(); ok {
			mv = strings.TrimSpace(mv)
			if mv != "" && mv != "(devel)" {
				// Repo tags are unprefixed (svu); strip a leading v so
				// go install @vX.Y.Z and ldflags both surface X.Y.Z.
				return strings.TrimPrefix(mv, "v")
			}
		}
	}
	if v := strings.TrimSpace(injected); v != "" {
		return v
	}
	return "dev"
}
