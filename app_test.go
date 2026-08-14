package fusionsolar

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSleepContextStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	err := sleepContext(ctx, 30*time.Second)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("sleepContext should return immediately on cancellation, took %s", elapsed)
	}
	if err == nil || err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestReportPanicWrapsErrorValue(t *testing.T) {
	inner := errors.New("element not found")
	err := reportPanic(inner)
	if err == nil {
		t.Fatal("expected non-nil error from reportPanic")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected wrapped panic error, got %v", err)
	}
	if !strings.HasPrefix(err.Error(), "panic: ") {
		t.Fatalf("expected panic prefix, got %q", err.Error())
	}
}

func TestReportPanicWrapsNonErrorValue(t *testing.T) {
	err := reportPanic("boom")
	if err == nil {
		t.Fatal("expected non-nil error from reportPanic")
	}
	if got := err.Error(); got != "panic: boom" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestRunConvertsPanicToError(t *testing.T) {
	// Drive the same recover pattern Run uses: a panic mid-flight becomes a
	// returned error rather than a nil success after recovery.
	inner := errors.New("must-element failed")
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = reportPanic(r)
			}
		}()
		panic(inner)
	}()
	if err == nil {
		t.Fatal("expected panic to surface as error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsAgreeButtonText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "portuguese", in: "Concordar", want: true},
		{name: "english", in: "Agree", want: true},
		{name: "padded", in: "  concordar  ", want: true},
		{name: "cancel", in: "Cancelar", want: false},
		{name: "refuse", in: "Refuse", want: false},
		{name: "empty", in: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isAgreeButtonText(tt.in); got != tt.want {
				t.Errorf("isAgreeButtonText(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSentryOptionsTagsReleaseAndDisablesTraces(t *testing.T) {
	opts := sentryOptions("https://public@example.com/1", "0.8.0")
	if opts.Dsn != "https://public@example.com/1" {
		t.Fatalf("Dsn = %q", opts.Dsn)
	}
	if opts.Release != "0.8.0" {
		t.Fatalf("Release = %q, want binary version tag", opts.Release)
	}
	if opts.TracesSampleRate != 0 {
		t.Fatalf("TracesSampleRate = %v, want 0 for batch CLI", opts.TracesSampleRate)
	}
}
