package broadcast

import (
	"testing"
	"time"
)

func TestScheduleBroadcastStatus(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Minute)

	st, at := scheduleBroadcastStatus(&future, now)
	if st != "scheduled" || at == nil || !at.Equal(future) {
		t.Fatalf("future: got %s %#v", st, at)
	}
	st, at = scheduleBroadcastStatus(&past, now)
	if st != "queued" || at != nil {
		t.Fatalf("past: got %s %#v", st, at)
	}
	st, at = scheduleBroadcastStatus(nil, now)
	if st != "queued" || at != nil {
		t.Fatalf("nil: got %s %#v", st, at)
	}
}
