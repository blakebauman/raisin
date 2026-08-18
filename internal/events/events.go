package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/blakebauman/raisin/internal/billing"
	"github.com/blakebauman/raisin/internal/config"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/snsverify"
	"github.com/blakebauman/raisin/internal/suppression"
	"github.com/blakebauman/raisin/internal/webhook"
	"github.com/redis/go-redis/v9"
)

type Processor struct {
	Pool         *db.Pool
	Redis        *redis.Client
	Webhooks     *webhook.Service
	Suppressions *suppression.Service
	Billing      *billing.Service
	Automations  interface {
		Trigger(ctx context.Context, teamID uuid.UUID, triggerType string, contactID, emailID, receivedID *uuid.UUID, contextData map[string]any) error
	}
}

type SESEvent struct {
	EventType string `json:"eventType"`
	Mail      struct {
		MessageID     string              `json:"messageId"`
		Destination   []string            `json:"destination"`
		Tags          map[string][]string `json:"tags"`
	} `json:"mail"`
	Bounce *struct {
		BounceType        string `json:"bounceType"`
		BouncedRecipients []struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"bouncedRecipients"`
	} `json:"bounce"`
	Complaint *struct {
		ComplainedRecipients []struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"complainedRecipients"`
	} `json:"complaint"`
	Click *struct {
		Link string `json:"link"`
	} `json:"click"`
	Open *struct{} `json:"open"`
}

func (p *Processor) HandleSESJSON(ctx context.Context, raw []byte) error {
	// SNS may wrap the message — require a valid signature when Type is set
	var envelope snsverify.Envelope
	body := raw
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Type != "" {
		if err := snsverify.Verify(envelope); err != nil {
			return fmt.Errorf("sns verify: %w", err)
		}
		if envelope.Type == "Notification" && envelope.Message != "" {
			body = []byte(envelope.Message)
		} else {
			return nil
		}
	}

	var ev SESEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	if ev.Mail.MessageID == "" {
		return nil
	}

	var emailID, teamID uuid.UUID
	var status string
	err := p.Pool.QueryRow(ctx, `
		SELECT id, team_id, status FROM emails WHERE provider_message_id = $1
	`, ev.Mail.MessageID).Scan(&emailID, &teamID, &status)
	if err != nil {
		return nil // unknown message
	}

	mapped := mapSESType(ev.EventType)
	permanentBounce := mapped == "email.bounced" && ev.Bounce != nil && strings.EqualFold(ev.Bounce.BounceType, "Permanent")
	transientBounce := mapped == "email.bounced" && ev.Bounce != nil && !strings.EqualFold(ev.Bounce.BounceType, "Permanent")

	var affected []string
	switch {
	case mapped == "email.bounced":
		affected = bounceRecipientEmails(ev)
	case mapped == "email.complained":
		affected = complaintRecipientEmails(ev)
	}

	data, _ := json.Marshal(ev)
	var eventID uuid.UUID
	var createdAt time.Time
	_ = p.Pool.QueryRow(ctx, `
		INSERT INTO email_events (team_id, email_id, type, data) VALUES ($1,$2,$3,$4)
		RETURNING id, created_at
	`, teamID, emailID, mapped, data).Scan(&eventID, &createdAt)

	// Soft/transient bounces are events only. Permanent bounce/complaint only flip
	// message status when every known destination was affected (no all-to_addrs guess).
	newStatus := ""
	if !transientBounce {
		candidate := statusFromEvent(mapped)
		if candidate == "bounced" || candidate == "complained" {
			dests := normalizeEmails(ev.Mail.Destination)
			if len(dests) == 0 {
				var toAddrs []string
				_ = p.Pool.QueryRow(ctx, `SELECT to_addrs FROM emails WHERE id = $1`, emailID).Scan(&toAddrs)
				dests = normalizeEmails(toAddrs)
			}
			if recipientsCoverAll(affected, dests) {
				newStatus = candidate
			}
		} else {
			newStatus = candidate
		}
	}
	if newStatus != "" && canTransitionStatus(status, newStatus) {
		_, _ = p.Pool.Exec(ctx, `UPDATE emails SET status = $2, updated_at = now() WHERE id = $1`, emailID, newStatus)
	}

	if mapped == "email.complained" || permanentBounce {
		reason := "bounce"
		if mapped == "email.complained" {
			reason = "complaint"
		}
		// Never fall back to all to_addrs — missing SES recipient lists means skip suppress.
		for _, a := range affected {
			_, _ = p.Suppressions.Add(ctx, teamID, a, reason)
		}
	}

	payload := map[string]any{
		"email_id":   emailID.String(),
		"message_id": ev.Mail.MessageID,
		"type":       mapped,
	}
	if len(affected) > 0 {
		payload["recipients"] = affected
	}
	if mapped == "email.bounced" && ev.Bounce != nil {
		payload["bounce_type"] = ev.Bounce.BounceType
	}
	if eventID != uuid.Nil {
		liveData, _ := json.Marshal(payload)
		p.publishLive(ctx, teamID, LiveEvent{
			ID: eventID, Type: mapped, CreatedAt: createdAt, Data: liveData,
		})
	}
	_ = p.Webhooks.Fanout(ctx, teamID, mapped, payload)
	if p.Automations != nil {
		_ = p.Automations.Trigger(ctx, teamID, mapped, nil, &emailID, nil, map[string]any{
			"email_id": emailID.String(), "type": mapped,
		})
	}
	return nil
}

