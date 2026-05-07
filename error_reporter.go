package fusionsolar

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// ReportError centralizes error reporting to both structured logging and Sentry.
// It ensures that all errors are captured consistently across the application.
// If err is non-nil, it is captured by Sentry.
func ReportError(msg string, err error, args ...any) {
	if err != nil {
		sentry.CaptureException(err)

		// Append error to arguments for structured logging
		args = append(args, "error", err)
	}

	slog.Error(msg, args...)
}
