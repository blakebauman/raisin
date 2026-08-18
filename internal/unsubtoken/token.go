package unsubtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultTTL = 90 * 24 * time.Hour

// Sign returns a URL-safe token binding emailID to a recipient address.
func Sign(secret string, emailID uuid.UUID, email string, ttl time.Duration) string {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	email = normalizeEmail(email)
	if email == "" {
		return ""
	}
	exp := time.Now().Add(ttl).Unix()
	payload := emailID.String() + "|" + email + "|" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

// Verify checks token and returns the bound recipient email.
func Verify(secret string, emailID uuid.UUID, token string) (string, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}
	payload := string(raw)
	fields := strings.Split(payload, "|")
	if len(fields) != 3 {
		return "", fmt.Errorf("invalid token")
	}
	id, err := uuid.Parse(fields[0])
	if err != nil || id != emailID {
		return "", fmt.Errorf("token email mismatch")
	}
	email := strings.TrimSpace(strings.ToLower(fields[1]))
	if email == "" {
		return "", fmt.Errorf("invalid token email")
	}
	exp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", fmt.Errorf("token expired")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[1])) {
		return "", fmt.Errorf("bad signature")
	}
	return email, nil
}

func normalizeEmail(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s, ">"); j > i {
			s = strings.TrimSpace(s[i+1 : j])
		}
	}
	return s
}
