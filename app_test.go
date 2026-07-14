package fusionsolar

import (
	"context"
	"testing"
	"time"
)

func TestSleepContextStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
