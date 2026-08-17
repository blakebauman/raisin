package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	HTTP   *http.Client
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	Endpoint  string    `json:"endpoint"`
	Events    []string  `json:"events"`
	Secret    string    `json:"signing_secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRequest struct {
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Webhook, error) {
	if req.Endpoint == "" || len(req.Events) == 0 {
		return nil, apierr.Validation("endpoint and events are required")
	}
	secret := "whsec_" + randomHex(24)
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO webhooks (team_id, endpoint, events, secret)
		VALUES ($1,$2,$3,$4) RETURNING id
	`, teamID, req.Endpoint, req.Events, secret).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id, true)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID, includeSecret bool) (*Webhook, error) {
	var w Webhook
	err := s.Pool.QueryRow(ctx, `
		SELECT id, endpoint, events, secret, enabled, created_at
		FROM webhooks WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&w.ID, &w.Endpoint, &w.Events, &w.Secret, &w.Enabled, &w.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	if !includeSecret {
		w.Secret = ""
	}
	return &w, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Webhook, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, endpoint, events, enabled, created_at
		FROM webhooks WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Endpoint, &w.Events, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &w)
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

func (s *Service) Update(ctx context.Context, teamID, id uuid.UUID, endpoint string, events []string, enabled *bool) (*Webhook, error) {
	if endpoint != "" {
		_, _ = s.Pool.Exec(ctx, `UPDATE webhooks SET endpoint = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, endpoint, teamID)
	}
	if events != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE webhooks SET events = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, events, teamID)
	}
	if enabled != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE webhooks SET enabled = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *enabled, teamID)
	}
	return s.Get(ctx, teamID, id, false)
}

// Fanout creates webhook_events for matching subscriptions and enqueues delivery.
func (s *Service) Fanout(ctx context.Context, teamID uuid.UUID, eventType string, payload any) error {
	body, err := json.Marshal(map[string]any{
		"type":       eventType,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"data":       payload,
	})
	if err != nil {
		return err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM webhooks
		WHERE team_id = $1 AND enabled AND $2 = ANY(events)
	`, teamID, eventType)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var wid uuid.UUID
		if err := rows.Scan(&wid); err != nil {
			return err
		}
		var eid uuid.UUID
		err := s.Pool.QueryRow(ctx, `
			INSERT INTO webhook_events (team_id, webhook_id, type, payload)
			VALUES ($1,$2,$3,$4) RETURNING id
		`, teamID, wid, eventType, body).Scan(&eid)
		if err != nil {
			return err
		}
		task, err := jobs.NewWebhookDeliverTask(eid.String())
		if err != nil {
			return err
		}
		if _, err := jobs.Enqueue(s.Client, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Deliver(ctx context.Context, eventID uuid.UUID) error {
	var endpoint, secret string
	var payload []byte
	var webhookID uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		SELECT w.endpoint, w.secret, e.payload, e.webhook_id
		FROM webhook_events e
		JOIN webhooks w ON w.id = e.webhook_id
		WHERE e.id = $1
	`, eventID).Scan(&endpoint, &secret, &payload, &webhookID)
	if err != nil {
		return err
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := sign(secret, ts, payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Raisin-Signature", fmt.Sprintf("t=%s,v1=%s", ts, sig))
	req.Header.Set("User-Agent", "Raisin-Webhooks/1.0")

	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	success := false
	status := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		status = resp.StatusCode
		success = status >= 200 && status < 300
	}
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO webhook_attempts (webhook_event_id, status_code, success, error)
		VALUES ($1,$2,$3,$4)
	`, eventID, status, success, nullErr(errMsg))
	if !success {
		return fmt.Errorf("webhook delivery failed: status=%d err=%s", status, errMsg)
	}
	return nil
}

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func nullErr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type EventRow struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type AttemptRow struct {
	ID          uuid.UUID `json:"id"`
	StatusCode  *int      `json:"status_code"`
	Success     bool      `json:"success"`
	Error       *string   `json:"error,omitempty"`
	AttemptedAt time.Time `json:"attempted_at"`
}

func (s *Service) ListEvents(ctx context.Context, teamID, webhookID uuid.UUID, limit int) ([]EventRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, type, payload, created_at FROM webhook_events
		WHERE team_id = $1 AND webhook_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, teamID, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventRow
	for rows.Next() {
		var e EventRow
		if err := rows.Scan(&e.ID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Service) ListAttempts(ctx context.Context, teamID, webhookID, eventID uuid.UUID) ([]AttemptRow, error) {
	// ensure ownership
	var ok bool
	_ = s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM webhook_events e
			JOIN webhooks w ON w.id = e.webhook_id
			WHERE e.id = $1 AND e.webhook_id = $2 AND w.team_id = $3
		)
	`, eventID, webhookID, teamID).Scan(&ok)
	if !ok {
		return nil, apierr.NotFound
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, status_code, success, error, attempted_at
		FROM webhook_attempts WHERE webhook_event_id = $1
		ORDER BY attempted_at DESC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttemptRow
	for rows.Next() {
		var a AttemptRow
		if err := rows.Scan(&a.ID, &a.StatusCode, &a.Success, &a.Error, &a.AttemptedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// VerifySignature checks Raisin-Signature: t=<unix>,v1=<hex>
func VerifySignature(secret, header string, body []byte, maxSkew time.Duration) bool {
	parts := map[string]string{}
	for _, p := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}
	ts, sig := parts["t"], parts["v1"]
	if ts == "" || sig == "" {
		return false
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if maxSkew <= 0 {
		maxSkew = 5 * time.Minute
	}
	if abs64(time.Now().Unix()-sec) > int64(maxSkew.Seconds()) {
		return false
	}
	expected := sign(secret, ts, body)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
