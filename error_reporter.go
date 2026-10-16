package fusionsolar

import (
	"errors"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// alreadyReportedError marks an error that was already sent to Sentry.
// Recovered panics use it so the CLI logs the failure without capturing
// the same event a second time.
type alreadyReportedError struct{ err error }

func (e alreadyReportedError) Error() string { return e.err.Error() }

func (e alreadyReportedError) Unwrap() error { return e.err }

func alreadyReported(err error) bool {
	var reported alreadyReportedError
	return errors.As(err, &reported)
}

// ReportError centralizes error reporting to both structured logging and Sentry.
// It ensures that all errors are captured consistently across the application.
// If err is non-nil and has not already been sent, it is captured by Sentry.
func ReportError(msg string, err error, args ...any) {
	if err != nil {
		if !alreadyReported(err) {
			sentry.CaptureException(err)
		}

		// Append error to arguments for structured logging
		args = append(args, "error", err)
	}

	slog.Error(msg, args...)
}

// FlushSentry flushes any buffered Sentry events.
// It acts as an abstraction so callers don't need to depend on the sentry package directly.
func FlushSentry(timeout time.Duration) {
	sentry.Flush(timeout)
}
