package ippool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Pool *db.Pool
}

type Pool struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Region       string    `json:"region"`
	Status       string    `json:"status"`
	SESConfigSet *string   `json:"ses_config_set,omitempty"`
	IPs          []IP      `json:"ips,omitempty"`
	Warmup       *Warmup   `json:"warmup,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type IP struct {
	ID      uuid.UUID `json:"id"`
	Address string    `json:"address"`
	Status  string    `json:"status"`
}

type Warmup struct {
	DayIndex  int   `json:"day_index"`
	DailyCap  int   `json:"daily_cap"`
	SentToday int   `json:"sent_today"`
	Plan      []int `json:"plan"`
}

type CreateRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Pool, error) {
	if req.Name == "" {
		return nil, apierr.Validation("name is required")
	}
	region := req.Region
	if region == "" {
		region = "us-east-1"
	}
	configSet := fmt.Sprintf("raisin-%s-%s", teamID.String()[:8], sanitize(req.Name))
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO ip_pools (team_id, name, region, status, ses_config_set)
		VALUES ($1,$2,$3,'warming',$4) RETURNING id
	`, teamID, req.Name, region, configSet).Scan(&id)
	if err != nil {
		return nil, err
	}
	// Local/dev: synthesize a dedicated IP address placeholder
	addr := fmt.Sprintf("203.0.113.%d", (id[15]%200)+10)
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO dedicated_ips (pool_id, address, provider_ref, status)
		VALUES ($1,$2,$3,'assigned')
	`, id, addr, "local-"+id.String())
	plan := []int{50, 100, 200, 400, 800, 1600, 3200, 6400, 10000, 20000}
	planJSON, _ := json.Marshal(plan)
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO warmup_schedules (pool_id, day_index, daily_cap, plan)
		VALUES ($1, 0, 50, $2)
	`, id, planJSON)
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Pool, error) {
	var p Pool
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, region, status, ses_config_set, created_at
		FROM ip_pools WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&p.ID, &p.Name, &p.Region, &p.Status, &p.SESConfigSet, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, address, status FROM dedicated_ips WHERE pool_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ip IP
		if err := rows.Scan(&ip.ID, &ip.Address, &ip.Status); err != nil {
			return nil, err
		}
		p.IPs = append(p.IPs, ip)
	}
	var w Warmup
	var planRaw []byte
	err = s.Pool.QueryRow(ctx, `
		SELECT day_index, daily_cap, sent_today, plan FROM warmup_schedules WHERE pool_id = $1
	`, id).Scan(&w.DayIndex, &w.DailyCap, &w.SentToday, &planRaw)
	if err == nil {
		_ = json.Unmarshal(planRaw, &w.Plan)
		p.Warmup = &w
	}
	return &p, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Pool, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM ip_pools WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Pool
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := s.Get(ctx, teamID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) Pause(ctx context.Context, teamID, id uuid.UUID) (*Pool, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE ip_pools SET status = 'paused', updated_at = now() WHERE id = $1 AND team_id = $2
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Resume(ctx context.Context, teamID, id uuid.UUID) (*Pool, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE ip_pools SET status = 'warming', updated_at = now() WHERE id = $1 AND team_id = $2 AND status = 'paused'
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM ip_pools WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

// ReserveSend checks warmup caps for a pool; returns config set name if allowed.
func (s *Service) ReserveSend(ctx context.Context, poolID uuid.UUID) (configSet string, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var status string
	var cs *string
	err = tx.QueryRow(ctx, `SELECT status, ses_config_set FROM ip_pools WHERE id = $1 FOR UPDATE`, poolID).Scan(&status, &cs)
	if err == pgx.ErrNoRows {
		return "", apierr.NotFound
	}
	if err != nil {
		return "", err
	}
	if status == "paused" || status == "failed" {
		return "", apierr.Validation("ip pool is not active")
	}
	var dayIndex, dailyCap, sentToday int
	var lastReset time.Time
	var planRaw []byte
	err = tx.QueryRow(ctx, `
		SELECT day_index, daily_cap, sent_today, last_reset_at, plan
		FROM warmup_schedules WHERE pool_id = $1 FOR UPDATE
	`, poolID).Scan(&dayIndex, &dailyCap, &sentToday, &lastReset, &planRaw)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if lastReset.UTC().Truncate(24*time.Hour).Before(now.Truncate(24 * time.Hour)) {
		dayIndex++
		var plan []int
		_ = json.Unmarshal(planRaw, &plan)
		if dayIndex < len(plan) {
			dailyCap = plan[dayIndex]
		} else if len(plan) > 0 {
			dailyCap = plan[len(plan)-1]
			status = "active"
			_, _ = tx.Exec(ctx, `UPDATE ip_pools SET status = 'active' WHERE id = $1`, poolID)
		}
		sentToday = 0
		_, err = tx.Exec(ctx, `
			UPDATE warmup_schedules SET day_index = $2, daily_cap = $3, sent_today = 0,
			last_reset_at = date_trunc('day', now()), updated_at = now()
			WHERE pool_id = $1
		`, poolID, dayIndex, dailyCap)
		if err != nil {
			return "", err
		}
	}
	if sentToday >= dailyCap {
		return "", apierr.New(429, "warmup_cap", "dedicated IP daily warmup cap reached")
	}
	_, err = tx.Exec(ctx, `UPDATE warmup_schedules SET sent_today = sent_today + 1, updated_at = now() WHERE pool_id = $1`, poolID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if cs != nil {
		return *cs, nil
	}
	return "", nil
}

func (s *Service) AssignDomain(ctx context.Context, teamID, poolID, domainID uuid.UUID) error {
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ip_pools WHERE id = $1 AND team_id = $2)`, poolID, teamID).Scan(&ok)
	if err != nil || !ok {
		return apierr.NotFound
	}
	tag, err := s.Pool.Exec(ctx, `
		UPDATE domains SET ip_pool_id = $1, updated_at = now() WHERE id = $2 AND team_id = $3
	`, poolID, domainID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "pool"
	}
	return string(out)
}
