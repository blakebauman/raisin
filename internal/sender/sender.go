package sender

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"github.com/blakebauman/raisin/internal/config"
	"github.com/google/uuid"
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
	CreateDomain(ctx context.Context, domain, region string) (*IdentityResult, error)
	GetDomain(ctx context.Context, domain, region string) (*IdentityResult, error)
	VerifyDomain(ctx context.Context, domain, region string) (*IdentityResult, error)
	DeleteDomain(ctx context.Context, domain, region string) error
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
	clients map[string]*sesv2.Client
	cfg     config.Config
}

func NewSESIdentity(cfg config.Config) (*SESIdentity, error) {
	return &SESIdentity{clients: map[string]*sesv2.Client{}, cfg: cfg}, nil
}

func (s *SESIdentity) clientFor(region string) (*sesv2.Client, error) {
	if region == "" {
		region = s.cfg.AWSRegion
	}
	if c, ok := s.clients[region]; ok {
		return c, nil
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	var clientOpts []func(*sesv2.Options)
	if s.cfg.AWSEndpointURL != "" {
		clientOpts = append(clientOpts, func(o *sesv2.Options) {
			o.BaseEndpoint = aws.String(s.cfg.AWSEndpointURL)
		})
	}
	c := sesv2.NewFromConfig(awsCfg, clientOpts...)
	s.clients[region] = c
	return c, nil
}

func mailFromMX(region string) string {
	if region == "" {
		region = "us-east-1"
	}
	return "feedback-smtp." + region + ".amazonses.com"
}

func (s *SESIdentity) CreateDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	client, err := s.clientFor(region)
	if err != nil {
		return nil, err
	}
	_, err = client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	})
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		return nil, err
	}
	return s.GetDomain(ctx, domain, region)
}

func (s *SESIdentity) GetDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	client, err := s.clientFor(region)
	if err != nil {
		return nil, err
	}
	out, err := client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{
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
		{Name: "send." + domain, Type: "MX", Value: mailFromMX(region)},
		{Name: "send." + domain, Type: "TXT", Value: `"v=spf1 include:amazonses.com ~all"`},
	}
	return res, nil
}

func (s *SESIdentity) VerifyDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	return s.GetDomain(ctx, domain, region)
}

func (s *SESIdentity) DeleteDomain(ctx context.Context, domain, region string) error {
	client, err := s.clientFor(region)
	if err != nil {
		return err
	}
	_, err = client.DeleteEmailIdentity(ctx, &sesv2.DeleteEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	})
	return err
}

// StubIdentity for local/dev without SES. VerifyDomain always succeeds so
// Mailpit flows can leave test mode and still send from a "verified" domain.
type StubIdentity struct{}

func (StubIdentity) CreateDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	return &IdentityResult{
		Verified: false,
		DKIM: []DNSRecord{
			{Name: "raisin1._domainkey." + domain, Type: "CNAME", Value: "raisin1.dkim.raisin.run"},
			{Name: "raisin2._domainkey." + domain, Type: "CNAME", Value: "raisin2.dkim.raisin.run"},
			{Name: "raisin3._domainkey." + domain, Type: "CNAME", Value: "raisin3.dkim.raisin.run"},
		},
		SPF: []DNSRecord{
			{Name: "send." + domain, Type: "MX", Value: mailFromMX(region)},
			{Name: "send." + domain, Type: "TXT", Value: `"v=spf1 include:amazonses.com ~all"`},
		},
	}, nil
}

func (StubIdentity) GetDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	return StubIdentity{}.CreateDomain(ctx, domain, region)
}

func (StubIdentity) VerifyDomain(ctx context.Context, domain, region string) (*IdentityResult, error) {
	res, err := StubIdentity{}.CreateDomain(ctx, domain, region)
	if err != nil {
		return nil, err
	}
	res.Verified = true
	return res, nil
}

func (StubIdentity) DeleteDomain(ctx context.Context, domain, region string) error { return nil }

func NewIdentity(cfg config.Config) IdentityManager {
	if cfg.SenderDriver == "ses" {
		id, err := NewSESIdentity(cfg)
		if err == nil {
			return id
		}
	}
	return StubIdentity{}
}

// ConfigurationSets provisions SES configuration sets + managed dedicated IP pools.
type ConfigurationSets interface {
	Ensure(ctx context.Context, name, region string) ([]PoolIPInfo, error)
}

// PoolIPInfo is a dedicated IP associated with a pool after Ensure.
type PoolIPInfo struct {
	Address     string
	ProviderRef string
	Status      string
}

type StubConfigurationSets struct{}

