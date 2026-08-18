package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/broadcast"
	"github.com/blakebauman/raisin/internal/domain"
	"github.com/blakebauman/raisin/internal/email"
	"github.com/blakebauman/raisin/internal/jobs"
	"github.com/blakebauman/raisin/internal/template"
	"github.com/blakebauman/raisin/internal/webhook"
)

func (s *Server) sendEmail(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var raw map[string]any
	if err := decode(r, &raw); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	req := email.SendRequest{
		From:    str(raw["from"]),
		To:      normalizeRecipients(raw["to"]),
		Cc:      normalizeRecipients(raw["cc"]),
		Bcc:     normalizeRecipients(raw["bcc"]),
		ReplyTo: normalizeRecipients(raw["reply_to"]),
		Subject: str(raw["subject"]),
		HTML:    str(raw["html"]),
		Text:    str(raw["text"]),
	}
	if h, ok := raw["headers"].(map[string]any); ok {
		req.Headers = map[string]string{}
		for k, v := range h {
			req.Headers[k] = str(v)
		}
	}
	if t, ok := raw["tags"].(map[string]any); ok {
		req.Tags = map[string]string{}
		for k, v := range t {
			req.Tags[k] = str(v)
		}
	}
	if sched, ok := raw["scheduled_at"].(string); ok && sched != "" {
		if ts, err := time.Parse(time.RFC3339, sched); err == nil {
			req.ScheduledAt = &ts
		}
	}
	if tid, ok := raw["template_id"].(string); ok && tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			req.TemplateID = &id
		}
	}
	if atts, ok := raw["attachments"].([]any); ok {
		for _, item := range atts {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			req.Attachments = append(req.Attachments, email.AttachmentIn{
				Filename:    str(m["filename"]),
				ContentType: str(m["content_type"]),
				Content:     str(m["content"]),
				ContentID:   str(m["content_id"]),
			})
		}
	}
	e, err := s.Emails.Send(r.Context(), team, req, strHeader(r, "Idempotency-Key"))
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, e)
}

func (s *Server) sendBatch(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var items []map[string]any
	if err := decode(r, &items); err != nil {
		apierr.Write(w, apierr.Validation("invalid json array"))
		return
	}
	reqs := make([]email.SendRequest, 0, len(items))
	for _, raw := range items {
		reqs = append(reqs, email.SendRequest{
			From: str(raw["from"]), To: normalizeRecipients(raw["to"]),
			Cc: normalizeRecipients(raw["cc"]), Bcc: normalizeRecipients(raw["bcc"]),
			ReplyTo: normalizeRecipients(raw["reply_to"]), Subject: str(raw["subject"]),
			HTML: str(raw["html"]), Text: str(raw["text"]),
		})
	}
	out, err := s.Emails.SendBatch(r.Context(), team, reqs)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) getEmail(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := s.Emails.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, e)
}

func (s *Server) listEmails(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, next, err := s.Emails.List(r.Context(), team.ID, 20, r.URL.Query().Get("cursor"))
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list, "next_cursor": next})
}

func (s *Server) cancelEmail(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := s.Emails.Cancel(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, e)
}

func (s *Server) updateEmail(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	e, err := s.Emails.UpdateSchedule(r.Context(), team.ID, id, body.ScheduledAt)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, e)
}

func (s *Server) emailMetrics(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	m, err := s.Metrics.TeamEmailMetrics(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, m)
}

func (s *Server) createDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req domain.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	d, err := s.Domains.Create(r.Context(), team, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Domains.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	d, err := s.Domains.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) verifyDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	d, err := s.Domains.Verify(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.Asynq != nil {
		if task, err := jobs.NewDomainVerifyTask(id.String(), team.ID.String()); err == nil {
			_, _ = s.Asynq.Enqueue(task)
		}
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req domain.UpdateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	d, err := s.Domains.Update(r.Context(), team.ID, id, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Domains.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Name       string `json:"name"`
		Permission string `json:"permission"`
	}
	if err := decode(r, &body); err != nil || body.Name == "" {
		apierr.Write(w, apierr.Validation("name is required"))
		return
	}
	if body.Permission == "" {
		body.Permission = "full_access"
	}
	raw, prefix, hash := auth.GenerateAPIKey()
	var id uuid.UUID
	err := s.Pool.QueryRow(r.Context(), `
		INSERT INTO api_keys (team_id, name, prefix, key_hash, permission)
		VALUES ($1,$2,$3,$4,$5) RETURNING id
	`, team.ID, body.Name, prefix, hash, body.Permission).Scan(&id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"id": id, "name": body.Name, "prefix": prefix, "permission": body.Permission, "token": raw,
	})
}

