package worker

import "testing"

func TestFinalizeBroadcastOutcome(t *testing.T) {
	cases := []struct {
		ok, fail     int
		wantStatus   string
		wantSentAt   bool
	}{
		{0, 0, "failed", false},
		{3, 0, "sent", true},
		{0, 2, "failed", false},
		{2, 1, "partial", true},
	}
	for _, tc := range cases {
		st, sentAt := finalizeBroadcastOutcome(tc.ok, tc.fail)
		if st != tc.wantStatus || sentAt != tc.wantSentAt {
			t.Fatalf("ok=%d fail=%d got (%s,%v) want (%s,%v)",
				tc.ok, tc.fail, st, sentAt, tc.wantStatus, tc.wantSentAt)
		}
	}
}
