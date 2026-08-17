package sender

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"
	"github.com/raisin-run/raisin/internal/config"
)

type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	ReplyTo     []string
	Subject     string
	HTML        string
	Text        string
	Headers     map[string]string
	ConfigSet   string
	Tags        map[string]string
	Attachments []Attachment
}

type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	ContentID   string
}

type Result struct {
	MessageID string
}

type Sender interface {
	Send(ctx context.Context, msg Message) (*Result, error)
}

func New(cfg config.Config) (Sender, error) {
	switch cfg.SenderDriver {
	case "ses":
		return NewSES(cfg)
	default:
		return NewMailpit(cfg.MailpitURL), nil
	}
}

// Mailpit sender for local development
type Mailpit struct {
	baseURL string
	client  *http.Client
}

func NewMailpit(baseURL string) *Mailpit {
	return &Mailpit{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *Mailpit) Send(ctx context.Context, msg Message) (*Result, error) {
	fromName, fromEmail := splitAddress(msg.From)
	payload := map[string]any{
		"From":    map[string]string{"Email": fromEmail, "Name": fromName},
		"To":      mailpitAddrs(msg.To),
		"Subject": msg.Subject,
		"HTML":    msg.HTML,
		"Text":    msg.Text,
	}
	if len(msg.Cc) > 0 {
		payload["Cc"] = mailpitAddrs(msg.Cc)
	}
	if len(msg.Bcc) > 0 {
		payload["Bcc"] = mailpitAddrs(msg.Bcc)
	}
	if len(msg.ReplyTo) > 0 {
		rn, re := splitAddress(msg.ReplyTo[0])
		payload["ReplyTo"] = []map[string]string{{"Email": re, "Name": rn}}
	}
	if len(msg.Attachments) > 0 {
		atts := make([]map[string]any, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			atts = append(atts, map[string]any{
				"Filename":    a.Filename,
				"ContentType": a.ContentType,
				"Content":     base64.StdEncoding.EncodeToString(a.Content),
			})
		}
		payload["Attachments"] = atts
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/api/v1/send", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mailpit: %s", string(data))
	}
	var out struct {
		ID string `json:"ID"`
	}
	_ = json.Unmarshal(data, &out)
	if out.ID == "" {
		out.ID = uuid.NewString()
	}
	return &Result{MessageID: out.ID}, nil
}

func splitAddress(s string) (name, email string) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", strings.TrimSpace(s)
	}
	return addr.Name, addr.Address
}

func mailpitAddrs(list []string) []map[string]string {
	out := make([]map[string]string, 0, len(list))
	for _, s := range list {
		n, e := splitAddress(s)
		out = append(out, map[string]string{"Email": e, "Name": n})
	}
	return out
}

// SES sender for production
type SES struct {
	client    *sesv2.Client
	configSet string
}

func NewSES(cfg config.Config) (*SES, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	var clientOpts []func(*sesv2.Options)
	if cfg.AWSEndpointURL != "" {
		clientOpts = append(clientOpts, func(o *sesv2.Options) {
			o.BaseEndpoint = aws.String(cfg.AWSEndpointURL)
		})
	}
	return &SES{
		client:    sesv2.NewFromConfig(awsCfg, clientOpts...),
		configSet: cfg.SESConfigurationSet,
	}, nil
}

func (s *SES) Send(ctx context.Context, msg Message) (*Result, error) {
	if len(msg.Attachments) > 0 {
		return s.sendRaw(ctx, msg)
	}
	dest := &types.Destination{
		ToAddresses:  msg.To,
		CcAddresses:  msg.Cc,
		BccAddresses: msg.Bcc,
	}
	content := &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(msg.Subject)},
			Body:    &types.Body{},
		},
	}
	if msg.HTML != "" {
		content.Simple.Body.Html = &types.Content{Data: aws.String(msg.HTML), Charset: aws.String("UTF-8")}
	}
	if msg.Text != "" {
		content.Simple.Body.Text = &types.Content{Data: aws.String(msg.Text), Charset: aws.String("UTF-8")}
	}
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(msg.From),
		Destination:      dest,
		Content:          content,
		ReplyToAddresses: msg.ReplyTo,
	}
	cs := msg.ConfigSet
	if cs == "" {
		cs = s.configSet
	}
	if cs != "" {
		input.ConfigurationSetName = aws.String(cs)
	}
	for k, v := range msg.Tags {
		input.EmailTags = append(input.EmailTags, types.MessageTag{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	out, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return nil, err
	}
	return &Result{MessageID: aws.ToString(out.MessageId)}, nil
}

func (s *SES) sendRaw(ctx context.Context, msg Message) (*Result, error) {
	raw, err := buildRawMIME(msg)
	if err != nil {
		return nil, err
	}
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(msg.From),
		Destination: &types.Destination{
			ToAddresses:  msg.To,
			CcAddresses:  msg.Cc,
			BccAddresses: msg.Bcc,
		},
		Content: &types.EmailContent{
			Raw: &types.RawMessage{Data: raw},
		},
		ReplyToAddresses: msg.ReplyTo,
	}
	cs := msg.ConfigSet
	if cs == "" {
		cs = s.configSet
	}
	if cs != "" {
		input.ConfigurationSetName = aws.String(cs)
	}
	out, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return nil, err
	}
	return &Result{MessageID: aws.ToString(out.MessageId)}, nil
}

