package raisin

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.raisin.run"

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Emails     *EmailsService
	Domains    *DomainsService
	Webhooks   *WebhooksService
	APIKeys    *APIKeysService
	Contacts   *ContactsService
	Templates  *TemplatesService
	Broadcasts *BroadcastsService
}

func NewClient(apiKey string) *Client {
	c := &Client{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	c.Emails = &EmailsService{c: c}
	c.Domains = &DomainsService{c: c}
	c.Webhooks = &WebhooksService{c: c}
	c.APIKeys = &APIKeysService{c: c}
	c.Contacts = &ContactsService{c: c}
	c.Templates = &TemplatesService{c: c}
	c.Broadcasts = &BroadcastsService{c: c}
	return c
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "raisin-go/0.1.0")
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		var ae APIError
		_ = json.Unmarshal(data, &ae)
		if ae.Message == "" {
			ae.Message = string(data)
			ae.StatusCode = res.StatusCode
		}
		return &ae
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

type APIError struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Name, e.StatusCode, e.Message)
}

type SendEmailRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Content     string `json:"content"` // base64
	ContentID   string `json:"content_id,omitempty"`
}

type Email struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        []string  `json:"to"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type EmailsService struct{ c *Client }

func (s *EmailsService) Send(ctx context.Context, req *SendEmailRequest) (*Email, error) {
	var out Email
	if err := s.c.do(ctx, http.MethodPost, "/emails", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *EmailsService) Get(ctx context.Context, id string) (*Email, error) {
	var out Email
	if err := s.c.do(ctx, http.MethodGet, "/emails/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *EmailsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/emails", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *EmailsService) Cancel(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodPost, "/emails/"+id+"/cancel", map[string]any{}, nil)
}

type DomainsService struct{ c *Client }

func (s *DomainsService) Create(ctx context.Context, name string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains", map[string]string{"name": name}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/domains", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) Get(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/domains/"+id, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) Verify(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains/"+id+"/verify", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/domains/"+id, nil, nil)
}

type WebhooksService struct{ c *Client }

func (s *WebhooksService) Create(ctx context.Context, endpoint string, events []string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/webhooks", map[string]any{"endpoint": endpoint, "events": events}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *WebhooksService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/webhooks", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *WebhooksService) ListEvents(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/webhooks/"+id+"/events", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *WebhooksService) ListAttempts(ctx context.Context, webhookID, eventID string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/webhooks/"+webhookID+"/events/"+eventID+"/attempts", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *WebhooksService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/webhooks/"+id, nil, nil)
}

type APIKeysService struct{ c *Client }

func (s *APIKeysService) Create(ctx context.Context, name string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/api-keys", map[string]string{"name": name}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIKeysService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/api-keys", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ContactsService struct{ c *Client }

func (s *ContactsService) Create(ctx context.Context, email string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/contacts", map[string]string{"email": email}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContactsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/contacts", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type TemplatesService struct{ c *Client }

func (s *TemplatesService) Create(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/templates", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TemplatesService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/templates", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type BroadcastsService struct{ c *Client }

func (s *BroadcastsService) Create(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/broadcasts", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *BroadcastsService) Send(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/broadcasts/"+id+"/send", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// VerifyWebhookSignature checks Raisin-Signature: t=<unix>,v1=<hex>
func VerifyWebhookSignature(secret, header string, body []byte, maxSkew time.Duration) bool {
	parts := map[string]string{}
	for _, p := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}
	ts, sig := parts["t"], parts["v1"]
	if ts == "" || sig == "" {
		return false
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if maxSkew <= 0 {
		maxSkew = 5 * time.Minute
	}
	if math.Abs(float64(time.Now().Unix()-sec)) > maxSkew.Seconds() {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}