func bounceRecipientEmails(ev SESEvent) []string {
	if ev.Bounce == nil {
		return nil
	}
	var out []string
	for _, r := range ev.Bounce.BouncedRecipients {
		if e := strings.TrimSpace(r.EmailAddress); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func complaintRecipientEmails(ev SESEvent) []string {
	if ev.Complaint == nil {
		return nil
	}
	var out []string
	for _, r := range ev.Complaint.ComplainedRecipients {
		if e := strings.TrimSpace(r.EmailAddress); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func normalizeEmails(addrs []string) []string {
	seen := make(map[string]struct{}, len(addrs))
	var out []string
	for _, a := range addrs {
		e := strings.ToLower(strings.TrimSpace(a))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

// recipientsCoverAll is true when every destination appears in affected (case-insensitive).
// Empty affected or destinations means unknown scope — never treat as full-message bounce.
func recipientsCoverAll(affected, destinations []string) bool {
	aff := normalizeEmails(affected)
	dests := normalizeEmails(destinations)
	if len(aff) == 0 || len(dests) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(aff))
	for _, a := range aff {
		set[a] = struct{}{}
	}
	for _, d := range dests {
		if _, ok := set[d]; !ok {
			return false
		}
	}
	return true
}

func isTerminalStatus(s string) bool {
	switch s {
	case "bounced", "complained", "failed", "suppressed", "canceled":
		return true
	default:
		return false
	}
}

// canTransitionStatus blocks delivery/sent from overwriting terminal outcomes.
func canTransitionStatus(from, to string) bool {
	if to == "" {
		return false
	}
	if isTerminalStatus(from) && !isTerminalStatus(to) {
		return false
	}
	return true
}

func (p *Processor) RecordLocalEvent(ctx context.Context, teamID, emailID uuid.UUID, eventType string, data any) error {
	payloadMap := ensureEmailID(data, emailID)
	payload, _ := json.Marshal(payloadMap)
	var eventID uuid.UUID
	var createdAt time.Time
	err := p.Pool.QueryRow(ctx, `
		INSERT INTO email_events (team_id, email_id, type, data) VALUES ($1,$2,$3,$4)
		RETURNING id, created_at
	`, teamID, emailID, eventType, payload).Scan(&eventID, &createdAt)
	if err != nil {
		return err
	}
	if st := statusFromEvent(eventType); st != "" {
		var cur string
		_ = p.Pool.QueryRow(ctx, `SELECT status FROM emails WHERE id = $1`, emailID).Scan(&cur)
		if canTransitionStatus(cur, st) {
			_, _ = p.Pool.Exec(ctx, `UPDATE emails SET status = $2, updated_at = now() WHERE id = $1`, emailID, st)
		}
	}
	p.publishLive(ctx, teamID, LiveEvent{
		ID: eventID, Type: eventType, CreatedAt: createdAt, Data: payload,
	})
	if err := p.Webhooks.Fanout(ctx, teamID, eventType, payloadMap); err != nil {
		return err
	}
	if p.Automations != nil {
		_ = p.Automations.Trigger(ctx, teamID, eventType, nil, &emailID, nil, map[string]any{
			"email_id": emailID.String(), "type": eventType,
		})
	}
	return nil
}

func ensureEmailID(data any, emailID uuid.UUID) map[string]any {
	out := map[string]any{}
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			out[k] = val
		}
	case nil:
		// empty
	default:
		b, err := json.Marshal(v)
		if err == nil {
			_ = json.Unmarshal(b, &out)
		}
	}
	if _, ok := out["email_id"]; !ok {
		out["email_id"] = emailID.String()
	}
	return out
}

func mapSESType(t string) string {
	switch strings.ToLower(t) {
	case "send":
		return "email.sent"
	case "delivery":
		return "email.delivered"
	case "bounce":
		return "email.bounced"
	case "complaint":
		return "email.complained"
	case "reject":
		return "email.failed"
	case "deliverydelay":
		return "email.delivery_delayed"
	case "open":
		return "email.opened"
	case "click":
		return "email.clicked"
	default:
		return "email." + strings.ToLower(t)
	}
}

func statusFromEvent(t string) string {
	switch t {
	case "email.sent":
		return "sent"
	case "email.delivered":
		return "delivered"
	case "email.bounced":
		return "bounced"
	case "email.complained":
		return "complained"
	case "email.failed":
		return "failed"
	case "email.suppressed":
		return "suppressed"
	default:
		return ""
	}
}

type SQSConsumer struct {
	Client   *sqs.Client
	QueueURL string
	Proc     *Processor
}

func NewSQSConsumer(cfg config.Config, proc *Processor) (*SQSConsumer, error) {
	if cfg.SQSEventsQueueURL == "" {
		return nil, fmt.Errorf("SQS_EVENTS_QUEUE_URL not set")
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	var opts []func(*sqs.Options)
	if cfg.AWSEndpointURL != "" {
		opts = append(opts, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(cfg.AWSEndpointURL)
		})
	}
	return &SQSConsumer{
		Client:   sqs.NewFromConfig(awsCfg, opts...),
		QueueURL: cfg.SQSEventsQueueURL,
		Proc:     proc,
	}, nil
}

func (c *SQSConsumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		out, err := c.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.QueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		for _, m := range out.Messages {
			if err := c.Proc.HandleSESJSON(ctx, []byte(aws.ToString(m.Body))); err != nil {
				// leave message for retry / DLQ rather than acknowledging a failed handle
				continue
			}
			_, _ = c.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(c.QueueURL),
				ReceiptHandle: m.ReceiptHandle,
			})
		}
	}
}
