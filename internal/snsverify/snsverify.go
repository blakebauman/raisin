package snsverify

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	certCache   = map[string]*x509.Certificate{}
	certCacheMu sync.Mutex
	httpClient  = &http.Client{Timeout: 10 * time.Second}
)

type Envelope struct {
	Type             string `json:"Type"`
	MessageID        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
	SubscribeURL     string `json:"SubscribeURL"`
	Token            string `json:"Token"`
}

// Verify checks an SNS envelope signature. Empty Signature/SigningCertURL is invalid.
func Verify(e Envelope) error {
	if e.Signature == "" || e.SigningCertURL == "" {
		return fmt.Errorf("missing SNS signature fields")
	}
	if e.SignatureVersion != "" && e.SignatureVersion != "1" {
		return fmt.Errorf("unsupported SignatureVersion %q", e.SignatureVersion)
	}
	if err := assertAWSCertURL(e.SigningCertURL); err != nil {
		return err
	}
	cert, err := fetchCert(e.SigningCertURL)
	if err != nil {
		return err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("cert is not RSA")
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil {
		return err
	}
	canonical := buildStringToSign(e)
	sum := sha1.Sum([]byte(canonical))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA1, sum[:], sig); err != nil {
		return fmt.Errorf("sns signature invalid: %w", err)
	}
	return nil
}

// ConfirmSubscribe GETs SubscribeURL after validating it is an AWS SNS HTTPS URL.
func ConfirmSubscribe(subscribeURL string) error {
	if err := assertAWSSubscribeURL(subscribeURL); err != nil {
		return err
	}
	resp, err := httpClient.Get(subscribeURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("subscribe confirmation HTTP %d", resp.StatusCode)
	}
	return nil
}

func assertAWSCertURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "https" || !strings.HasSuffix(strings.ToLower(u.Host), ".amazonaws.com") {
		return fmt.Errorf("invalid SigningCertURL host")
	}
	return nil
}

func assertAWSSubscribeURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := strings.ToLower(u.Host)
	if u.Scheme != "https" {
		return fmt.Errorf("invalid SubscribeURL scheme")
	}
	// sns.<region>.amazonaws.com or sns.amazonaws.com
	if host != "sns.amazonaws.com" && !strings.HasPrefix(host, "sns.") {
		return fmt.Errorf("invalid SubscribeURL host")
	}
	if !strings.HasSuffix(host, ".amazonaws.com") {
		return fmt.Errorf("invalid SubscribeURL host")
	}
	return nil
}

func fetchCert(certURL string) (*x509.Certificate, error) {
	certCacheMu.Lock()
	if c, ok := certCache[certURL]; ok {
		certCacheMu.Unlock()
		return c, nil
	}
	certCacheMu.Unlock()

	resp, err := httpClient.Get(certURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	certCacheMu.Lock()
	certCache[certURL] = cert
	certCacheMu.Unlock()
	return cert, nil
}

func buildStringToSign(e Envelope) string {
	var b strings.Builder
	write := func(k, v string) {
		if v == "" {
			return
		}
		b.WriteString(k)
		b.WriteByte('\n')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	switch e.Type {
	case "Notification":
		write("Message", e.Message)
		write("MessageId", e.MessageID)
		if e.Subject != "" {
			write("Subject", e.Subject)
		}
		write("Timestamp", e.Timestamp)
		write("TopicArn", e.TopicArn)
		write("Type", e.Type)
	case "SubscriptionConfirmation", "UnsubscribeConfirmation":
		write("Message", e.Message)
		write("MessageId", e.MessageID)
		write("SubscribeURL", e.SubscribeURL)
		write("Timestamp", e.Timestamp)
		write("Token", e.Token)
		write("TopicArn", e.TopicArn)
		write("Type", e.Type)
	}
	return b.String()
}
