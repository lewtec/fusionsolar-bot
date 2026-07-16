package fusionsolar

import (
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
)

func setupReportErrorSentry(t *testing.T) *sentry.MockTransport {
	t.Helper()
	transport := &sentry.MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}
	t.Cleanup(func() {
		sentry.CurrentHub().BindClient(nil)
	})
	return transport
}

func TestReportErrorAttachesArgsToSentryContext(t *testing.T) {
	transport := setupReportErrorSentry(t)

	inner := errors.New("decode failed")
	ReportError("Error decoding base64 image", inner, "station", "Casa", "value", "12,3")

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 sentry event, got %d", len(events))
	}
	report, ok := events[0].Contexts["report"]
	if !ok {
		t.Fatalf("expected report context on event, got %#v", events[0].Contexts)
	}
	if got, _ := report["message"].(string); got != "Error decoding base64 image" {
		t.Fatalf("report.message = %v, want Error decoding base64 image", report["message"])
	}
	if got, _ := report["station"].(string); got != "Casa" {
		t.Fatalf("report.station = %v, want Casa", report["station"])
	}
	if got, _ := report["value"].(string); got != "12,3" {
		t.Fatalf("report.value = %v, want 12,3", report["value"])
	}
	if events[0].Exception == nil || len(events[0].Exception) == 0 {
		t.Fatal("expected exception payload on event")
	}
}

func TestReportErrorSkipsSentryWhenErrNil(t *testing.T) {
	transport := setupReportErrorSentry(t)

	ReportError("nothing to capture", nil, "station", "Casa")

	if events := transport.Events(); len(events) != 0 {
		t.Fatalf("expected no sentry events for nil err, got %d", len(events))
	}
}

func TestReportErrorIgnoresOddTrailingArg(t *testing.T) {
	transport := setupReportErrorSentry(t)

	ReportError("partial args", errors.New("boom"), "station", "Casa", "orphan")

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 sentry event, got %d", len(events))
	}
	report := events[0].Contexts["report"]
	if _, ok := report["orphan"]; ok {
		t.Fatalf("orphan key should be ignored without a value, got %#v", report)
	}
	if got, _ := report["station"].(string); got != "Casa" {
		t.Fatalf("report.station = %v, want Casa", report["station"])
	}
}
