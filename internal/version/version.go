package version

// version is set via ldflags at build time (e.g. -X ...version.version=1.2.3).
// Defaults to "dev" for local builds.
var version = "dev"

// Get returns the build version string.
func Get() string {
	return version
}
