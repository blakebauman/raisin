package audience

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

type Contact struct {
	ID           uuid.UUID      `json:"id"`
	Email        string         `json:"email"`
	FirstName    *string        `json:"first_name"`
	LastName     *string        `json:"last_name"`
	Unsubscribed bool           `json:"unsubscribed"`
	Properties   map[string]any `json:"properties"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Segment struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Topic struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	Description          *string   `json:"description"`
	DefaultSubscription  string    `json:"default_subscription"`
	CreatedAt            time.Time `json:"created_at"`
}

type ContactProperty struct {
	ID   uuid.UUID `json:"id"`
	Key  string    `json:"key"`
	Type string    `json:"type"`
}

func (s *Service) CreateContact(ctx context.Context, teamID uuid.UUID, email, first, last string, props map[string]any) (*Contact, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, apierr.Validation("email is required")
	}
	pj, _ := json.Marshal(props)
	if pj == nil {
		pj = []byte("{}")
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO contacts (team_id, email, first_name, last_name, properties)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (team_id, email) DO UPDATE SET
			first_name = COALESCE(EXCLUDED.first_name, contacts.first_name),
			last_name = COALESCE(EXCLUDED.last_name, contacts.last_name),
			properties = contacts.properties || EXCLUDED.properties,
			updated_at = now()
		RETURNING id
	`, teamID, email, nullStr(first), nullStr(last), pj).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetContact(ctx, teamID, id)
}

func (s *Service) GetContact(ctx context.Context, teamID, id uuid.UUID) (*Contact, error) {
	var c Contact
	var props []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, unsubscribed, properties, created_at
		FROM contacts WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName, &c.Unsubscribed, &props, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(props, &c.Properties)
	return &c, nil
}

func (s *Service) ListContacts(ctx context.Context, teamID uuid.UUID) ([]*Contact, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, email, first_name, last_name, unsubscribed, properties, created_at
		FROM contacts WHERE team_id = $1 ORDER BY created_at DESC LIMIT 200
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Contact
	for rows.Next() {
		var c Contact
		var props []byte
		if err := rows.Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName, &c.Unsubscribed, &props, &c.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(props, &c.Properties)
		out = append(out, &c)
	}
	return out, nil
}

func (s *Service) UpdateContact(ctx context.Context, teamID, id uuid.UUID, first, last *string, unsubscribed *bool) (*Contact, error) {
	if first != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE contacts SET first_name = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *first, teamID)
	}
	if last != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE contacts SET last_name = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *last, teamID)
	}
	if unsubscribed != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE contacts SET unsubscribed = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *unsubscribed, teamID)
	}
	return s.GetContact(ctx, teamID, id)
}

func (s *Service) DeleteContact(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM contacts WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

func (s *Service) CreateSegment(ctx context.Context, teamID uuid.UUID, name string) (*Segment, error) {
	if name == "" {
		return nil, apierr.Validation("name is required")
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `INSERT INTO segments (team_id, name) VALUES ($1,$2) RETURNING id`, teamID, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &Segment{ID: id, Name: name, CreatedAt: time.Now()}, nil
}

func (s *Service) ListSegments(ctx context.Context, teamID uuid.UUID) ([]*Segment, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, created_at FROM segments WHERE team_id = $1 ORDER BY created_at DESC`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Segment
	for rows.Next() {
		var seg Segment
		if err := rows.Scan(&seg.ID, &seg.Name, &seg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &seg)
	}
	return out, nil
}

func (s *Service) DeleteSegment(ctx context.Context, teamID, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM segments WHERE id = $1 AND team_id = $2`, id, teamID)
	return err
}

func (s *Service) AddToSegment(ctx context.Context, segmentID, contactID uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO segment_members (segment_id, contact_id) VALUES ($1,$2) ON CONFLICT DO NOTHING
	`, segmentID, contactID)
	return err
}

func (s *Service) RemoveFromSegment(ctx context.Context, segmentID, contactID uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM segment_members WHERE segment_id = $1 AND contact_id = $2`, segmentID, contactID)
	return err
}

func (s *Service) CreateTopic(ctx context.Context, teamID uuid.UUID, name, desc, def string) (*Topic, error) {
	if name == "" {
		return nil, apierr.Validation("name is required")
	}
	if def == "" {
		def = "opt_in"
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO topics (team_id, name, description, default_subscription) VALUES ($1,$2,$3,$4) RETURNING id
	`, teamID, name, nullStr(desc), def).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetTopic(ctx, teamID, id)
}

func (s *Service) GetTopic(ctx context.Context, teamID, id uuid.UUID) (*Topic, error) {
	var t Topic
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, description, default_subscription, created_at FROM topics WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&t.ID, &t.Name, &t.Description, &t.DefaultSubscription, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &t, err
}

func (s *Service) ListTopics(ctx context.Context, teamID uuid.UUID) ([]*Topic, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, description, default_subscription, created_at FROM topics WHERE team_id = $1
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.DefaultSubscription, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, nil
}

func (s *Service) DeleteTopic(ctx context.Context, teamID, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM topics WHERE id = $1 AND team_id = $2`, id, teamID)
	return err
}

func (s *Service) CreateProperty(ctx context.Context, teamID uuid.UUID, key, typ string) (*ContactProperty, error) {
	if key == "" {
		return nil, apierr.Validation("key is required")
	}
	if typ == "" {
		typ = "string"
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO contact_properties (team_id, key, type) VALUES ($1,$2,$3) RETURNING id
	`, teamID, key, typ).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &ContactProperty{ID: id, Key: key, Type: typ}, nil
}

func (s *Service) ListProperties(ctx context.Context, teamID uuid.UUID) ([]*ContactProperty, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, key, type FROM contact_properties WHERE team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ContactProperty
	for rows.Next() {
		var p ContactProperty
		if err := rows.Scan(&p.ID, &p.Key, &p.Type); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, nil
}

func (s *Service) SegmentContactIDs(ctx context.Context, segmentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.id FROM contacts c
		JOIN segment_members sm ON sm.contact_id = c.id
		WHERE sm.segment_id = $1 AND c.unsubscribed = FALSE
	`, segmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
