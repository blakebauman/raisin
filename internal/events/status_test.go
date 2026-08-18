package events

import "testing"

func TestRecipientsCoverAll(t *testing.T) {
	cases := []struct {
		name         string
		affected     []string
		destinations []string
		want         bool
	}{
		{"full cover", []string{"A@ex.com", "b@ex.com"}, []string{"a@ex.com", "B@ex.com"}, true},
		{"partial", []string{"a@ex.com"}, []string{"a@ex.com", "b@ex.com"}, false},
		{"empty affected", nil, []string{"a@ex.com"}, false},
		{"empty dests", []string{"a@ex.com"}, nil, false},
		{"extra affected ok", []string{"a@ex.com", "x@ex.com"}, []string{"a@ex.com"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := recipientsCoverAll(tc.affected, tc.destinations); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCanTransitionStatus(t *testing.T) {
	if canTransitionStatus("bounced", "delivered") {
		t.Fatal("terminal must not downgrade to delivered")
	}
	if canTransitionStatus("complained", "sent") {
		t.Fatal("terminal must not downgrade to sent")
	}
	if !canTransitionStatus("sent", "delivered") {
		t.Fatal("sent -> delivered should be allowed")
	}
	if !canTransitionStatus("delivered", "bounced") {
		t.Fatal("delivered -> bounced should be allowed")
	}
	if !canTransitionStatus("bounced", "complained") {
		t.Fatal("terminal -> terminal should be allowed")
	}
	if canTransitionStatus("failed", "") {
		t.Fatal("empty to should be rejected")
	}
}

func TestNormalizeEmails(t *testing.T) {
	got := normalizeEmails([]string{" A@Ex.com ", "a@ex.com", "", "b@ex.com"})
	if len(got) != 2 || got[0] != "a@ex.com" || got[1] != "b@ex.com" {
		t.Fatalf("got %#v", got)
	}
}
