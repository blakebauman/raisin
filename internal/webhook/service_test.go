package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"type":"email.sent"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	header := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	if !VerifySignature(secret, header, body, 0) {
		t.Fatal("expected valid signature")
	}
	if VerifySignature(secret, header, []byte(`{}`), 0) {
		t.Fatal("expected body mismatch to fail")
	}
	if VerifySignature("other", header, body, 0) {
		t.Fatal("expected secret mismatch to fail")
	}
}
