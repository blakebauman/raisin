package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/sender"
	"github.com/google/uuid"
)

// Claim starts exclusive ownership challenge via DNS TXT.
func (s *Service) Claim(ctx context.Context, teamID, id uuid.UUID) (*Domain, error) {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	if d.Status != "verified" {
		return nil, apierr.Validation("domain must be verified before claiming")
	}
	var taken bool
	err = s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM domains
			WHERE lower(name) = lower($1) AND claimed_at IS NOT NULL AND id <> $2 AND status = 'verified'
		)
	`, d.Name, id).Scan(&taken)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, apierr.Validation("domain is already claimed by another team")
	}
	tok := make([]byte, 16)
	_, _ = rand.Read(tok)
	token := "raisin-claim=" + hex.EncodeToString(tok)
	_, err = s.Pool.Exec(ctx, `
		UPDATE domains SET claim_token = $2, updated_at = now() WHERE id = $1 AND team_id = $3
	`, id, token, teamID)
	if err != nil {
		return nil, err
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM domain_records WHERE domain_id = $1 AND record_type = 'Claim'`, id)
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO domain_records (domain_id, record_type, name, type, value, status)
		VALUES ($1,'Claim',$2,'TXT',$3,'pending')
	`, id, "_raisin-challenge."+d.Name, fmt.Sprintf("%q", token))
	return s.Get(ctx, teamID, id)
}

// ConfirmClaim marks the domain claimed after the challenge TXT is present in DNS
// (stub identity skips live lookup for local/dev).
func (s *Service) ConfirmClaim(ctx context.Context, teamID, id uuid.UUID) (*Domain, error) {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	var token *string
	_ = s.Pool.QueryRow(ctx, `SELECT claim_token FROM domains WHERE id = $1 AND team_id = $2`, id, teamID).Scan(&token)
	if token == nil || *token == "" {
		return nil, apierr.Validation("start claim first via POST /domains/{id}/claim")
	}
	var taken bool
	_ = s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM domains
			WHERE lower(name) = lower($1) AND claimed_at IS NOT NULL AND id <> $2
		)
	`, d.Name, id).Scan(&taken)
	if taken {
		return nil, apierr.Validation("domain is already claimed by another team")
	}
	var claimRecs int
	_ = s.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM domain_records WHERE domain_id = $1 AND record_type = 'Claim'
	`, id).Scan(&claimRecs)
	if claimRecs == 0 {
		return nil, apierr.Validation("claim TXT record missing — call /claim first")
	}
	// SES/production: require live DNS TXT. Local stub identity skips the lookup.
	if _, stub := s.Identity.(sender.StubIdentity); !stub {
		host := "_raisin-challenge." + d.Name
		txts, lookupErr := net.LookupTXT(host)
		if lookupErr != nil {
			return nil, apierr.Validation("claim TXT not found in DNS yet")
		}
		found := false
		want := *token
		for _, t := range txts {
			t = strings.Trim(t, `"`)
			if t == want || strings.Contains(t, want) {
				found = true
				break
			}
		}
		if !found {
			return nil, apierr.Validation("claim TXT does not match challenge token")
		}
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE domains SET claimed_at = now(), updated_at = now() WHERE id = $1 AND team_id = $2
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE domain_records SET status = 'verified' WHERE domain_id = $1 AND record_type = 'Claim'`, id)
	return s.Get(ctx, teamID, id)
}

func (s *Service) ReleaseClaim(ctx context.Context, teamID, id uuid.UUID) (*Domain, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE domains SET claimed_at = NULL, claim_token = NULL, updated_at = now()
		WHERE id = $1 AND team_id = $2
	`, id, teamID)
	if err != nil {
		return nil, err
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM domain_records WHERE domain_id = $1 AND record_type = 'Claim'`, id)
	return s.Get(ctx, teamID, id)
}

type BIMIRequest struct {
	Selector string `json:"selector"`
	SVGURL   string `json:"svg_url"`
	VMCURL   string `json:"vmc_url"`
}

func (s *Service) SetBIMI(ctx context.Context, teamID, id uuid.UUID, req BIMIRequest) (*Domain, error) {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	if req.SVGURL == "" {
		return nil, apierr.Validation("svg_url is required")
	}
	sel := req.Selector
	if sel == "" {
		sel = "default"
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE domains SET bimi_selector = $2, bimi_svg_url = $3, bimi_vmc_url = $4, updated_at = now()
		WHERE id = $1 AND team_id = $5
	`, id, sel, req.SVGURL, nullStrPtr(req.VMCURL), teamID)
	if err != nil {
		return nil, err
	}
	val := fmt.Sprintf(`"v=BIMI1; l=%s;`, req.SVGURL)
	if req.VMCURL != "" {
		val += fmt.Sprintf(" a=%s;", req.VMCURL)
	}
	val += `"`
	_, _ = s.Pool.Exec(ctx, `DELETE FROM domain_records WHERE domain_id = $1 AND record_type = 'BIMI'`, id)
	_, _ = s.Pool.Exec(ctx, `
		INSERT INTO domain_records (domain_id, record_type, name, type, value, status)
		VALUES ($1,'BIMI',$2,'TXT',$3,'pending')
	`, id, sel+"._bimi."+d.Name, val)
	return s.Get(ctx, teamID, id)
}

func (s *Service) EnableReceiving(ctx context.Context, teamID, id uuid.UUID, enabled bool) (*Domain, error) {
	d, err := s.Get(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	if enabled && d.Status != "verified" {
		return nil, apierr.Validation("domain must be verified to enable receiving")
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE domains SET receiving_enabled = $2, updated_at = now() WHERE id = $1 AND team_id = $3
	`, id, enabled, teamID)
	if err != nil {
		return nil, err
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM domain_records WHERE domain_id = $1 AND record_type = 'Inbound'`, id)
	if enabled {
		mx := fmt.Sprintf("inbound-smtp.%s.amazonaws.com", d.Region)
		prio := 10
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO domain_records (domain_id, record_type, name, type, value, priority, status)
			VALUES ($1,'Inbound',$2,'MX',$3,$4,'pending')
		`, id, d.Name, mx, prio)
	}
	return s.Get(ctx, teamID, id)
}

func nullStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func AllowedRegions() []string {
	return []string{"us-east-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-southeast-1", "ap-northeast-1"}
}

func NormalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return "us-east-1"
	}
	for _, r := range AllowedRegions() {
		if r == region {
			return region
		}
	}
	return "us-east-1"
}