func (StubConfigurationSets) Ensure(ctx context.Context, name, region string) ([]PoolIPInfo, error) {
	return nil, nil
}

type SESConfigurationSets struct {
	cfg config.Config
}

func NewConfigurationSets(cfg config.Config) ConfigurationSets {
	if cfg.SenderDriver == "ses" {
		return &SESConfigurationSets{cfg: cfg}
	}
	return StubConfigurationSets{}
}

func (s *SESConfigurationSets) Ensure(ctx context.Context, name, region string) ([]PoolIPInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("configuration set name required")
	}
	if region == "" {
		region = s.cfg.AWSRegion
	}
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var clientOpts []func(*sesv2.Options)
	if s.cfg.AWSEndpointURL != "" {
		clientOpts = append(clientOpts, func(o *sesv2.Options) {
			o.BaseEndpoint = aws.String(s.cfg.AWSEndpointURL)
		})
	}
	client := sesv2.NewFromConfig(awsCfg, clientOpts...)
	_, err = client.CreateConfigurationSet(ctx, &sesv2.CreateConfigurationSetInput{
		ConfigurationSetName: aws.String(name),
	})
	if err != nil {
		var exists *types.AlreadyExistsException
		if !errors.As(err, &exists) {
			return nil, err
		}
	}
	if err := s.ensureEventDestination(ctx, client, name); err != nil {
		return nil, err
	}
	if err := s.ensureDedicatedPool(ctx, client, name); err != nil {
		return nil, err
	}
	return s.listPoolIPs(ctx, client, name)
}

func (s *SESConfigurationSets) ensureDedicatedPool(ctx context.Context, client *sesv2.Client, name string) error {
	_, err := client.CreateDedicatedIpPool(ctx, &sesv2.CreateDedicatedIpPoolInput{
		PoolName:    aws.String(name),
		ScalingMode: types.ScalingModeManaged,
	})
	if err != nil {
		var exists *types.AlreadyExistsException
		if !errors.As(err, &exists) {
			return fmt.Errorf("create dedicated ip pool: %w", err)
		}
	}
	_, err = client.PutConfigurationSetDeliveryOptions(ctx, &sesv2.PutConfigurationSetDeliveryOptionsInput{
		ConfigurationSetName: aws.String(name),
		SendingPoolName:      aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("bind config set to dedicated pool: %w", err)
	}
	return nil
}

func (s *SESConfigurationSets) listPoolIPs(ctx context.Context, client *sesv2.Client, poolName string) ([]PoolIPInfo, error) {
	out, err := client.GetDedicatedIps(ctx, &sesv2.GetDedicatedIpsInput{
		PoolName: aws.String(poolName),
	})
	if err != nil {
		return nil, fmt.Errorf("list dedicated ips: %w", err)
	}
	var ips []PoolIPInfo
	for _, dip := range out.DedicatedIps {
		addr := aws.ToString(dip.Ip)
		if addr == "" {
			continue
		}
		status := "assigned"
		switch dip.WarmupStatus {
		case types.WarmupStatusInProgress:
			status = "warming"
		case types.WarmupStatusDone:
			status = "active"
		}
		ips = append(ips, PoolIPInfo{
			Address:     addr,
			ProviderRef: poolName + "/" + addr,
			Status:      status,
		})
	}
	return ips, nil
}

func (s *SESConfigurationSets) ensureEventDestination(ctx context.Context, client *sesv2.Client, configSet string) error {
	topic := s.cfg.SESSNSTopicARN
	if topic == "" {
		return nil
	}
	dest := &types.EventDestinationDefinition{
		Enabled: true,
		MatchingEventTypes: []types.EventType{
			types.EventTypeSend,
			types.EventTypeReject,
			types.EventTypeBounce,
			types.EventTypeComplaint,
			types.EventTypeDelivery,
			types.EventTypeOpen,
			types.EventTypeClick,
			types.EventTypeDeliveryDelay,
		},
		SnsDestination: &types.SnsDestination{TopicArn: aws.String(topic)},
	}
	_, err := client.CreateConfigurationSetEventDestination(ctx, &sesv2.CreateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(configSet),
		EventDestinationName: aws.String("sns"),
		EventDestination:     dest,
	})
	if err == nil {
		return nil
	}
	var exists *types.AlreadyExistsException
	if !errors.As(err, &exists) {
		return err
	}
	_, err = client.UpdateConfigurationSetEventDestination(ctx, &sesv2.UpdateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(configSet),
		EventDestinationName: aws.String("sns"),
		EventDestination:     dest,
	})
	return err
}
