package audience

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
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
		var emailAddr string
		_ = s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1 AND team_id = $2`, id, teamID).Scan(&emailAddr)
		if emailAddr != "" {
			if *unsubscribed {
				_, _ = s.Pool.Exec(ctx, `
					INSERT INTO suppressions (team_id, email, reason)
					VALUES ($1, $2, 'unsubscribe')
					ON CONFLICT (team_id, email) DO UPDATE SET reason = EXCLUDED.reason
				`, teamID, emailAddr)
			} else {
				_, _ = s.Pool.Exec(ctx, `
					DELETE FROM suppressions WHERE team_id = $1 AND email = $2 AND reason = 'unsubscribe'
				`, teamID, emailAddr)
			}
		}
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

func (s *Service) AddToSegment(ctx context.Context, teamID, segmentID, contactID uuid.UUID) error {
	var ok bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM segments s
			JOIN contacts c ON c.id = $2 AND c.team_id = s.team_id
			WHERE s.id = $1 AND s.team_id = $3
		)
	`, segmentID, contactID, teamID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return apierr.NotFound
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO segment_members (segment_id, contact_id) VALUES ($1,$2) ON CONFLICT DO NOTHING
	`, segmentID, contactID)
	return err
}

func (s *Service) RemoveFromSegment(ctx context.Context, teamID, segmentID, contactID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM segment_members sm
		USING segments s, contacts c
		WHERE sm.segment_id = s.id AND sm.contact_id = c.id
		  AND sm.segment_id = $1 AND sm.contact_id = $2
		  AND s.team_id = $3 AND c.team_id = $3
	`, segmentID, contactID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
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

type TopicSubscription struct {
	TopicID              uuid.UUID `json:"topic_id"`
	Name                 string    `json:"name"`
	DefaultSubscription  string    `json:"default_subscription"`
	Subscribed           bool      `json:"subscribed"`
	Explicit             bool      `json:"explicit"`
}

// SetTopicSubscription upserts an explicit subscription row. Contact and topic must belong to the team.
func (s *Service) SetTopicSubscription(ctx context.Context, teamID, contactID, topicID uuid.UUID, subscribed bool) error {
	var ok bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM contacts c
			JOIN topics t ON t.team_id = c.team_id
			WHERE c.id = $1 AND t.id = $2 AND c.team_id = $3
		)
	`, contactID, topicID, teamID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return apierr.NotFound
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO topic_subscriptions (topic_id, contact_id, subscribed, updated_at)
		VALUES ($1,$2,$3,now())
		ON CONFLICT (topic_id, contact_id) DO UPDATE
		SET subscribed = EXCLUDED.subscribed, updated_at = now()
	`, topicID, contactID, subscribed)
	return err
}

// ListContactTopics returns every team topic with effective subscription status for the contact.
func (s *Service) ListContactTopics(ctx context.Context, teamID, contactID uuid.UUID) ([]*TopicSubscription, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND team_id = $2)`, contactID, teamID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apierr.NotFound
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT t.id, t.name, t.default_subscription,
		       COALESCE(ts.subscribed, t.default_subscription = 'opt_out') AS subscribed,
		       (ts.contact_id IS NOT NULL) AS explicit
		FROM topics t
		LEFT JOIN topic_subscriptions ts ON ts.topic_id = t.id AND ts.contact_id = $1
		WHERE t.team_id = $2
		ORDER BY t.created_at DESC
	`, contactID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*TopicSubscription
	for rows.Next() {
		var ts TopicSubscription
		if err := rows.Scan(&ts.TopicID, &ts.Name, &ts.DefaultSubscription, &ts.Subscribed, &ts.Explicit); err != nil {
			return nil, err
		}
		out = append(out, &ts)
	}
	return out, nil
}

// TopicRecipientEmails returns contact emails eligible for a topic broadcast (optional segment filter).
func (s *Service) TopicRecipientEmails(ctx context.Context, teamID, topicID uuid.UUID, segmentID *uuid.UUID) ([]string, error) {
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM topics WHERE id = $1 AND team_id = $2)`, topicID, teamID).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apierr.NotFound
	}

	q := `
		SELECT c.email
		FROM contacts c
		CROSS JOIN topics t
		LEFT JOIN topic_subscriptions ts ON ts.topic_id = t.id AND ts.contact_id = c.id
		WHERE t.id = $1 AND c.team_id = $2 AND t.team_id = $2
		  AND c.unsubscribed = FALSE
		  AND COALESCE(ts.subscribed, t.default_subscription = 'opt_out') = TRUE
	`
	args := []any{topicID, teamID}
	if segmentID != nil {
		q += `
		  AND EXISTS (
		    SELECT 1 FROM segment_members sm
		    WHERE sm.contact_id = c.id AND sm.segment_id = $3
		  )`
		args = append(args, *segmentID)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var emails []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
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
