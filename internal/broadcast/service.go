package broadcast

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/raisin-run/raisin/internal/apierr"
	"github.com/raisin-run/raisin/internal/db"
	"github.com/raisin-run/raisin/internal/jobs"
)

type Service struct {
	Pool   *db.Pool
	Client *asynq.Client
}

type Broadcast struct {
	ID          uuid.UUID  `json:"id"`
	SegmentID   *uuid.UUID `json:"segment_id"`
	Name        *string    `json:"name"`
	From        string     `json:"from"`
	Subject     string     `json:"subject"`
	HTML        *string    `json:"html"`
	Text        *string    `json:"text"`
	TemplateID  *uuid.UUID `json:"template_id"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRequest struct {
	SegmentID  *uuid.UUID `json:"segment_id"`
	Name       string     `json:"name"`
	From       string     `json:"from"`
	Subject    string     `json:"subject"`
	HTML       string     `json:"html"`
	Text       string     `json:"text"`
	TemplateID *uuid.UUID `json:"template_id"`
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Broadcast, error) {
	if req.From == "" || req.Subject == "" {
		return nil, apierr.Validation("from and subject are required")
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO broadcasts (team_id, segment_id, name, from_addr, subject, html, text, template_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, teamID, req.SegmentID, nullStr(req.Name), req.From, req.Subject, nullStr(req.HTML), nullStr(req.Text), req.TemplateID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Broadcast, error) {
	var b Broadcast
	err := s.Pool.QueryRow(ctx, `
		SELECT id, segment_id, name, from_addr, subject, html, text, template_id, status, scheduled_at, sent_at, created_at
		FROM broadcasts WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&b.ID, &b.SegmentID, &b.Name, &b.From, &b.Subject, &b.HTML, &b.Text, &b.TemplateID, &b.Status, &b.ScheduledAt, &b.SentAt, &b.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &b, err
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Broadcast, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, segment_id, name, from_addr, subject, html, text, template_id, status, scheduled_at, sent_at, created_at
		FROM broadcasts WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Broadcast
	for rows.Next() {
		var b Broadcast
		if err := rows.Scan(&b.ID, &b.SegmentID, &b.Name, &b.From, &b.Subject, &b.HTML, &b.Text, &b.TemplateID, &b.Status, &b.ScheduledAt, &b.SentAt, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, nil
}

func (s *Service) Send(ctx context.Context, teamID, id uuid.UUID) (*Broadcast, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE broadcasts SET status = 'queued', updated_at = now() WHERE id = $1 AND team_id = $2 AND status IN ('draft','queued')
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	task, err := jobs.NewBroadcastSendTask(id.String(), teamID.String())
	if err != nil {
		return nil, err
	}
	if _, err := jobs.Enqueue(s.Client, task); err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Cancel(ctx context.Context, teamID, id uuid.UUID) (*Broadcast, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE broadcasts SET status = 'canceled', updated_at = now()
		WHERE id = $1 AND team_id = $2 AND status IN ('draft','queued','sending')
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM broadcasts WHERE id = $1 AND team_id = $2`, id, teamID)
	return err
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
