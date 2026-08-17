package email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/raisin-run/raisin/internal/apierr"
	"github.com/raisin-run/raisin/internal/auth"
	"github.com/raisin-run/raisin/internal/db"
	"github.com/raisin-run/raisin/internal/jobs"
	"github.com/raisin-run/raisin/internal/storage"
)

type Service struct {
	Pool    *db.Pool
	Client  *asynq.Client
	Storage storage.Store
}

type SendRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc"`
	Bcc         []string          `json:"bcc"`
	ReplyTo     []string          `json:"reply_to"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html"`
	Text        string            `json:"text"`
	Headers     map[string]string `json:"headers"`
	Tags        map[string]string `json:"tags"`
	ScheduledAt *time.Time        `json:"scheduled_at"`
	TemplateID  *uuid.UUID        `json:"template_id"`
	Attachments []AttachmentIn    `json:"attachments"`
}

type AttachmentIn struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"` // base64
	ContentID   string `json:"content_id"`
}

type Email struct {
	ID                uuid.UUID         `json:"id"`
	TeamID            uuid.UUID         `json:"team_id"`
	From              string            `json:"from"`
	To                []string          `json:"to"`
	Cc                []string          `json:"cc"`
	Bcc               []string          `json:"bcc"`
	ReplyTo           []string          `json:"reply_to"`
	Subject           string            `json:"subject"`
	HTML              *string           `json:"html,omitempty"`
	Text              *string           `json:"text,omitempty"`
	Headers           map[string]string `json:"headers"`
	Tags              map[string]string `json:"tags"`
	Status            string            `json:"status"`
	ProviderMessageID *string           `json:"provider_message_id,omitempty"`
	ScheduledAt       *time.Time        `json:"scheduled_at,omitempty"`
	SentAt            *time.Time        `json:"sent_at,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func (s *Service) Send(ctx context.Context, team *auth.Team, req SendRequest, idempotencyKey string) (*Email, error) {
	if err := validateSend(req); err != nil {
		return nil, err
	}
	if team.BillingStatus == "paused" {
		return nil, apierr.QuotaExceeded
	}

	domainName := extractDomain(req.From)
	var domainID *uuid.UUID
	var domainStatus string
	var openTrack, clickTrack bool
	err := s.Pool.QueryRow(ctx, `
		SELECT id, status, open_tracking, click_tracking FROM domains
		WHERE team_id = $1 AND name = $2
	`, team.ID, domainName).Scan(&domainID, &domainStatus, &openTrack, &clickTrack)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if !team.TestMode {
		if err == pgx.ErrNoRows || domainStatus != "verified" {
			return nil, apierr.DomainNotVerified
		}
	}

	// Suppression check
	for _, addr := range append(append(req.To, req.Cc...), req.Bcc...) {
		emailAddr := normalizeAddr(addr)
		var exists bool
		_ = s.Pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM suppressions WHERE team_id = $1 AND email = $2)
		`, team.ID, emailAddr).Scan(&exists)
		if exists {
			return nil, apierr.Validation(fmt.Sprintf("%s is on the suppression list", emailAddr))
		}
	}

	// Quota
	ok, err := s.checkQuota(ctx, team)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apierr.QuotaExceeded
	}

	if idempotencyKey != "" {
		var existingID uuid.UUID
		err := s.Pool.QueryRow(ctx, `
			SELECT email_id FROM idempotency_keys
			WHERE team_id = $1 AND key = $2 AND expires_at > now() AND email_id IS NOT NULL
		`, team.ID, idempotencyKey).Scan(&existingID)
		if err == nil {
			return s.Get(ctx, team.ID, existingID)
		}
	}

	headersJSON, _ := json.Marshal(req.Headers)
	if headersJSON == nil {
		headersJSON = []byte("{}")
	}
	tagsJSON, _ := json.Marshal(req.Tags)
	if tagsJSON == nil {
		tagsJSON = []byte("{}")
	}
	if req.Cc == nil {
		req.Cc = []string{}
	}
	if req.Bcc == nil {
		req.Bcc = []string{}
	}
	if req.ReplyTo == nil {
		req.ReplyTo = []string{}
	}

	status := "queued"
	var scheduledAt *time.Time
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
		status = "scheduled"
		scheduledAt = req.ScheduledAt
	}

	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO emails (
			team_id, domain_id, from_addr, to_addrs, cc_addrs, bcc_addrs, reply_to,
			subject, html, text, headers, tags, status, scheduled_at, template_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`, team.ID, domainID, req.From, req.To, req.Cc, req.Bcc, req.ReplyTo,
		req.Subject, nullStr(req.HTML), nullStr(req.Text), headersJSON, tagsJSON,
		status, scheduledAt, req.TemplateID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if s.Storage != nil {
		for _, a := range req.Attachments {
			if a.Filename == "" || a.Content == "" {
				continue
			}
			raw, err := base64.StdEncoding.DecodeString(a.Content)
			if err != nil {
				// try raw URL encoding variant
				raw, err = base64.URLEncoding.DecodeString(a.Content)
				if err != nil {
					return nil, apierr.Validation("invalid attachment content encoding")
				}
			}
			ct := a.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			key := storage.KeyForEmail(team.ID.String(), id.String(), a.Filename)
			if err := s.Storage.Put(ctx, key, raw, ct); err != nil {
				return nil, fmt.Errorf("store attachment: %w", err)
			}
			_, _ = s.Pool.Exec(ctx, `
				INSERT INTO attachments (team_id, email_id, filename, content_type, size_bytes, s3_key, content_id)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
			`, team.ID, id, a.Filename, ct, len(raw), key, nullStr(a.ContentID))
		}
	}

	if idempotencyKey != "" {
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO idempotency_keys (team_id, key, email_id)
			VALUES ($1,$2,$3)
			ON CONFLICT (team_id, key) DO NOTHING
		`, team.ID, idempotencyKey, id)
	}

	task, err := jobs.NewEmailSendTask(id.String(), team.ID.String())
	if err != nil {
		return nil, err
	}
	if scheduledAt != nil {
		task, err = jobs.NewScheduledEmailSendTask(id.String(), team.ID.String(), *scheduledAt)
		if err != nil {
			return nil, err
		}
	}
	if _, err := jobs.Enqueue(s.Client, task); err != nil {
		return nil, err
	}

	return s.Get(ctx, team.ID, id)
}

