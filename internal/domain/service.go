package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/sender"
)

type Service struct {
	Pool     *db.Pool
	Identity sender.IdentityManager
}

type Domain struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Region            string     `json:"region"`
	Status            string     `json:"status"`
	OpenTracking      bool       `json:"open_tracking"`
	ClickTracking     bool       `json:"click_tracking"`
	TrackingSubdomain string     `json:"tracking_subdomain"`
	ClaimedAt         *time.Time `json:"claimed_at,omitempty"`
	ReceivingEnabled  bool       `json:"receiving_enabled"`
	BIMISelector      string     `json:"bimi_selector,omitempty"`
	BIMISVGURL        *string    `json:"bimi_svg_url,omitempty"`
	BIMIVMCURL        *string    `json:"bimi_vmc_url,omitempty"`
	IPPoolID          *uuid.UUID `json:"ip_pool_id,omitempty"`
	Records           []Record   `json:"records"`
	CreatedAt         time.Time  `json:"created_at"`
}

type Record struct {
	Record   string  `json:"record"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Value    string  `json:"value"`
	TTL      string  `json:"ttl"`
	Status   string  `json:"status"`
	Priority *int    `json:"priority,omitempty"`
}

type CreateRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

type UpdateRequest struct {
	OpenTracking  *bool `json:"open_tracking"`
	ClickTracking *bool `json:"click_tracking"`
}

func (s *Service) Create(ctx context.Context, team *auth.Team, req CreateRequest) (*Domain, error) {
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if name == "" {
		return nil, apierr.Validation("name is required")
	}
	region := NormalizeRegion(req.Region)

	identity, err := s.Identity.CreateDomain(ctx, name, region)
	if err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}

	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO domains (team_id, name, region, status)
		VALUES ($1,$2,$3,'pending')
		RETURNING id
	`, team.ID, name, region).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, r := range identity.DKIM {
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO domain_records (domain_id, record_type, name, type, value, status)
			VALUES ($1,'DKIM',$2,$3,$4,'not_started')
		`, id, r.Name, r.Type, r.Value)
	}
	for i, r := range identity.SPF {
		prio := (*int)(nil)
		if r.Type == "MX" {
			p := 10
			prio = &p
		}
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO domain_records (domain_id, record_type, name, type, value, priority, status)
			VALUES ($1,'SPF',$2,$3,$4,$5,'not_started')
		`, id, r.Name, r.Type, r.Value, prio)
		_ = i
	}
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO domain_records (domain_id, record_type, name, type, value, status)
		VALUES ($1,'Tracking',$2,'CNAME',$3,'not_started')
	`, id, "links."+name, "track.raisin.run")

	return s.Get(ctx, team.ID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Domain, error) {
	var d Domain
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, region, status, open_tracking, click_tracking, tracking_subdomain,
		       claimed_at, receiving_enabled, bimi_selector, bimi_svg_url, bimi_vmc_url, ip_pool_id, created_at
		FROM domains WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(
		&d.ID, &d.Name, &d.Region, &d.Status, &d.OpenTracking, &d.ClickTracking, &d.TrackingSubdomain,
		&d.ClaimedAt, &d.ReceivingEnabled, &d.BIMISelector, &d.BIMISVGURL, &d.BIMIVMCURL, &d.IPPoolID, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	recs, err := s.records(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.Records = recs
	return &d, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Domain, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, region, status, open_tracking, click_tracking, tracking_subdomain,
		       claimed_at, receiving_enabled, bimi_selector, bimi_svg_url, bimi_vmc_url, ip_pool_id, created_at
		FROM domains WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Region, &d.Status, &d.OpenTracking, &d.ClickTracking, &d.TrackingSubdomain,
			&d.ClaimedAt, &d.ReceivingEnabled, &d.BIMISelector, &d.BIMISVGURL, &d.BIMIVMCURL, &d.IPPoolID, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		recs, _ := s.records(ctx, d.ID)
		d.Records = recs
		out = append(out, &d)
	}
	return out, nil
}

func (s *Service) Verify(ctx context.Context, teamID, id uuid.UUID) (*Domain, error) {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	identity, err := s.Identity.VerifyDomain(ctx, d.Name, d.Region)
	if err != nil {
		return nil, err
	}
	status := "pending"
	recStatus := "pending"
	if identity.Verified {
		status = "verified"
		recStatus = "verified"
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE domains SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	_, _ = s.Pool.Exec(ctx, `UPDATE domain_records SET status = $2 WHERE domain_id = $1`, id, recStatus)
	return s.Get(ctx, teamID, id)
}

func (s *Service) Update(ctx context.Context, teamID, id uuid.UUID, req UpdateRequest) (*Domain, error) {
	if req.OpenTracking != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE domains SET open_tracking = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *req.OpenTracking, teamID)
	}
	if req.ClickTracking != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE domains SET click_tracking = $2, updated_at = now() WHERE id = $1 AND team_id = $3`, id, *req.ClickTracking, teamID)
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return err
	}
	_ = s.Identity.DeleteDomain(ctx, d.Name, d.Region)
	_, err = s.Pool.Exec(ctx, `DELETE FROM domains WHERE id = $1 AND team_id = $2`, id, teamID)
	return err
}

func (s *Service) records(ctx context.Context, domainID uuid.UUID) ([]Record, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT record_type, name, type, value, ttl, status, priority
		FROM domain_records WHERE domain_id = $1
	`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Record, &r.Name, &r.Type, &r.Value, &r.TTL, &r.Status, &r.Priority); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
