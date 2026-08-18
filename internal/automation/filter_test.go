package automation

import (
	"encoding/json"
	"testing"
)

func TestFilterMatches(t *testing.T) {
	ctx := map[string]any{"email": "a@b.com", "segment": "vip"}
	cases := []struct {
		filter string
		ok     bool
	}{
		{`{}`, true},
		{``, true},
		{`{"email":"a@b.com"}`, true},
		{`{"email":"other@b.com"}`, false},
		{`{"email":"a@b.com","segment":"vip"}`, true},
		{`{"missing":"x"}`, false},
	}
	for _, c := range cases {
		if got := filterMatches(json.RawMessage(c.filter), ctx); got != c.ok {
			t.Fatalf("filter %s: got %v want %v", c.filter, got, c.ok)
		}
	}
}
