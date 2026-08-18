package events_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/blakebauman/raisin/internal/events"
)

func TestSESEventBounceTypeJSON(t *testing.T) {
	raw := []byte(`{"eventType":"Bounce","mail":{"messageId":"m1"},"bounce":{"bounceType":"Transient"}}`)
	var ev events.SESEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Bounce == nil || !strings.EqualFold(ev.Bounce.BounceType, "Transient") {
		t.Fatalf("got %+v", ev.Bounce)
	}
	raw2 := []byte(`{"eventType":"Bounce","mail":{"messageId":"m1"},"bounce":{"bounceType":"Permanent"}}`)
	var ev2 events.SESEvent
	if err := json.Unmarshal(raw2, &ev2); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(ev2.Bounce.BounceType, "Permanent") {
		t.Fatalf("got %+v", ev2.Bounce)
	}
}
