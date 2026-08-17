package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/blakebauman/raisin/internal/db"
)

type Team struct {
	ID            uuid.UUID
	Name          string
	Slug          string
	TestMode      bool
	MonthlyQuota  int
	BillingStatus string
}

type APIKey struct {
	ID         uuid.UUID
	TeamID     uuid.UUID
	Name       string
	Prefix     string
	Permission string
}

type ContextKey string

const TeamKey ContextKey = "team"
const APIKeyCtxKey ContextKey = "api_key"
const OAuthScopesKey ContextKey = "oauth_scopes"

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func GenerateAPIKey() (raw string, prefix string, hash string) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	raw = "ra_" + hex.EncodeToString(b)
	prefix = raw[:10]
	hash = HashKey(raw)
	return raw, prefix, hash
}

func LookupAPIKey(ctx context.Context, pool *db.Pool, raw string) (*Team, *APIKey, error) {
	if raw == "" {
		return nil, nil, fmt.Errorf("empty key")
	}
	raw = strings.TrimPrefix(raw, "Bearer ")
	raw = strings.TrimSpace(raw)
	hash := HashKey(raw)
	var key APIKey
	var team Team
	err := pool.QueryRow(ctx, `
		SELECT k.id, k.team_id, k.name, k.prefix, k.permission,
		       t.id, t.name, t.slug, t.test_mode, t.monthly_quota, t.billing_status
		FROM api_keys k
		JOIN teams t ON t.id = k.team_id
		WHERE k.key_hash = $1
	`, hash).Scan(
		&key.ID, &key.TeamID, &key.Name, &key.Prefix, &key.Permission,
		&team.ID, &team.Name, &team.Slug, &team.TestMode, &team.MonthlyQuota, &team.BillingStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("invalid key")
		}
		return nil, nil, err
	}
	_, _ = pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, key.ID)
	return &team, &key, nil
}

type ConsoleClaims struct {
	TeamID uuid.UUID `json:"team_id"`
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

func IssueConsoleJWT(secret string, teamID, userID uuid.UUID, role string, ttl time.Duration) (string, error) {
	claims := ConsoleClaims{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "raisin-console",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseConsoleJWT(secret, token string) (*ConsoleClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &ConsoleClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*ConsoleClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func LoadTeam(ctx context.Context, pool *db.Pool, teamID uuid.UUID) (*Team, error) {
	var team Team
	err := pool.QueryRow(ctx, `
		SELECT id, name, slug, test_mode, monthly_quota, billing_status
		FROM teams WHERE id = $1
	`, teamID).Scan(&team.ID, &team.Name, &team.Slug, &team.TestMode, &team.MonthlyQuota, &team.BillingStatus)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func TeamFromContext(ctx context.Context) *Team {
	t, _ := ctx.Value(TeamKey).(*Team)
	return t
}

func ScopesFromContext(ctx context.Context) []string {
	s, _ := ctx.Value(OAuthScopesKey).([]string)
	return s
}

// HasScope returns true when no OAuth scopes are bound (API key / console JWT)
// or when the required scope is present.
func HasScope(ctx context.Context, required string) bool {
	scopes := ScopesFromContext(ctx)
	if len(scopes) == 0 {
		return true
	}
	for _, s := range scopes {
		if s == required || s == "*" || s == "full_access" {
			return true
		}
	}
	return false
}