func (s *Service) SendBatch(ctx context.Context, team *auth.Team, reqs []SendRequest) ([]*Email, error) {
	if len(reqs) == 0 {
		return nil, apierr.Validation("batch cannot be empty")
	}
	if len(reqs) > 100 {
		return nil, apierr.Validation("batch max is 100 emails")
	}
	out := make([]*Email, 0, len(reqs))
	for _, r := range reqs {
		e, err := s.Send(ctx, team, r, "")
		if err != nil {
			return out, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Email, error) {
	var e Email
	var headers, tags []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id, team_id, from_addr, to_addrs, cc_addrs, bcc_addrs, reply_to,
		       subject, html, text, headers, tags, status, provider_message_id,
		       scheduled_at, sent_at, created_at, updated_at
		FROM emails WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(
		&e.ID, &e.TeamID, &e.From, &e.To, &e.Cc, &e.Bcc, &e.ReplyTo,
		&e.Subject, &e.HTML, &e.Text, &headers, &tags, &e.Status, &e.ProviderMessageID,
		&e.ScheduledAt, &e.SentAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(headers, &e.Headers)
	_ = json.Unmarshal(tags, &e.Tags)
	if e.Headers == nil {
		e.Headers = map[string]string{}
	}
	if e.Tags == nil {
		e.Tags = map[string]string{}
	}
	return &e, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]*Email, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, team_id, from_addr, to_addrs, cc_addrs, bcc_addrs, reply_to,
		       subject, html, text, headers, tags, status, provider_message_id,
		       scheduled_at, sent_at, created_at, updated_at
		FROM emails
		WHERE team_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, teamID, parseCursor(cursor), limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var list []*Email
	for rows.Next() {
		var e Email
		var headers, tags []byte
		if err := rows.Scan(
			&e.ID, &e.TeamID, &e.From, &e.To, &e.Cc, &e.Bcc, &e.ReplyTo,
			&e.Subject, &e.HTML, &e.Text, &headers, &tags, &e.Status, &e.ProviderMessageID,
			&e.ScheduledAt, &e.SentAt, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, "", err
		}
		_ = json.Unmarshal(headers, &e.Headers)
		_ = json.Unmarshal(tags, &e.Tags)
		list = append(list, &e)
	}
	next := ""
	if len(list) > limit {
		list = list[:limit]
		next = list[len(list)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return list, next, nil
}

func (s *Service) Cancel(ctx context.Context, teamID, id uuid.UUID) (*Email, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE emails SET status = 'canceled', updated_at = now()
		WHERE id = $1 AND team_id = $2 AND status = 'scheduled'
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, apierr.Validation("only scheduled emails can be canceled")
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) UpdateSchedule(ctx context.Context, teamID, id uuid.UUID, at time.Time) (*Email, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE emails SET scheduled_at = $3, status = 'scheduled', updated_at = now()
		WHERE id = $1 AND team_id = $2 AND status IN ('scheduled', 'queued')
	`, id, teamID, at)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, apierr.Validation("email cannot be rescheduled")
	}
	task, err := jobs.NewScheduledEmailSendTask(id.String(), teamID.String(), at)
	if err != nil {
		return nil, err
	}
	if _, err := jobs.Enqueue(s.Client, task); err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) checkQuota(ctx context.Context, team *auth.Team) (bool, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	var sent int
	err := s.Pool.QueryRow(ctx, `
		SELECT COALESCE(emails_sent, 0) FROM usage_periods
		WHERE team_id = $1 AND period_start = $2
	`, team.ID, start).Scan(&sent)
	if err == pgx.ErrNoRows {
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO usage_periods (team_id, period_start, period_end, emails_sent)
			VALUES ($1,$2,$3,0) ON CONFLICT DO NOTHING
		`, team.ID, start, end)
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return sent < team.MonthlyQuota, nil
}

func validateSend(req SendRequest) error {
	if req.From == "" {
		return apierr.Validation("from is required")
	}
	if len(req.To) == 0 {
		return apierr.Validation("to is required")
	}
	if req.Subject == "" && req.TemplateID == nil {
		return apierr.Validation("subject is required")
	}
	if req.HTML == "" && req.Text == "" && req.TemplateID == nil {
		return apierr.Validation("html or text is required")
	}
	return nil
}

func extractDomain(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		if i := strings.LastIndex(from, "@"); i >= 0 {
			return strings.ToLower(strings.Trim(from[i+1:], "> "))
		}
		return ""
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

func normalizeAddr(s string) string {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return strings.ToLower(addr.Address)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseCursor(c string) *time.Time {
	if c == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, c)
	if err != nil {
		return nil
	}
	return &t
}

type AttachmentMeta struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentID   *string   `json:"content_id,omitempty"`
	S3Key       string    `json:"-"`
}

func (s *Service) ListAttachments(ctx context.Context, teamID, emailID uuid.UUID) ([]AttachmentMeta, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, filename, content_type, size_bytes, content_id, s3_key
		FROM attachments WHERE team_id = $1 AND email_id = $2 ORDER BY created_at
	`, teamID, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttachmentMeta
	for rows.Next() {
		var a AttachmentMeta
		if err := rows.Scan(&a.ID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.ContentID, &a.S3Key); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *Service) GetAttachment(ctx context.Context, teamID, emailID, attachmentID uuid.UUID) (*AttachmentMeta, []byte, error) {
	var a AttachmentMeta
	err := s.Pool.QueryRow(ctx, `
		SELECT id, filename, content_type, size_bytes, content_id, s3_key
		FROM attachments WHERE id = $1 AND email_id = $2 AND team_id = $3
	`, attachmentID, emailID, teamID).Scan(&a.ID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.ContentID, &a.S3Key)
	if err == pgx.ErrNoRows {
		return nil, nil, apierr.NotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if s.Storage == nil {
		return &a, nil, nil
	}
	body, _, err := s.Storage.Get(ctx, a.S3Key)
	if err != nil {
		return nil, nil, err
	}
	return &a, body, nil
}

type Event struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Service) ListEvents(ctx context.Context, teamID, emailID uuid.UUID) ([]Event, error) {
	// ensure email belongs to team
	var exists bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM emails WHERE id = $1 AND team_id = $2)`, emailID, teamID).Scan(&exists)
	if !exists {
		return nil, apierr.NotFound
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, type, data, created_at FROM email_events
		WHERE email_id = $1 AND team_id = $2 ORDER BY created_at ASC
	`, emailID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Type, &e.Data, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
