package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raisin-run/raisin/internal/apierr"
	"github.com/raisin-run/raisin/internal/auth"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Server) listEmailEvents(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := s.Emails.ListEvents(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

// consoleProvision upserts a user+personal team from Better Auth and returns a team JWT.
func (s *Server) consoleProvision(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Secret string `json:"secret"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	if body.Secret != s.Cfg.JWTSecret {
		apierr.Write(w, apierr.Forbidden)
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || !strings.Contains(email, "@") {
		apierr.Write(w, apierr.Validation("email is required"))
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	ctx := r.Context()
	var userID uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		err = s.Pool.QueryRow(ctx, `
			INSERT INTO users (email, name, email_verified) VALUES ($1,$2,TRUE) RETURNING id
		`, email, name).Scan(&userID)
		if err != nil {
			writeErr(w, err)
			return
		}
	}

	var teamID uuid.UUID
	var role string
	err = s.Pool.QueryRow(ctx, `
		SELECT t.id, m.role FROM members m
		JOIN teams t ON t.id = m.team_id
		WHERE m.user_id = $1
		ORDER BY m.created_at ASC LIMIT 1
	`, userID).Scan(&teamID, &role)
	if err != nil {
		slug := slugify(name)
		base := slug
		for i := 0; i < 5; i++ {
			err = s.Pool.QueryRow(ctx, `
				INSERT INTO teams (name, slug, test_mode, monthly_quota)
				VALUES ($1,$2,TRUE,$3) RETURNING id
			`, name+"'s Team", slug, s.Cfg.DefaultMonthlyQuota).Scan(&teamID)
			if err == nil {
				break
			}
			slug = base + "-" + uuid.NewString()[:8]
		}
		if err != nil {
			writeErr(w, err)
			return
		}
		_, err = s.Pool.Exec(ctx, `
			INSERT INTO members (team_id, user_id, role) VALUES ($1,$2,'owner')
		`, teamID, userID)
		if err != nil {
			writeErr(w, err)
			return
		}
		role = "owner"
	}

	tok, err := auth.IssueConsoleJWT(s.Cfg.JWTSecret, teamID, userID, role, 24*time.Hour)
	if err != nil {
		writeErr(w, err)
		return
	}
	slog.Info("console provisioned", "email", email, "team_id", teamID)
	apierr.WriteJSON(w, 200, map[string]any{
		"token":   tok,
		"team_id": teamID,
		"user_id": userID,
		"role":    role,
		"email":   email,
	})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "team"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
