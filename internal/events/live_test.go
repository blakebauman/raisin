package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLiveChannel(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	got := LiveChannel(id)
	want := "raisin:events:11111111-1111-1111-1111-111111111111"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMatchStreamFilters(t *testing.T) {
	emailID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ev := LiveEvent{
		ID:        uuid.New(),
		Type:      "email.opened",
		CreatedAt: time.Now().UTC(),
		Data:      []byte(`{"email_id":"22222222-2222-2222-2222-222222222222"}`),
	}
	types := map[string]struct{}{"email.opened": {}, "email.clicked": {}}
	if !MatchStreamFilters(ev, types, &emailID) {
		t.Fatal("expected match")
	}
	other := uuid.New()
	if MatchStreamFilters(ev, types, &other) {
		t.Fatal("expected email_id miss")
	}
	if MatchStreamFilters(ev, map[string]struct{}{"email.clicked": {}}, nil) {
		t.Fatal("expected type miss")
	}
	if !MatchStreamFilters(ev, nil, nil) {
		t.Fatal("empty filters should match")
	}
}

func TestEnsureEmailID(t *testing.T) {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	got := ensureEmailID(map[string]any{"link": "https://x"}, id)
	if got["email_id"] != id.String() {
		t.Fatalf("expected inject email_id, got %#v", got)
	}
	if got["link"] != "https://x" {
		t.Fatalf("lost link: %#v", got)
	}
	kept := ensureEmailID(map[string]any{"email_id": "keep"}, id)
	if kept["email_id"] != "keep" {
		t.Fatalf("should keep existing email_id")
	}
}
