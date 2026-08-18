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
	APIKey       string
	BaseURL      string
	HTTPClient   *http.Client
	Emails       *EmailsService
	Domains      *DomainsService
	Webhooks     *WebhooksService
	APIKeys      *APIKeysService
	Contacts          *ContactsService
	ContactProperties *ContactPropertiesService
	Topics            *TopicsService
	Templates    *TemplatesService
	Broadcasts   *BroadcastsService
	Suppressions *SuppressionsService
	Automations  *AutomationsService
	IPPools      *IPPoolsService
	OAuth        *OAuthService
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
	c.ContactProperties = &ContactPropertiesService{c: c}
	c.Topics = &TopicsService{c: c}
	c.Templates = &TemplatesService{c: c}
	c.Broadcasts = &BroadcastsService{c: c}
	c.Suppressions = &SuppressionsService{c: c}
	c.Automations = &AutomationsService{c: c}
	c.IPPools = &IPPoolsService{c: c}
	c.OAuth = &OAuthService{c: c}
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

func (s *EmailsService) ListReceived(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/emails/received", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *EmailsService) GetReceived(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/emails/received/"+id, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *EmailsService) ListReceivedAttachments(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/emails/received/"+id+"/attachments", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *EmailsService) GetReceivedAttachment(ctx context.Context, id, attachmentID string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/emails/received/"+id+"/attachments/"+attachmentID, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type DomainsService struct{ c *Client }

func (s *DomainsService) Create(ctx context.Context, name string, region ...string) (map[string]any, error) {
	body := map[string]string{"name": name}
	if len(region) > 0 && region[0] != "" {
		body["region"] = region[0]
	}
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains", body, &out); err != nil {
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

func (s *DomainsService) Regions(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/domains/regions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) Claim(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains/"+id+"/claim", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) ConfirmClaim(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains/"+id+"/claim/confirm", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) SetBIMI(ctx context.Context, id string, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains/"+id+"/bimi", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DomainsService) SetReceiving(ctx context.Context, id string, enabled bool) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/domains/"+id+"/receiving", map[string]any{"enabled": enabled}, &out); err != nil {
		return nil, err
	}
	return out, nil
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

type ContactCreateOpts struct {
	FirstName  string
	LastName   string
	Properties map[string]any
}

func (s *ContactsService) Create(ctx context.Context, email string, opts *ContactCreateOpts) (map[string]any, error) {
	body := map[string]any{"email": email}
	if opts != nil {
		if opts.FirstName != "" {
			body["first_name"] = opts.FirstName
		}
		if opts.LastName != "" {
			body["last_name"] = opts.LastName
		}
		if len(opts.Properties) > 0 {
			body["properties"] = opts.Properties
		}
	}
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/contacts", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContactsService) Update(ctx context.Context, id string, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPatch, "/contacts/"+id, body, &out); err != nil {
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

func (s *ContactsService) ListTopics(ctx context.Context, contactID string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/contacts/"+contactID+"/topics", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContactsService) SetTopic(ctx context.Context, contactID, topicID string, subscribed bool) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPut, "/contacts/"+contactID+"/topics/"+topicID, map[string]any{"subscribed": subscribed}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ContactPropertiesService struct{ c *Client }

func (s *ContactPropertiesService) Create(ctx context.Context, key, typ string) (map[string]any, error) {
	body := map[string]any{"key": key}
	if typ != "" {
		body["type"] = typ
	}
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/contact-properties", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContactPropertiesService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/contact-properties", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContactPropertiesService) Delete(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodDelete, "/contact-properties/"+id, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type TopicsService struct{ c *Client }

func (s *TopicsService) Create(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/topics", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TopicsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/topics", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TopicsService) Delete(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodDelete, "/topics/"+id, nil, &out); err != nil {
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

func (s *BroadcastsService) Send(ctx context.Context, id string, body map[string]any) (map[string]any, error) {
	if body == nil {
		body = map[string]any{}
	}
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/broadcasts/"+id+"/send", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type SuppressionsService struct{ c *Client }

func (s *SuppressionsService) Create(ctx context.Context, email, reason string) (map[string]any, error) {
	if reason == "" {
		reason = "manual"
	}
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/suppressions", map[string]string{"email": email, "reason": reason}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SuppressionsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/suppressions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SuppressionsService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/suppressions/"+id, nil, nil)
}

type AutomationsService struct{ c *Client }

func (s *AutomationsService) Create(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/automations", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AutomationsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/automations", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AutomationsService) Enable(ctx context.Context, id string, enabled bool) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPatch, "/automations/"+id, map[string]any{"enabled": enabled}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AutomationsService) Update(ctx context.Context, id string, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPatch, "/automations/"+id, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AutomationsService) ListRuns(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/automations/"+id+"/runs", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AutomationsService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/automations/"+id, nil, nil)
}

type IPPoolsService struct{ c *Client }

func (s *IPPoolsService) Create(ctx context.Context, name, region string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/ip-pools", map[string]string{"name": name, "region": region}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) List(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/ip-pools", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) Get(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/ip-pools/"+id, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) AssignDomain(ctx context.Context, poolID, domainID string) error {
	return s.c.do(ctx, http.MethodPost, "/ip-pools/"+poolID+"/assign-domain", map[string]string{"domain_id": domainID}, nil)
}

func (s *IPPoolsService) Pause(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/ip-pools/"+id+"/pause", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) Resume(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/ip-pools/"+id+"/resume", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) WarmupTick(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/ip-pools/"+id+"/warmup/tick", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IPPoolsService) Delete(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/ip-pools/"+id, nil, nil)
}

type OAuthService struct{ c *Client }

func (s *OAuthService) CreateApp(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodPost, "/oauth/apps", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OAuthService) ListApps(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.do(ctx, http.MethodGet, "/oauth/apps", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OAuthService) DeleteApp(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/oauth/apps/"+id, nil, nil)
}

// Token exchanges an authorization code or refresh token (no API key required).
func (s *OAuthService) Token(ctx context.Context, body map[string]any) (map[string]any, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.c.BaseURL, "/")+"/oauth/token", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "raisin-go/0.1.0")
	res, err := s.c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth token: %s", string(raw))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
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
