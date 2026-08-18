package audience

import "testing"

func TestNormalizePropertyKey(t *testing.T) {
	cases := map[string]string{
		"Plan Tier": "plan_tier",
		"company":   "company",
		"  Foo-Bar ": "foo_bar",
		"!!!":       "",
	}
	for in, want := range cases {
		if got := normalizePropertyKey(in); got != want {
			t.Fatalf("%q -> %q want %q", in, got, want)
		}
	}
}
