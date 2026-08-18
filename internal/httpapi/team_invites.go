package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/auth"
)

type teamMember struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Name      *string   `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type teamInvite struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	InviteURL  string     `json:"invite_url,omitempty"`
	Token      string     `json:"token,omitempty"`
}

func (s *Server) consoleClaimsOrWrite(w http.ResponseWriter, r *http.Request) *auth.ConsoleClaims {
	claims := auth.ConsoleClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.New(http.StatusForbidden, "console_required", "This endpoint requires a console session (X-Team-Token)"))
		return nil
	}
	return claims
}

func (s *Server) requireTeamAdmin(w http.ResponseWriter, r *http.Request) *auth.ConsoleClaims {
	claims := s.consoleClaimsOrWrite(w, r)
	if claims == nil {
		return nil
	}
	if !auth.IsTeamAdmin(claims.Role) {
		apierr.Write(w, apierr.New(http.StatusForbidden, "forbidden", "Owner or admin role required"))
		return nil
	}
	return claims
}

func inviteURL(origin, token string) string {
	origin = strings.TrimRight(origin, "/")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return origin + "/invite?token=" + token
}

func generateInviteToken() (raw, hash string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = "ri_" + hex.EncodeToString(b)
	hash = auth.HashKey(raw)
	return raw, hash, nil
}

func (s *Server) listTeamMembers(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	if s.consoleClaimsOrWrite(w, r) == nil {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `
		SELECT m.id, m.user_id, u.email, u.name, m.role, m.created_at
		FROM members m
		JOIN users u ON u.id = m.user_id
		WHERE m.team_id = $1
		ORDER BY
			CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
			m.created_at ASC
	`, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()
	out := []teamMember{}
	for rows.Next() {
		var m teamMember
		if err := rows.Scan(&m.ID, &m.UserID, &m.Email, &m.Name, &m.Role, &m.CreatedAt); err != nil {
			writeErr(w, err)
			return
		}
		out = append(out, m)
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) updateTeamMember(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	claims := s.consoleClaimsOrWrite(w, r)
	if claims == nil {
		return
	}
	if claims.Role != "owner" {
		apierr.Write(w, apierr.New(http.StatusForbidden, "forbidden", "Only the owner can change member roles"))
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role != "admin" && role != "member" {
		apierr.Write(w, apierr.Validation("role must be admin or member"))
		return
	}

	var userID uuid.UUID
	var currentRole string
	err := s.Pool.QueryRow(r.Context(), `
		SELECT user_id, role FROM members WHERE id = $1 AND team_id = $2
	`, id, team.ID).Scan(&userID, &currentRole)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierr.Write(w, apierr.NotFound)
			return
		}
		writeErr(w, err)
		return
	}
	if currentRole == "owner" {
		apierr.Write(w, apierr.Validation("cannot change the owner role"))
		return
	}

	tag, err := s.Pool.Exec(r.Context(), `
		UPDATE members SET role = $3 WHERE id = $1 AND team_id = $2 AND role <> 'owner'
	`, id, team.ID, role)
	if err != nil {
		writeErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		apierr.Write(w, apierr.NotFound)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"id": id, "user_id": userID, "role": role})
}

func (s *Server) removeTeamMember(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	claims := s.consoleClaimsOrWrite(w, r)
	if claims == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var userID uuid.UUID
	var role string
	err := s.Pool.QueryRow(r.Context(), `
		SELECT user_id, role FROM members WHERE id = $1 AND team_id = $2
	`, id, team.ID).Scan(&userID, &role)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierr.Write(w, apierr.NotFound)
			return
		}
		writeErr(w, err)
		return
	}
	if role == "owner" {
		apierr.Write(w, apierr.Validation("cannot remove the team owner"))
		return
	}
	leavingSelf := userID == claims.UserID
	if !leavingSelf {
		if !auth.IsTeamAdmin(claims.Role) {
			apierr.Write(w, apierr.New(http.StatusForbidden, "forbidden", "Owner or admin role required"))
			return
		}
		if claims.Role == "admin" && role == "admin" {
			apierr.Write(w, apierr.New(http.StatusForbidden, "forbidden", "Admins cannot remove other admins"))
			return
		}
	}

	_, err = s.Pool.Exec(r.Context(), `DELETE FROM members WHERE id = $1 AND team_id = $2`, id, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"deleted": true})
}

func (s *Server) listTeamInvites(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	if s.requireTeamAdmin(w, r) == nil {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `
		SELECT id, email, role, expires_at, accepted_at, revoked_at, created_at
		FROM team_invites
		WHERE team_id = $1 AND accepted_at IS NULL AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()
	out := []teamInvite{}
	for rows.Next() {
		var inv teamInvite
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt, &inv.CreatedAt); err != nil {
			writeErr(w, err)
			return
		}
		out = append(out, inv)
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) createTeamInvite(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	claims := s.requireTeamAdmin(w, r)
	if claims == nil {
		return
	}
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || !strings.Contains(email, "@") {
		apierr.Write(w, apierr.Validation("email is required"))
		return
	}
	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role == "" {
		role = "member"
	}
	if role != "admin" && role != "member" {
		apierr.Write(w, apierr.Validation("role must be admin or member"))
		return
	}
	if claims.Role == "admin" && role == "admin" {
		apierr.Write(w, apierr.New(http.StatusForbidden, "forbidden", "Admins cannot invite other admins"))
		return
	}

	var exists bool
	err := s.Pool.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM members m
			JOIN users u ON u.id = m.user_id
			WHERE m.team_id = $1 AND u.email = $2
		)
	`, team.ID, email).Scan(&exists)
	if err != nil {
		writeErr(w, err)
		return
	}
	if exists {
		apierr.Write(w, apierr.Validation("user is already a team member"))
		return
	}

	raw, hash, err := generateInviteToken()
	if err != nil {
		writeErr(w, err)
		return
	}
	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	var inv teamInvite
	var invitedBy *uuid.UUID
	if claims.UserID != uuid.Nil {
		uid := claims.UserID
		invitedBy = &uid
	}
	err = s.Pool.QueryRow(r.Context(), `
		INSERT INTO team_invites (team_id, email, role, token_hash, invited_by, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, email, role, expires_at, created_at
	`, team.ID, email, role, hash, invitedBy, expires).Scan(
		&inv.ID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "team_invites_pending_email_uidx") {
			apierr.Write(w, apierr.Validation("an open invite already exists for this email"))
			return
		}
		writeErr(w, err)
		return
	}
	inv.Token = raw
	inv.InviteURL = inviteURL(s.Cfg.ConsoleOrigin, raw)
	apierr.WriteJSON(w, 201, inv)
}

func (s *Server) revokeTeamInvite(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	if s.requireTeamAdmin(w, r) == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	tag, err := s.Pool.Exec(r.Context(), `
		UPDATE team_invites
		SET revoked_at = now()
		WHERE id = $1 AND team_id = $2 AND accepted_at IS NULL AND revoked_at IS NULL
	`, id, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		apierr.Write(w, apierr.NotFound)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"revoked": true})
}

func (s *Server) previewTeamInvite(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		apierr.Write(w, apierr.Validation("token is required"))
		return
	}
	hash := auth.HashKey(token)
	var teamName, email, role string
	var expiresAt time.Time
	var acceptedAt, revokedAt *time.Time
	err := s.Pool.QueryRow(r.Context(), `
		SELECT t.name, i.email, i.role, i.expires_at, i.accepted_at, i.revoked_at
		FROM team_invites i
		JOIN teams t ON t.id = i.team_id
		WHERE i.token_hash = $1
	`, hash).Scan(&teamName, &email, &role, &expiresAt, &acceptedAt, &revokedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierr.Write(w, apierr.NotFound)
			return
		}
		writeErr(w, err)
		return
	}
	status := "pending"
	switch {
	case acceptedAt != nil:
		status = "accepted"
	case revokedAt != nil:
		status = "revoked"
	case time.Now().UTC().After(expiresAt):
		status = "expired"
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"team_name":  teamName,
		"email":      email,
		"role":       role,
		"expires_at": expiresAt,
		"status":     status,
	})
}

func (s *Server) acceptTeamInvite(w http.ResponseWriter, r *http.Request) {
	claims := s.consoleClaimsOrWrite(w, r)
	if claims == nil {
		return
	}
	if claims.UserID == uuid.Nil {
		apierr.Write(w, apierr.Validation("console session is missing user identity; sign in again"))
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	token := strings.TrimSpace(body.Token)
	if token == "" {
		apierr.Write(w, apierr.Validation("token is required"))
		return
	}
	hash := auth.HashKey(token)

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	var inviteID, teamID uuid.UUID
	var inviteEmail, role string
	var expiresAt time.Time
	var acceptedAt, revokedAt *time.Time
	err = tx.QueryRow(r.Context(), `
		SELECT id, team_id, email, role, expires_at, accepted_at, revoked_at
		FROM team_invites
		WHERE token_hash = $1
		FOR UPDATE
	`, hash).Scan(&inviteID, &teamID, &inviteEmail, &role, &expiresAt, &acceptedAt, &revokedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierr.Write(w, apierr.NotFound)
			return
		}
		writeErr(w, err)
		return
	}
	if acceptedAt != nil {
		apierr.Write(w, apierr.Validation("invite already accepted"))
		return
	}
	if revokedAt != nil {
		apierr.Write(w, apierr.Validation("invite was revoked"))
		return
	}
	if time.Now().UTC().After(expiresAt) {
		apierr.Write(w, apierr.Validation("invite has expired"))
		return
	}

	var userEmail string
	err = tx.QueryRow(r.Context(), `SELECT email FROM users WHERE id = $1`, claims.UserID).Scan(&userEmail)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !strings.EqualFold(userEmail, inviteEmail) {
		apierr.Write(w, apierr.New(http.StatusForbidden, "email_mismatch", "Sign in as "+inviteEmail+" to accept this invite"))
		return
	}

	var already bool
	err = tx.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM members WHERE team_id = $1 AND user_id = $2)
	`, teamID, claims.UserID).Scan(&already)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !already {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO members (team_id, user_id, role) VALUES ($1,$2,$3)
		`, teamID, claims.UserID, role)
		if err != nil {
			writeErr(w, err)
			return
		}
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE team_invites SET accepted_at = now() WHERE id = $1
	`, inviteID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeErr(w, err)
		return
	}

	tok, err := auth.IssueConsoleJWT(s.Cfg.JWTSecret, teamID, claims.UserID, role, 24*time.Hour)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"token":   tok,
		"team_id": teamID,
		"user_id": claims.UserID,
		"role":    role,
		"email":   userEmail,
	})
}
