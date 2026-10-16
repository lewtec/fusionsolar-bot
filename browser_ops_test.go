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

func TestIsRetryableBrowserErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "wrapped deadline", err: fmt.Errorf("cdp: %w", context.DeadlineExceeded), want: true},
		{name: "deadline text", err: errors.New("context deadline exceeded"), want: true},
		{name: "eof", err: io.EOF, want: true},
		{name: "eof text", err: errors.New("EOF"), want: true},
		{name: "unexpected eof", err: io.ErrUnexpectedEOF, want: true},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "connection refused", err: errors.New("dial tcp 127.0.0.1:9222: connect: connection refused"), want: false},
		{name: "other", err: errElementNotFound, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRetryableBrowserErr(tt.err); got != tt.want {
				t.Errorf("isRetryableBrowserErr(%v) = %v; want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryBrowserOpRetriesDeadlineThenSucceeds(t *testing.T) {
	attempts := 0
	err := retryBrowserOp(t.Context(), "query elements", 3, time.Second, time.Millisecond, func(context.Context) error {
		attempts++
		if attempts < 2 {
			return context.DeadlineExceeded
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRetryBrowserOpDoesNotStartWhenRunContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	attempts := 0
	err := retryBrowserOp(ctx, "query elements", 3, time.Second, time.Millisecond, func(context.Context) error {
		attempts++
		return context.DeadlineExceeded
	})
	if attempts != 0 {
		t.Fatalf("attempts = %d, want 0", attempts)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !strings.Contains(err.Error(), "query elements") {
		t.Fatalf("expected operation name in error, got %v", err)
	}
}

func TestRetryBrowserOpStopsWhenRunContextEndsAfterFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	attempts := 0
	err := retryBrowserOp(ctx, "query elements", 3, time.Second, time.Millisecond, func(context.Context) error {
		attempts++
		cancel()
		return io.EOF
	})
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected run cancellation, got %v", err)
	}
	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected browser error in message, got %v", err)
	}
}

func TestRetryBrowserOpDoesNotRetryConnectionRefused(t *testing.T) {
	attempts := 0
	refused := errors.New("dial tcp 127.0.0.1:9222: connect: connection refused")
	err := retryBrowserOp(t.Context(), "connect", 3, time.Second, time.Millisecond, func(context.Context) error {
		attempts++
		return refused
	})
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if !errors.Is(err, refused) {
		t.Fatalf("expected connection refused, got %v", err)
	}
}

func TestRetryBrowserOpExhaustsEOF(t *testing.T) {
	attempts := 0
	err := retryBrowserOp(t.Context(), `query elements "i.cookiePolicy-icon"`, 3, time.Second, time.Millisecond, func(context.Context) error {
		attempts++
		return io.EOF
	})
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") || !strings.Contains(err.Error(), "cookiePolicy") {
		t.Fatalf("expected selector and attempt count, got %v", err)
	}
}

func TestRetryBrowserOpBoundsCallTimeout(t *testing.T) {
	start := time.Now()
	err := retryBrowserOp(t.Context(), "query elements", 1, 40*time.Millisecond, time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("call timeout was not bounded, took %s", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}
