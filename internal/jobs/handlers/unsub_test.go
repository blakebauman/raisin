package worker

import "testing"

func TestResolveUnsubTargets(t *testing.T) {
	cases := []struct {
		to   []string
		req  string
		want int
	}{
		{[]string{"a@b.com"}, "", 1},
		{[]string{"a@b.com", "c@d.com"}, "", 0},
		{[]string{"a@b.com", "c@d.com"}, "c@d.com", 1},
		{[]string{"Name <a@b.com>"}, "a@b.com", 1},
		{[]string{"a@b.com"}, "other@b.com", 0},
	}
	for _, c := range cases {
		got := resolveUnsubTargets(c.to, c.req)
		if len(got) != c.want {
			t.Fatalf("to=%v req=%q got %v want len %d", c.to, c.req, got, c.want)
		}
	}
}
