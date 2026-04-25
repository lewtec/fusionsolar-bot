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
	sleepContext(ctx, 30*time.Second)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("sleepContext should return immediately on cancellation, took %s", elapsed)
	}
}
