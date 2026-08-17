package auth_test

import (
	"testing"

	"github.com/raisin-run/raisin/internal/auth"
)

func TestHashKeyStable(t *testing.T) {
	a := auth.HashKey("ra_demo_00000000000000000000000000000000")
	b := auth.HashKey("ra_demo_00000000000000000000000000000000")
	if a != b || a == "" {
		t.Fatalf("hash mismatch")
	}
	if a == auth.HashKey("other") {
		t.Fatal("expected different hash")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	raw, prefix, hash := auth.GenerateAPIKey()
	if len(raw) < 10 || prefix == "" || hash == "" {
		t.Fatalf("bad key gen")
	}
	if auth.HashKey(raw) != hash {
		t.Fatal("hash does not match raw")
	}
}
