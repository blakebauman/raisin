package unsubtoken

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSignVerify(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tok := Sign("secret", id, "User <A@Example.com>", time.Hour)
	got, err := Verify("secret", id, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestVerifyRejects(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	other := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tok := Sign("secret", id, "a@example.com", time.Hour)
	if _, err := Verify("wrong", id, tok); err == nil {
		t.Fatal("expected bad secret")
	}
	if _, err := Verify("secret", other, tok); err == nil {
		t.Fatal("expected id mismatch")
	}
	if _, err := Verify("secret", id, tok+"x"); err == nil {
		t.Fatal("expected bad sig")
	}
	expired := Sign("secret", id, "a@example.com", -time.Hour)
	if _, err := Verify("secret", id, expired); err == nil {
		t.Fatal("expected expired")
	}
	if !strings.Contains(tok, ".") {
		t.Fatal(tok)
	}
}
