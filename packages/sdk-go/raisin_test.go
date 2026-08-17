package raisin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"type":"email.sent"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	header := fmt.Sprintf("t=%s,v1=%s", ts, sig)
	if !VerifyWebhookSignature(secret, header, body, 0) {
		t.Fatal("expected valid")
	}
	if VerifyWebhookSignature(secret, header, []byte("{}"), 0) {
		t.Fatal("expected body mismatch")
	}
}
