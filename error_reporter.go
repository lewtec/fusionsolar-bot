package fusionsolar

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// ReportError centralizes error reporting, ensuring that all unexpected errors
// are logged properly and reported to Sentry if available.
func ReportError(err error, msg string, args ...any) {
	if err != nil {
		slog.Error(msg, append([]any{"error", err}, args...)...)
		sentry.CaptureException(err)
	}
}
