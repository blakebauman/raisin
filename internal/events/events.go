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
)

type Processor struct {
	Pool         *db.Pool
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

	data, _ := json.Marshal(ev)
	_, _ = p.Pool.Exec(ctx, `
		INSERT INTO email_events (team_id, email_id, type, data) VALUES ($1,$2,$3,$4)
	`, teamID, emailID, mapped, data)

	// Soft/transient bounces are recorded as events only — do not flip terminal status.
	newStatus := ""
	if !transientBounce {
		newStatus = statusFromEvent(mapped)
	}
	if newStatus != "" {
		_, _ = p.Pool.Exec(ctx, `UPDATE emails SET status = $2, updated_at = now() WHERE id = $1`, emailID, newStatus)
	}

	if mapped == "email.complained" || permanentBounce {
		reason := "bounce"
		addrs := bounceRecipientEmails(ev)
		if mapped == "email.complained" {
			reason = "complaint"
			addrs = complaintRecipientEmails(ev)
		}
		if len(addrs) == 0 {
			// Fallback only when SES omitted recipient lists
			_ = p.Pool.QueryRow(ctx, `SELECT to_addrs FROM emails WHERE id = $1`, emailID).Scan(&addrs)
		}
		for _, a := range addrs {
			_, _ = p.Suppressions.Add(ctx, teamID, a, reason)
		}
	}

	_ = p.Webhooks.Fanout(ctx, teamID, mapped, map[string]any{
		"email_id":   emailID.String(),
		"message_id": ev.Mail.MessageID,
		"type":       mapped,
	})
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

func (p *Processor) RecordLocalEvent(ctx context.Context, teamID, emailID uuid.UUID, eventType string, data any) error {
	payload, _ := json.Marshal(data)
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO email_events (team_id, email_id, type, data) VALUES ($1,$2,$3,$4)
	`, teamID, emailID, eventType, payload)
	if err != nil {
		return err
	}
	if st := statusFromEvent(eventType); st != "" {
		_, _ = p.Pool.Exec(ctx, `UPDATE emails SET status = $2, updated_at = now() WHERE id = $1`, emailID, st)
	}
	if err := p.Webhooks.Fanout(ctx, teamID, eventType, data); err != nil {
		return err
	}
	if p.Automations != nil {
		_ = p.Automations.Trigger(ctx, teamID, eventType, nil, &emailID, nil, map[string]any{
			"email_id": emailID.String(), "type": eventType,
		})
	}
	return nil
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