func buildRawMIME(msg Message) ([]byte, error) {
	boundary := "raisin-" + uuid.NewString()
	var b strings.Builder
	from := msg.From
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	if len(msg.Cc) > 0 {
		b.WriteString("Cc: " + strings.Join(msg.Cc, ", ") + "\r\n")
	}
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	for k, v := range msg.Headers {
		b.WriteString(k + ": " + v + "\r\n")
	}
	b.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n\r\n")
	b.WriteString("--" + boundary + "\r\n")
	if msg.HTML != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.HTML + "\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text + "\r\n")
	}
	for _, a := range msg.Attachments {
		b.WriteString("--" + boundary + "\r\n")
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		b.WriteString("Content-Type: " + ct + "; name=\"" + a.Filename + "\"\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("Content-Disposition: attachment; filename=\"" + a.Filename + "\"\r\n")
		if a.ContentID != "" {
			b.WriteString("Content-ID: <" + a.ContentID + ">\r\n")
		}
		b.WriteString("\r\n")
		b.WriteString(base64.StdEncoding.EncodeToString(a.Content) + "\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String()), nil
}

// IdentityManager creates/verifies SES domain identities
type IdentityManager interface {
	CreateDomain(ctx context.Context, domain string) (*IdentityResult, error)
	GetDomain(ctx context.Context, domain string) (*IdentityResult, error)
	VerifyDomain(ctx context.Context, domain string) (*IdentityResult, error)
	DeleteDomain(ctx context.Context, domain string) error
}

type IdentityResult struct {
	Verified bool
	DKIM     []DNSRecord
	SPF      []DNSRecord
}

type DNSRecord struct {
	Name  string
	Type  string
	Value string
}

type SESIdentity struct {
	client *sesv2.Client
}

func NewSESIdentity(cfg config.Config) (*SESIdentity, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	var clientOpts []func(*sesv2.Options)
	if cfg.AWSEndpointURL != "" {
		clientOpts = append(clientOpts, func(o *sesv2.Options) {
			o.BaseEndpoint = aws.String(cfg.AWSEndpointURL)
		})
	}
	return &SESIdentity{client: sesv2.NewFromConfig(awsCfg, clientOpts...)}, nil
}

func (s *SESIdentity) CreateDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	_, err := s.client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	})
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		return nil, err
	}
	return s.GetDomain(ctx, domain)
}

func (s *SESIdentity) GetDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	out, err := s.client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	})
	if err != nil {
		return nil, err
	}
	res := &IdentityResult{Verified: out.VerifiedForSendingStatus}
	if out.DkimAttributes != nil {
		for _, t := range out.DkimAttributes.Tokens {
			res.DKIM = append(res.DKIM, DNSRecord{
				Name:  fmt.Sprintf("%s._domainkey.%s", t, domain),
				Type:  "CNAME",
				Value: fmt.Sprintf("%s.dkim.amazonses.com", t),
			})
		}
	}
	res.SPF = []DNSRecord{
		{Name: "send." + domain, Type: "MX", Value: "feedback-smtp.us-east-1.amazonses.com"},
		{Name: "send." + domain, Type: "TXT", Value: `"v=spf1 include:amazonses.com ~all"`},
	}
	return res, nil
}

func (s *SESIdentity) VerifyDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	return s.GetDomain(ctx, domain)
}

func (s *SESIdentity) DeleteDomain(ctx context.Context, domain string) error {
	_, err := s.client.DeleteEmailIdentity(ctx, &sesv2.DeleteEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	})
	return err
}

// StubIdentity for local/dev without SES. VerifyDomain always succeeds so
// Mailpit flows can leave test mode and still send from a "verified" domain.
type StubIdentity struct{}

func (StubIdentity) CreateDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	return &IdentityResult{
		Verified: false,
		DKIM: []DNSRecord{
			{Name: "raisin1._domainkey." + domain, Type: "CNAME", Value: "raisin1.dkim.raisin.run"},
			{Name: "raisin2._domainkey." + domain, Type: "CNAME", Value: "raisin2.dkim.raisin.run"},
			{Name: "raisin3._domainkey." + domain, Type: "CNAME", Value: "raisin3.dkim.raisin.run"},
		},
		SPF: []DNSRecord{
			{Name: "send." + domain, Type: "MX", Value: "feedback-smtp.us-east-1.amazonses.com"},
			{Name: "send." + domain, Type: "TXT", Value: `"v=spf1 include:amazonses.com ~all"`},
		},
	}, nil
}

func (StubIdentity) GetDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	return StubIdentity{}.CreateDomain(ctx, domain)
}

func (StubIdentity) VerifyDomain(ctx context.Context, domain string) (*IdentityResult, error) {
	res, err := StubIdentity{}.CreateDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	res.Verified = true
	return res, nil
}

func (StubIdentity) DeleteDomain(ctx context.Context, domain string) error { return nil }

func NewIdentity(cfg config.Config) IdentityManager {
	if cfg.SenderDriver == "ses" {
		id, err := NewSESIdentity(cfg)
		if err == nil {
			return id
		}
	}
	return StubIdentity{}
}
