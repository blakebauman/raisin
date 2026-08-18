package broadcast

import (
	"context"
	"strings"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/jobs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Pool   *db.Pool
	Client *asynq.Client
}

type Broadcast struct {
	ID          uuid.UUID  `json:"id"`
	SegmentID   *uuid.UUID `json:"segment_id"`
	TopicID     *uuid.UUID `json:"topic_id"`
	Name        *string    `json:"name"`
	From        string     `json:"from"`
	Subject     string     `json:"subject"`
	HTML        *string    `json:"html"`
	Text        *string    `json:"text"`
	TemplateID  *uuid.UUID `json:"template_id"`
	Status      string     `json:"status"`
	SentCount   int        `json:"sent_count"`
	FailedCount int        `json:"failed_count"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRequest struct {
	SegmentID  string `json:"segment_id"`
	TopicID    string `json:"topic_id"`
	Name       string `json:"name"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	HTML       string `json:"html"`
	Text       string `json:"text"`
	TemplateID string `json:"template_id"`
}

func parseOptionalUUID(s string) (*uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, apierr.Validation("invalid uuid")
	}
	return &id, nil
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Broadcast, error) {
	if req.From == "" || req.Subject == "" {
		return nil, apierr.Validation("from and subject are required")
	}
	segID, err := parseOptionalUUID(req.SegmentID)
	if err != nil {
		return nil, err
	}
	topicID, err := parseOptionalUUID(req.TopicID)
	if err != nil {
		return nil, err
	}
	if topicID != nil {
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM topics WHERE id = $1 AND team_id = $2)`, *topicID, teamID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierr.Validation("topic not found")
		}
	}
	if segID != nil {
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM segments WHERE id = $1 AND team_id = $2)`, *segID, teamID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierr.Validation("segment not found")
		}
	}
	tplID, err := parseOptionalUUID(req.TemplateID)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO broadcasts (team_id, segment_id, topic_id, name, from_addr, subject, html, text, template_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id
	`, teamID, segID, topicID, nullStr(req.Name), req.From, req.Subject, nullStr(req.HTML), nullStr(req.Text), tplID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Broadcast, error) {
	var b Broadcast
	err := s.Pool.QueryRow(ctx, `
		SELECT id, segment_id, topic_id, name, from_addr, subject, html, text, template_id,
		       status, sent_count, failed_count, scheduled_at, sent_at, created_at
		FROM broadcasts WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(
		&b.ID, &b.SegmentID, &b.TopicID, &b.Name, &b.From, &b.Subject, &b.HTML, &b.Text, &b.TemplateID,
		&b.Status, &b.SentCount, &b.FailedCount, &b.ScheduledAt, &b.SentAt, &b.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &b, err
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Broadcast, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, segment_id, topic_id, name, from_addr, subject, html, text, template_id,
		       status, sent_count, failed_count, scheduled_at, sent_at, created_at
		FROM broadcasts WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Broadcast
	for rows.Next() {
		var b Broadcast
		if err := rows.Scan(
			&b.ID, &b.SegmentID, &b.TopicID, &b.Name, &b.From, &b.Subject, &b.HTML, &b.Text, &b.TemplateID,
			&b.Status, &b.SentCount, &b.FailedCount, &b.ScheduledAt, &b.SentAt, &b.CreatedAt,
		); err != nil {
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
