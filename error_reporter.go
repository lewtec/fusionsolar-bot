package fusionsolar

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// ReportError centralizes error handling by capturing the exception in Sentry
// and logging it with slog, adhering to the Single Responsibility Principle.
func ReportError(msg string, err error, args ...any) {
	slogArgs := append([]any{"error", err}, args...)
	slog.Error(msg, slogArgs...)
	sentry.CaptureException(err)
}
