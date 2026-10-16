package fusionsolar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

var (
	errElementNotFound   = errors.New("element not found")
	errMustElementFailed = errors.New("must-element failed")
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
	err := reportPanic(errElementNotFound)
	if err == nil {
		t.Fatal("expected non-nil error from reportPanic")
	}
	if !errors.Is(err, errElementNotFound) {
		t.Fatalf("expected wrapped panic error, got %v", err)
	}
	if !strings.HasPrefix(err.Error(), "panic: ") {
		t.Fatalf("expected panic prefix, got %q", err.Error())
	}
	if !alreadyReported(err) {
		t.Fatal("expected recovered panic to be marked already reported")
	}
}

func TestAlreadyReportedSurvivesWrap(t *testing.T) {
	err := fmt.Errorf("get stations: %w", reportPanic(io.EOF))
	if !alreadyReported(err) {
		t.Fatal("expected wrapped panic to stay marked already reported")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected wrapped EOF, got %v", err)
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
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = reportPanic(r)
			}
		}()
		panic(errMustElementFailed)
	}()
	if err == nil {
		t.Fatal("expected panic to surface as error, got nil")
	}
	if !errors.Is(err, errMustElementFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopBrowserRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	if stopBrowserRetry(ctx, io.EOF) {
		t.Fatal("EOF should stay retryable while the run is active")
	}
	cancel()
	if !stopBrowserRetry(ctx, io.EOF) {
		t.Fatal("a finished run should stop retries")
	}
	if !stopBrowserRetry(t.Context(), errElementNotFound) {
		t.Fatal("non-transient errors should stop retries")
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
