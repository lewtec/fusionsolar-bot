package version

import "testing"

func TestResolvePrefersLdflags(t *testing.T) {
	got := resolve("1.2.3", func() mainModule {
		t.Fatal("mainModule should not be consulted when ldflags are set")
		return mainModule{}
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
	got := resolve("dev", func() mainModule {
		return mainModule{Version: "v0.6.1", OK: true}
	})
	if got != "0.6.1" {
		t.Fatalf("resolve() = %q, want stripped module version", got)
	}

	got = resolve("dev", func() mainModule {
		return mainModule{Version: "0.6.1", OK: true}
	})
	if got != "0.6.1" {
		t.Fatalf("resolve() = %q, want bare module version", got)
	}
}

func TestResolveDevelUsesShortRevision(t *testing.T) {
	got := resolve("dev", func() mainModule {
		return mainModule{
			Version:  "(devel)",
			Revision: "abcdef0123456789",
			OK:       true,
		}
	})
	if got != "dev-abcdef0" {
		t.Fatalf("resolve() = %q, want dev-abcdef0 from VCS revision", got)
	}
}

func TestResolvePseudoVersionUsesShortRevision(t *testing.T) {
	got := resolve("dev", func() mainModule {
		return mainModule{
			Version:  "v0.1.1-0.20260721222329-88501666e7f6+dirty",
			Revision: "88501666e7f63553f7ca953b937b8531361a4a86",
			OK:       true,
		}
	})
	if got != "dev-8850166" {
		t.Fatalf("resolve() = %q, want short VCS id over pseudo-version", got)
	}
}

func TestResolveDevelWithoutRevisionStaysDev(t *testing.T) {
	got := resolve("dev", func() mainModule {
		return mainModule{Version: "(devel)", OK: true}
	})
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev for (devel) without revision", got)
	}
}

func TestResolveEmptyInjectionUsesModuleOrDev(t *testing.T) {
	got := resolve("  ", func() mainModule {
		return mainModule{Version: "v1.0.0", OK: true}
	})
	if got != "1.0.0" {
		t.Fatalf("resolve() = %q, want module version when injection empty", got)
	}

	got = resolve("", func() mainModule {
		return mainModule{Version: "(devel)", OK: true}
	})
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev when nothing else available", got)
	}

	got = resolve("", nil)
	if got != "dev" {
		t.Fatalf("resolve() = %q, want dev with nil main", got)
	}
}

func TestResolveModuleVersionBeatsRevision(t *testing.T) {
	got := resolve("dev", func() mainModule {
		return mainModule{
			Version:  "v0.9.0",
			Revision: "deadbeefcafebabe",
			OK:       true,
		}
	})
	if got != "0.9.0" {
		t.Fatalf("resolve() = %q, want module version over revision", got)
	}
}

func TestIsPseudoVersion(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0.6.1", false},
		{"v0.6.1", false},
		{"(devel)", false},
		{"", false},
		{"v0.1.1-0.20260721222329-88501666e7f6+dirty", true},
		{"v0.0.0-20260721222329-88501666e7f6", true},
		{"0.1.1-0.20260721222329-88501666e7f6", true},
	}
	for _, tc := range cases {
		if got := isPseudoVersion(tc.in); got != tc.want {
			t.Fatalf("isPseudoVersion(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestShortRevision(t *testing.T) {
	if got := shortRevision("abcdefghij"); got != "abcdefg" {
		t.Fatalf("shortRevision long = %q", got)
	}
	if got := shortRevision("abc"); got != "abc" {
		t.Fatalf("shortRevision short = %q", got)
	}
	if got := shortRevision("  "); got != "" {
		t.Fatalf("shortRevision blank = %q", got)
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