func (s *Server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `
		SELECT id, name, prefix, permission, last_used_at, created_at FROM api_keys WHERE team_id = $1 ORDER BY created_at DESC
	`, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var name, prefix, perm string
		var last *time.Time
		var created time.Time
		if err := rows.Scan(&id, &name, &prefix, &perm, &last, &created); err != nil {
			writeErr(w, err)
			return
		}
		list = append(list, map[string]any{
			"id": id, "name": name, "prefix": prefix, "permission": perm,
			"last_used_at": last, "created_at": created,
		})
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	_, err := s.Pool.Exec(r.Context(), `DELETE FROM api_keys WHERE id = $1 AND team_id = $2`, id, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req webhook.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	wh, err := s.Webhooks.Create(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, wh)
}

func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Webhooks.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getWebhook(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	wh, err := s.Webhooks.Get(r.Context(), team.ID, id, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, wh)
}

func (s *Server) updateWebhook(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Endpoint string   `json:"endpoint"`
		Events   []string `json:"events"`
		Enabled  *bool    `json:"enabled"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	wh, err := s.Webhooks.Update(r.Context(), team.ID, id, body.Endpoint, body.Events, body.Enabled)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, wh)
}

func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Webhooks.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) addSuppression(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Email  string `json:"email"`
		Reason string `json:"reason"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	sp, err := s.Suppressions.Add(r.Context(), team.ID, body.Email, body.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, sp)
}

func (s *Server) addSuppressionsBatch(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Emails []string `json:"emails"`
		Reason string   `json:"reason"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	n, err := s.Suppressions.AddBatch(r.Context(), team.ID, body.Emails, body.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]int{"added": n})
}

func (s *Server) listSuppressions(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Suppressions.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) removeSuppression(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.Suppressions.Remove(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) createContact(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Email      string         `json:"email"`
		FirstName  string         `json:"first_name"`
		LastName   string         `json:"last_name"`
		Properties map[string]any `json:"properties"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	c, err := s.Audience.CreateContact(r.Context(), team.ID, body.Email, body.FirstName, body.LastName, body.Properties)
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.Automations != nil {
		_ = s.Automations.Trigger(r.Context(), team.ID, "contact.created", &c.ID, nil, nil, map[string]any{
			"contact_id": c.ID.String(),
			"email":      c.Email,
		})
	}
	apierr.WriteJSON(w, 200, c)
}

func (s *Server) listContacts(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Audience.ListContacts(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getContact(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := s.Audience.GetContact(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, c)
}

func (s *Server) updateContact(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Unsubscribed *bool   `json:"unsubscribed"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	c, err := s.Audience.UpdateContact(r.Context(), team.ID, id, body.FirstName, body.LastName, body.Unsubscribed)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, c)
}

func (s *Server) deleteContact(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Audience.DeleteContact(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) addContactSegment(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	cid, ok := parseID(w, r)
	if !ok {
		return
	}
	sid, err := uuid.Parse(chi.URLParam(r, "segmentId"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid segment id"))
		return
	}
	if err := s.Audience.AddToSegment(r.Context(), team.ID, sid, cid); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) removeContactSegment(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	cid, ok := parseID(w, r)
	if !ok {
		return
	}
	sid, err := uuid.Parse(chi.URLParam(r, "segmentId"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid segment id"))
		return
	}
	if err := s.Audience.RemoveFromSegment(r.Context(), team.ID, sid, cid); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) createProperty(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Key  string `json:"key"`
		Type string `json:"type"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	p, err := s.Audience.CreateProperty(r.Context(), team.ID, body.Key, body.Type)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) listProperties(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Audience.ListProperties(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) createSegment(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	seg, err := s.Audience.CreateSegment(r.Context(), team.ID, body.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, seg)
}

func (s *Server) listSegments(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Audience.ListSegments(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) deleteSegment(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Audience.DeleteSegment(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) createTopic(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		Name                 string `json:"name"`
		Description          string `json:"description"`
		DefaultSubscription  string `json:"default_subscription"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	t, err := s.Audience.CreateTopic(r.Context(), team.ID, body.Name, body.Description, body.DefaultSubscription)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) listTopics(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Audience.ListTopics(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getTopic(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.Audience.GetTopic(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) deleteTopic(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Audience.DeleteTopic(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) listContactTopics(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	cid, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := s.Audience.ListContactTopics(r.Context(), team.ID, cid)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) setContactTopic(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	cid, ok := parseID(w, r)
	if !ok {
		return
	}
	topicID, err := uuid.Parse(chi.URLParam(r, "topicId"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid topic id"))
		return
	}
	var body struct {
		Subscribed *bool `json:"subscribed"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	subscribed := true
	if body.Subscribed != nil {
		subscribed = *body.Subscribed
	}
	if err := s.Audience.SetTopicSubscription(r.Context(), team.ID, cid, topicID, subscribed); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"ok": true, "subscribed": subscribed})
}

func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req template.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	t, err := s.Templates.Create(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Templates.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.Templates.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req template.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	t, err := s.Templates.Update(r.Context(), team.ID, id, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) publishTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.Templates.Publish(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) duplicateTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.Templates.Duplicate(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, t)
}

func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Templates.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) createBroadcast(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req broadcast.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	b, err := s.Broadcasts.Create(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, b)
}

func (s *Server) listBroadcasts(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Broadcasts.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getBroadcast(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	b, err := s.Broadcasts.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, b)
}

func (s *Server) sendBroadcast(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var opts broadcast.SendOptions
	var raw map[string]any
	if err := decode(r, &raw); err == nil {
		if imm, ok := raw["immediate"].(bool); ok && imm {
			opts.Immediate = true
		}
		if sched, ok := raw["scheduled_at"].(string); ok && sched != "" {
			if ts, err := time.Parse(time.RFC3339, sched); err == nil {
				opts.ScheduledAt = &ts
			}
		}
	}
	b, err := s.Broadcasts.Send(r.Context(), team.ID, id, opts)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, b)
}

func (s *Server) cancelBroadcast(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	b, err := s.Broadcasts.Cancel(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, b)
}

func (s *Server) deleteBroadcast(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Broadcasts.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) listReceived(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Inbound.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getReceived(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := s.Inbound.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, e)
}

func (s *Server) listLogs(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `
		SELECT id, method, path, status_code, request_id, duration_ms, created_at
		FROM api_logs WHERE team_id = $1 ORDER BY created_at DESC LIMIT 100
	`, team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var method, path string
		var status int
		var reqID *string
		var dur *int
		var created time.Time
		if err := rows.Scan(&id, &method, &path, &status, &reqID, &dur, &created); err != nil {
			writeErr(w, err)
			return
		}
		list = append(list, map[string]any{
			"id": id, "method": method, "path": path, "status_code": status,
			"request_id": reqID, "duration_ms": dur, "created_at": created,
		})
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getLog(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var method, path string
	var status int
	var reqID *string
	var dur *int
	var created time.Time
	err := s.Pool.QueryRow(r.Context(), `
		SELECT method, path, status_code, request_id, duration_ms, created_at
		FROM api_logs WHERE id = $1 AND team_id = $2
	`, id, team.ID).Scan(&method, &path, &status, &reqID, &dur, &created)
	if err != nil {
		writeErr(w, apierr.NotFound)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"id": id, "method": method, "path": path, "status_code": status,
		"request_id": reqID, "duration_ms": dur, "created_at": created,
	})
}

func (s *Server) getUsage(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	u, err := s.Billing.CurrentUsage(r.Context(), team.ID, team.MonthlyQuota, team.BillingStatus)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, u)
}

func (s *Server) createCheckout(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	if body.SuccessURL == "" {
		body.SuccessURL = s.Cfg.ConsoleOrigin + "/settings?success=1"
	}
	if body.CancelURL == "" {
		body.CancelURL = s.Cfg.ConsoleOrigin + "/settings"
	}
	url, err := s.Billing.CreateCheckout(r.Context(), team.ID, team.Name, body.SuccessURL, body.CancelURL)
	if err != nil {
		writeErr(w, apierr.Validation(err.Error()))
		return
	}
	apierr.WriteJSON(w, 200, map[string]string{"url": url})
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return ""
	}
}
