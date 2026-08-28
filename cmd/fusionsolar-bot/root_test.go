package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTimeoutFlagDefaultsToTenMinutes(t *testing.T) {
	flag := rootCmd.Flags().Lookup("timeout")
	if flag == nil {
		t.Fatal("timeout flag is not defined")
	}

	got, err := time.ParseDuration(flag.DefValue)
	if err != nil {
		t.Fatalf("failed to parse timeout default %q: %v", flag.DefValue, err)
	}

	if got != 10*time.Minute {
		t.Fatalf("expected timeout default to be 10m, got %s", got)
	}
}

func TestVerboseFlagDefaultsToFalse(t *testing.T) {
	flag := rootCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Fatal("verbose flag is not defined")
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected verbose default false, got %q", flag.DefValue)
	}
}

func TestSmtpFromFlagDescribesFromAddress(t *testing.T) {
	flag := rootCmd.Flags().Lookup("smtp-from")
	if flag == nil {
		t.Fatal("smtp-from flag is not defined")
	}
	usage := strings.ToLower(flag.Usage)
	if usage == "" || strings.Contains(usage, "username") {
		t.Fatalf("smtp-from usage should describe From address, got %q", flag.Usage)
	}
}

func TestRootCmdVersionIsSet(t *testing.T) {
	if strings.TrimSpace(rootCmd.Version) == "" {
		t.Fatal("rootCmd.Version should be set via internal/version")
	}
	// Cobra owns --version when Version is set; custom flag must stay gone so -v remains verbose.
	if rootCmd.Flags().Lookup("version") != nil {
		t.Fatal("custom --version flag must not shadow cobra's Version handling")
	}
}

func TestMaxLoginRetriesFlagDefault(t *testing.T) {
	flag := rootCmd.Flags().Lookup("max-login-retries")
	if flag == nil {
		t.Fatal("max-login-retries flag is not defined")
	}
	if flag.DefValue != "5" {
		t.Fatalf("expected max-login-retries default 5, got %q", flag.DefValue)
	}
}

func TestMustBindPanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("mustBind should panic on non-nil error")
		}
	}()
	mustBind(errors.New("bind failed"))
}

func TestMustBindNoopOnNil(t *testing.T) {
	mustBind(nil)
}
