package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Pool *db.Pool
}

type App struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"` // only on create
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateAppRequest struct {
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(prefix string, n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) CreateApp(ctx context.Context, teamID uuid.UUID, req CreateAppRequest) (*App, error) {
	if req.Name == "" || len(req.RedirectURIs) == 0 {
		return nil, apierr.Validation("name and redirect_uris are required")
	}
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"emails:send", "emails:read", "domains:read"}
	}
	clientID, err := randomToken("ra_app_", 18)
	if err != nil {
		return nil, err
	}
	secret, err := randomToken("ra_sec_", 24)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO oauth_apps (team_id, name, client_id, client_secret_hash, redirect_uris, scopes)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id
	`, teamID, req.Name, clientID, hashToken(secret), req.RedirectURIs, scopes).Scan(&id)
	if err != nil {
		return nil, err
	}
	app, err := s.GetApp(ctx, teamID, id)
	if err != nil {
		return nil, err
	}
	app.ClientSecret = secret
	return app, nil
}

func (s *Service) GetApp(ctx context.Context, teamID, id uuid.UUID) (*App, error) {
	var a App
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, client_id, redirect_uris, scopes, created_at
		FROM oauth_apps WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&a.ID, &a.Name, &a.ClientID, &a.RedirectURIs, &a.Scopes, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &a, err
}

func (s *Service) ListApps(ctx context.Context, teamID uuid.UUID) ([]*App, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, client_id, redirect_uris, scopes, created_at
		FROM oauth_apps WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*App
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.Name, &a.ClientID, &a.RedirectURIs, &a.Scopes, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, nil
}

func (s *Service) DeleteApp(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM oauth_apps WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

type AuthorizeRequest struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
	TeamID      uuid.UUID
}

func (s *Service) CreateAuthorizationCode(ctx context.Context, req AuthorizeRequest) (code string, err error) {
	var appID uuid.UUID
	var redirects []string
	var appScopes []string
	err = s.Pool.QueryRow(ctx, `
		SELECT id, redirect_uris, scopes FROM oauth_apps WHERE client_id = $1
	`, req.ClientID).Scan(&appID, &redirects, &appScopes)
	if err == pgx.ErrNoRows {
		return "", apierr.Validation("invalid client_id")
	}
	if err != nil {
		return "", err
	}
	ok := false
	for _, u := range redirects {
		if u == req.RedirectURI {
			ok = true
			break
		}
	}
	if !ok {
		return "", apierr.Validation("redirect_uri mismatch")
	}
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = appScopes
	}
	code, err = randomToken("ra_code_", 24)
	if err != nil {
		return "", err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO oauth_authorization_codes (app_id, team_id, code_hash, redirect_uri, scopes, expires_at)
		VALUES ($1,$2,$3,$4,$5, now() + interval '10 minutes')
	`, appID, req.TeamID, hashToken(code), req.RedirectURI, scopes)
	return code, err
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

