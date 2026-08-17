package inbound

import (
	"context"
	"encoding/json"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/storage"
)

type Service struct {
	Pool    *db.Pool
	Storage storage.Store
}

type ReceivedEmail struct {
	ID        uuid.UUID `json:"id"`
	From      string    `json:"from"`
	To        []string  `json:"to"`
	Subject   *string   `json:"subject"`
	HTML      *string   `json:"html"`
	Text      *string   `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) Store(ctx context.Context, teamID uuid.UUID, from string, to []string, subject, html, text, s3Key, providerMessageID string, domainID *uuid.UUID) (*ReceivedEmail, error) {
	if providerMessageID != "" {
		var existing uuid.UUID
		err := s.Pool.QueryRow(ctx, `
			SELECT id FROM received_emails WHERE team_id = $1 AND provider_message_id = $2
		`, teamID, providerMessageID).Scan(&existing)
		if err == nil {
			return s.Get(ctx, teamID, existing)
		}
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO received_emails (team_id, from_addr, to_addrs, subject, html, text, s3_key, provider_message_id, domain_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id
	`, teamID, from, to, nullStr(subject), nullStr(html), nullStr(text), nullStr(s3Key), nullStr(providerMessageID), domainID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*ReceivedEmail, error) {
	var e ReceivedEmail
	err := s.Pool.QueryRow(ctx, `
		SELECT id, from_addr, to_addrs, subject, html, text, created_at
		FROM received_emails WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&e.ID, &e.From, &e.To, &e.Subject, &e.HTML, &e.Text, &e.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &e, err
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*ReceivedEmail, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, from_addr, to_addrs, subject, html, text, created_at
		FROM received_emails WHERE team_id = $1 ORDER BY created_at DESC LIMIT 100
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ReceivedEmail
	for rows.Next() {
		var e ReceivedEmail
		if err := rows.Scan(&e.ID, &e.From, &e.To, &e.Subject, &e.HTML, &e.Text, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, nil
}

// HandleSESNNotification parses SES inbound receipt JSON (mail + receipt with S3 action).
func (s *Service) HandleSESNNotification(ctx context.Context, raw []byte) error {
	var n struct {
		NotificationType string `json:"notificationType"`
		Mail             struct {
			Source      string   `json:"source"`
			Destination []string `json:"destination"`
			CommonHeaders struct {
				Subject string `json:"subject"`
			} `json:"commonHeaders"`
			MessageID string `json:"messageId"`
		} `json:"mail"`
		Receipt struct {
			Action struct {
				Type   string `json:"type"`
				Bucket string `json:"bucketName"`
				Key    string `json:"objectKey"`
			} `json:"action"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return err
	}

	// Resolve team by recipient domain
	teamID, domainID, err := s.teamForRecipients(ctx, n.Mail.Destination)
	if err != nil || teamID == uuid.Nil {
		return nil // drop unknown
	}

	html, text := "", ""
	s3Key := n.Receipt.Action.Key
	s3Bucket := n.Receipt.Action.Bucket
	// When SNS action is primary in the notification, S3 key may be absent —
	// SES S3 action stores under object_key_prefix + messageId by default.
	if s3Key == "" && n.Mail.MessageID != "" {
		s3Key = "ses/" + n.Mail.MessageID
	}
	if s.Storage != nil && s3Key != "" {
		body, _, err := s.Storage.GetFrom(ctx, s3Bucket, s3Key)
		if err == nil {
			html, text = parseMIMEBodies(string(body))
			_ = s.Storage.Put(ctx, storage.KeyForInbound(teamID.String(), n.Mail.MessageID), body, "message/rfc822")
		}
	}

	_, err = s.Store(ctx, teamID, n.Mail.Source, n.Mail.Destination, n.Mail.CommonHeaders.Subject, html, text, s3Key, n.Mail.MessageID, domainID)
	return err
}

func (s *Service) teamForRecipients(ctx context.Context, recipients []string) (uuid.UUID, *uuid.UUID, error) {
	for _, r := range recipients {
		addr, err := mail.ParseAddress(r)
		email := r
		if err == nil {
			email = addr.Address
		}
		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			continue
		}
		domainName := strings.ToLower(parts[1])
		var teamID, domainID uuid.UUID
		err = s.Pool.QueryRow(ctx, `
			SELECT team_id, id FROM domains
			WHERE name = $1 AND status = 'verified' AND receiving_enabled = true
			ORDER BY claimed_at NULLS LAST
			LIMIT 1
		`, domainName).Scan(&teamID, &domainID)
		if err == nil {
			return teamID, &domainID, nil
		}
	}
	return uuid.Nil, nil, nil
}

func parseMIMEBodies(raw string) (html, text string) {
	// Minimal: if looks like HTML use as html else text
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "content-type: text/html") || strings.Contains(lower, "<html") {
		// crude body extract
		if i := strings.Index(raw, "\r\n\r\n"); i >= 0 {
			html = raw[i+4:]
		} else if i := strings.Index(raw, "\n\n"); i >= 0 {
			html = raw[i+2:]
		} else {
			html = raw
		}
		return
	}
	if i := strings.Index(raw, "\r\n\r\n"); i >= 0 {
		text = raw[i+4:]
	} else if i := strings.Index(raw, "\n\n"); i >= 0 {
		text = raw[i+2:]
	} else {
		text = raw
	}
	return
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
