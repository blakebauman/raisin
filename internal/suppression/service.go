package suppression

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/raisin-run/raisin/internal/apierr"
	"github.com/raisin-run/raisin/internal/db"
)

type Service struct {
	Pool *db.Pool
}

type Suppression struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) Add(ctx context.Context, teamID uuid.UUID, email, reason string) (*Suppression, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, apierr.Validation("email is required")
	}
	if reason == "" {
		reason = "manual"
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO suppressions (team_id, email, reason)
		VALUES ($1,$2,$3)
		ON CONFLICT (team_id, email) DO UPDATE SET reason = EXCLUDED.reason
		RETURNING id
	`, teamID, email, reason).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) AddBatch(ctx context.Context, teamID uuid.UUID, emails []string, reason string) (int, error) {
	if len(emails) > 100 {
		return 0, apierr.Validation("max 100 emails per batch")
	}
	n := 0
	for _, e := range emails {
		if _, err := s.Add(ctx, teamID, e, reason); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Suppression, error) {
	var sp Suppression
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, reason, created_at FROM suppressions WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&sp.ID, &sp.Email, &sp.Reason, &sp.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Suppression, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, email, reason, created_at FROM suppressions WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Suppression
	for rows.Next() {
		var sp Suppression
		if err := rows.Scan(&sp.ID, &sp.Email, &sp.Reason, &sp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &sp)
	}
	return out, nil
}

func (s *Service) Remove(ctx context.Context, teamID uuid.UUID, idOrEmail string) error {
	id, err := uuid.Parse(idOrEmail)
	if err == nil {
		tag, err := s.Pool.Exec(ctx, `DELETE FROM suppressions WHERE id = $1 AND team_id = $2`, id, teamID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apierr.NotFound
		}
		return nil
	}
	tag, err := s.Pool.Exec(ctx, `DELETE FROM suppressions WHERE email = $1 AND team_id = $2`, strings.ToLower(idOrEmail), teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}