func (s *Service) ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	var appID, teamID uuid.UUID
	var secretHash string
	err := s.Pool.QueryRow(ctx, `
		SELECT id, team_id, client_secret_hash FROM oauth_apps WHERE client_id = $1
	`, clientID).Scan(&appID, &teamID, &secretHash)
	if err == pgx.ErrNoRows || hashToken(clientSecret) != secretHash {
		return nil, apierr.Unauthorized
	}
	if err != nil {
		return nil, err
	}
	var codeID uuid.UUID
	var scopes []string
	var used *time.Time
	var expires time.Time
	var storedRedirect string
	err = s.Pool.QueryRow(ctx, `
		SELECT id, scopes, used_at, expires_at, redirect_uri, team_id
		FROM oauth_authorization_codes WHERE code_hash = $1 AND app_id = $2
	`, hashToken(code), appID).Scan(&codeID, &scopes, &used, &expires, &storedRedirect, &teamID)
	if err == pgx.ErrNoRows {
		return nil, apierr.Validation("invalid code")
	}
	if err != nil {
		return nil, err
	}
	if used != nil || time.Now().After(expires) || storedRedirect != redirectURI {
		return nil, apierr.Validation("code expired or invalid")
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE oauth_authorization_codes SET used_at = now() WHERE id = $1`, codeID)
	return s.issueTokens(ctx, appID, teamID, scopes)
}

// RefreshExchange rotates a refresh token into a new access + refresh pair.
func (s *Service) RefreshExchange(ctx context.Context, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	var appID uuid.UUID
	var secretHash string
	err := s.Pool.QueryRow(ctx, `
		SELECT id, client_secret_hash FROM oauth_apps WHERE client_id = $1
	`, clientID).Scan(&appID, &secretHash)
	if err == pgx.ErrNoRows || hashToken(clientSecret) != secretHash {
		return nil, apierr.Unauthorized
	}
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(refreshToken)
	raw = strings.TrimPrefix(raw, "Bearer ")
	if !strings.HasPrefix(raw, "ra_rtk_") {
		return nil, apierr.Validation("invalid refresh_token")
	}
	var tokenID, teamID uuid.UUID
	var scopes []string
	var expires time.Time
	var revoked *time.Time
	err = s.Pool.QueryRow(ctx, `
		SELECT id, team_id, scopes, expires_at, revoked_at
		FROM oauth_refresh_tokens WHERE token_hash = $1 AND app_id = $2
	`, hashToken(raw), appID).Scan(&tokenID, &teamID, &scopes, &expires, &revoked)
	if err == pgx.ErrNoRows {
		return nil, apierr.Unauthorized
	}
	if err != nil {
		return nil, err
	}
	if revoked != nil || time.Now().After(expires) {
		return nil, apierr.Unauthorized
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at = now() WHERE id = $1`, tokenID)
	return s.issueTokens(ctx, appID, teamID, scopes)
}

func (s *Service) issueTokens(ctx context.Context, appID, teamID uuid.UUID, scopes []string) (*TokenResponse, error) {
	access, err := randomToken("ra_atk_", 32)
	if err != nil {
		return nil, err
	}
	refresh, err := randomToken("ra_rtk_", 32)
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO oauth_access_tokens (app_id, team_id, token_hash, scopes, expires_at)
		VALUES ($1,$2,$3,$4, now() + interval '1 hour')
	`, appID, teamID, hashToken(access), scopes)
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO oauth_refresh_tokens (app_id, team_id, token_hash, scopes, expires_at)
		VALUES ($1,$2,$3,$4, now() + interval '30 days')
	`, appID, teamID, hashToken(refresh), scopes)
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refresh,
		Scope:        strings.Join(scopes, " "),
	}, nil
}

// LookupAccessToken resolves a Bearer OAuth access token to a team and scopes.
func (s *Service) LookupAccessToken(ctx context.Context, authorization string) (*auth.Team, []string, error) {
	raw := strings.TrimSpace(authorization)
	raw = strings.TrimPrefix(raw, "Bearer ")
	raw = strings.TrimPrefix(raw, "bearer ")
	if !strings.HasPrefix(raw, "ra_atk_") {
		return nil, nil, apierr.Unauthorized
	}
	var teamID uuid.UUID
	var expires time.Time
	var scopes []string
	err := s.Pool.QueryRow(ctx, `
		SELECT team_id, expires_at, scopes FROM oauth_access_tokens WHERE token_hash = $1
	`, hashToken(raw)).Scan(&teamID, &expires, &scopes)
	if err == pgx.ErrNoRows {
		return nil, nil, apierr.Unauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	if time.Now().After(expires) {
		return nil, nil, apierr.Unauthorized
	}
	team, err := auth.LoadTeam(ctx, s.Pool, teamID)
	if err != nil {
		return nil, nil, err
	}
	return team, scopes, nil
}

// PublicApp returns non-secret metadata for the consent screen.
func (s *Service) PublicApp(ctx context.Context, clientID string) (*App, error) {
	var a App
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, client_id, redirect_uris, scopes, created_at
		FROM oauth_apps WHERE client_id = $1
	`, clientID).Scan(&a.ID, &a.Name, &a.ClientID, &a.RedirectURIs, &a.Scopes, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	return &a, err
}
