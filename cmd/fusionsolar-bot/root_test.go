package main

import (
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
