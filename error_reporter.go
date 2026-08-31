package fusionsolar

import (
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// ReportError centralizes error reporting to both structured logging and Sentry.
// Structured key/value args (pairs, same shape as slog) are attached to the
// Sentry event under the "report" context so operators keep station/value
// fields that would otherwise only appear in local logs.
// If err is non-nil, it is captured by Sentry.
func ReportError(msg string, err error, args ...any) {
	if err != nil {
		sentry.WithScope(func(scope *sentry.Scope) {
			ctx := sentry.Context{"message": msg}
			for i := 0; i+1 < len(args); i += 2 {
				key, ok := args[i].(string)
				if !ok {
					continue
				}
				ctx[key] = args[i+1]
			}
			scope.SetContext("report", ctx)
			sentry.CaptureException(err)
		})

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
