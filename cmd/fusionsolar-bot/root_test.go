package main

import (
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
