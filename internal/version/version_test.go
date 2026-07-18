package version

import "testing"

func TestResolvePrefersLdflags(t *testing.T) {
	got := resolve("1.2.3", func() (string, bool) {
		t.Fatal("mainVersion should not be consulted when ldflags are set")
		return "", false
	})
	if got != "1.2.3" {
		t.Fatalf("resolve() = %q, want ldflags version", got)
	}
}

func TestResolveTrimsLdflags(t *testing.T) {
	got := resolve("  0.7.0\n", nil)
	if got != "0.7.0" {
		t.Fatalf("resolve() = %q, want trimmed ldflags", got)
	}
}

func TestResolveFallsBackToModuleVersion(t *testing.T) {
	got := resolve("dev", func() (string, bool) { return "v0.6.1", true })
	if got != "0.6.1" {
		t.Fatalf("resolve() = %q, want stripped module version", got)
	}

	got = resolve("dev", func() (string, bool) { return "0.6.1", true })
	if got != "0.6.1" {
		t.Fatalf("resolve() = %q, want bare module version", got)
	}
}

func TestResolveIgnoresDevelModuleVersion(t *testing.T) {
	got := resolve("dev", func() (string, bool) { return "(devel)", true })
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev for (devel)", got)
	}
}

func TestResolveEmptyInjectionUsesModuleOrDev(t *testing.T) {
	got := resolve("  ", func() (string, bool) { return "v1.0.0", true })
	if got != "1.0.0" {
		t.Fatalf("resolve() = %q, want module version when injection empty", got)
	}

	got = resolve("", func() (string, bool) { return "(devel)", true })
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev when nothing else available", got)
	}

	got = resolve("", nil)
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev with nil mainVersion", got)
	}
}

func TestGetUsesPackageVersionVar(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	version = "9.9.9"
	if got := Get(); got != "9.9.9" {
		t.Fatalf("Get() = %q, want 9.9.9", got)
	}
}
