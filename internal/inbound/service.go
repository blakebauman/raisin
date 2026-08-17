package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/mail"
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

type AttachmentMeta struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageKey  string    `json:"-"`
}

func (s *Service) Store(ctx context.Context, teamID uuid.UUID, from string, to []string, subject, html, text, s3Key, providerMessageID string, domainID *uuid.UUID) (*ReceivedEmail, error) {
	return s.StoreWithAttachments(ctx, teamID, from, to, subject, html, text, s3Key, providerMessageID, domainID, nil)
}

func (s *Service) StoreWithAttachments(ctx context.Context, teamID uuid.UUID, from string, to []string, subject, html, text, s3Key, providerMessageID string, domainID *uuid.UUID, atts []ParsedAttachment) (*ReceivedEmail, error) {
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
	for _, a := range atts {
		if err := s.saveAttachment(ctx, teamID, id, a); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) saveAttachment(ctx context.Context, teamID, receivedID uuid.UUID, a ParsedAttachment) error {
	if a.Filename == "" || len(a.Content) == 0 {
		return nil
	}
	ct := a.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	key := storage.KeyForInboundAttachment(teamID.String(), receivedID.String(), a.Filename)
	if s.Storage != nil {
		if err := s.Storage.Put(ctx, key, a.Content, ct); err != nil {
			return fmt.Errorf("store attachment: %w", err)
		}
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO received_attachments (received_email_id, filename, content_type, size_bytes, storage_key)
		VALUES ($1,$2,$3,$4,$5)
	`, receivedID, filepath.Base(a.Filename), ct, len(a.Content), key)
	return err
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

func (s *Service) ListAttachments(ctx context.Context, teamID, receivedID uuid.UUID) ([]AttachmentMeta, error) {
	var ok bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM received_emails WHERE id = $1 AND team_id = $2)`, receivedID, teamID).Scan(&ok)
	if !ok {
		return nil, apierr.NotFound
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, filename, content_type, size_bytes, storage_key
		FROM received_attachments WHERE received_email_id = $1 ORDER BY created_at
	`, receivedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttachmentMeta
	for rows.Next() {
		var a AttachmentMeta
		if err := rows.Scan(&a.ID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StorageKey); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *Service) GetAttachment(ctx context.Context, teamID, receivedID, attachmentID uuid.UUID) (*AttachmentMeta, []byte, error) {
	var ok bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM received_emails WHERE id = $1 AND team_id = $2)`, receivedID, teamID).Scan(&ok)
	if !ok {
		return nil, nil, apierr.NotFound
	}
	var a AttachmentMeta
	err := s.Pool.QueryRow(ctx, `
		SELECT id, filename, content_type, size_bytes, storage_key
		FROM received_attachments WHERE id = $1 AND received_email_id = $2
	`, attachmentID, receivedID).Scan(&a.ID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StorageKey)
	if err == pgx.ErrNoRows {
		return nil, nil, apierr.NotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if s.Storage == nil {
		return &a, nil, nil
	}
	body, _, err := s.Storage.Get(ctx, a.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return &a, body, nil
}

// HandleSESNNotification parses SES inbound receipt JSON (mail + receipt with S3 action).
func (s *Service) HandleSESNNotification(ctx context.Context, raw []byte) error {
	var n struct {
		NotificationType string `json:"notificationType"`
		Mail             struct {
			Source        string   `json:"source"`
			Destination   []string `json:"destination"`
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

	teamID, domainID, err := s.teamForRecipients(ctx, n.Mail.Destination)
	if err != nil || teamID == uuid.Nil {
		return nil
	}

	html, text := "", ""
	var atts []ParsedAttachment
	s3Key := n.Receipt.Action.Key
	s3Bucket := n.Receipt.Action.Bucket
	if s3Key == "" && n.Mail.MessageID != "" {
		s3Key = "ses/" + n.Mail.MessageID
	}
	if s.Storage != nil && s3Key != "" {
		body, _, err := s.Storage.GetFrom(ctx, s3Bucket, s3Key)
		if err == nil {
			html, text, atts = parseMIMEMessage(body)
			_ = s.Storage.Put(ctx, storage.KeyForInbound(teamID.String(), n.Mail.MessageID), body, "message/rfc822")
		}
	}

	_, err = s.StoreWithAttachments(ctx, teamID, n.Mail.Source, n.Mail.Destination, n.Mail.CommonHeaders.Subject, html, text, s3Key, n.Mail.MessageID, domainID, atts)
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
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "content-type: text/html") || strings.Contains(lower, "<html") {
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
