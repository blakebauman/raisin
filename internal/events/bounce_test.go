package events_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/blakebauman/raisin/internal/events"
)

func TestSESEventBounceRecipientsJSON(t *testing.T) {
	raw := []byte(`{
		"eventType":"Bounce",
		"mail":{"messageId":"m1"},
		"bounce":{
			"bounceType":"Permanent",
			"bouncedRecipients":[{"emailAddress":"bad@example.com"},{"emailAddress":"also@example.com"}]
		}
	}`)
	var ev events.SESEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(ev.Bounce.BounceType, "Permanent") {
		t.Fatalf("type %+v", ev.Bounce)
	}
	if len(ev.Bounce.BouncedRecipients) != 2 {
		t.Fatalf("recipients %+v", ev.Bounce.BouncedRecipients)
	}
}

func TestSESEventTransientBounceJSON(t *testing.T) {
	raw := []byte(`{"eventType":"Bounce","mail":{"messageId":"m1"},"bounce":{"bounceType":"Transient","bouncedRecipients":[{"emailAddress":"soft@example.com"}]}}`)
	var ev events.SESEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(ev.Bounce.BounceType, "Transient") {
		t.Fatalf("got %+v", ev.Bounce)
	}
}
