package template

import (
	"context"
	"encoding/json"
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

type Template struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Subject     *string   `json:"subject"`
	HTML        *string   `json:"html"`
	Text        *string   `json:"text"`
	Variables   []any     `json:"variables"`
	Status      string    `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	HTML      string `json:"html"`
	Text      string `json:"text"`
	Variables []any  `json:"variables"`
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Template, error) {
	if req.Name == "" {
		return nil, apierr.Validation("name is required")
	}
	vj, _ := json.Marshal(req.Variables)
	if vj == nil {
		vj = []byte("[]")
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO templates (team_id, name, subject, html, text, variables)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id
	`, teamID, req.Name, nullStr(req.Subject), nullStr(req.HTML), nullStr(req.Text), vj).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Template, error) {
	var t Template
	var vars []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, subject, html, text, variables, status, published_at, created_at, updated_at
		FROM templates WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&t.ID, &t.Name, &t.Subject, &t.HTML, &t.Text, &vars, &t.Status, &t.PublishedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(vars, &t.Variables)
	return &t, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Template, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, subject, html, text, variables, status, published_at, created_at, updated_at
		FROM templates WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Template
	for rows.Next() {
		var t Template
		var vars []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Subject, &t.HTML, &t.Text, &vars, &t.Status, &t.PublishedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(vars, &t.Variables)
		out = append(out, &t)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, teamID, id uuid.UUID, req CreateRequest) (*Template, error) {
	vj, _ := json.Marshal(req.Variables)
	if vj == nil {
		vj = []byte("[]")
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE templates SET name = COALESCE(NULLIF($3,''), name),
			subject = COALESCE($4, subject), html = COALESCE($5, html), text = COALESCE($6, text),
			variables = $7, updated_at = now()
		WHERE id = $1 AND team_id = $2
	`, id, teamID, req.Name, nullStr(req.Subject), nullStr(req.HTML), nullStr(req.Text), vj)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Publish(ctx context.Context, teamID, id uuid.UUID) (*Template, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE templates SET status = 'published', published_at = now(), updated_at = now()
		WHERE id = $1 AND team_id = $2
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Duplicate(ctx context.Context, teamID, id uuid.UUID) (*Template, error) {
	src, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	subj, html, text := "", "", ""
	if src.Subject != nil {
		subj = *src.Subject
	}
	if src.HTML != nil {
		html = *src.HTML
	}
	if src.Text != nil {
		text = *src.Text
	}
	return s.Create(ctx, teamID, CreateRequest{
		Name: src.Name + " (copy)", Subject: subj, HTML: html, Text: text, Variables: src.Variables,
	})
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM templates WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

// Render replaces {{var}} placeholders.
func Render(html string, vars map[string]string) string {
	out := html
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
		out = strings.ReplaceAll(out, "{{ "+k+" }}", v)
	}
	return out
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
