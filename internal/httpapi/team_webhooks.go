package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/raisin-run/raisin/internal/apierr"
)

func (s *Server) listWebhookEvents(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := s.Webhooks.ListEvents(r.Context(), team.ID, id, 50)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) listWebhookAttempts(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	wid, ok := parseID(w, r)
	if !ok {
		return
	}
	eid, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid event id"))
		return
	}
	list, err := s.Webhooks.ListAttempts(r.Context(), team.ID, wid, eid)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getTeam(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var name, slug, billing string
	var testMode bool
	var quota int
	err := s.Pool.QueryRow(r.Context(), `
		SELECT name, slug, test_mode, monthly_quota, billing_status FROM teams WHERE id = $1
	`, team.ID).Scan(&name, &slug, &testMode, &quota, &billing)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"id":             team.ID,
		"name":           name,
		"slug":           slug,
		"test_mode":      testMode,
		"monthly_quota":  quota,
		"billing_status": billing,
	})
}

func (s *Server) updateTeam(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Name     *string `json:"name"`
		TestMode *bool   `json:"test_mode"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	if body.Name != nil && *body.Name != "" {
		if _, err := s.Pool.Exec(r.Context(), `UPDATE teams SET name = $2, updated_at = now() WHERE id = $1`, team.ID, *body.Name); err != nil {
			writeErr(w, err)
			return
		}
	}
	if body.TestMode != nil {
		if _, err := s.Pool.Exec(r.Context(), `UPDATE teams SET test_mode = $2, updated_at = now() WHERE id = $1`, team.ID, *body.TestMode); err != nil {
			writeErr(w, err)
			return
		}
	}
	s.getTeam(w, r)
}
