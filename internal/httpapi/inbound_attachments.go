package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/snsverify"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) listAttachments(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := s.Emails.ListAttachments(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	emailID, ok := parseID(w, r)
	if !ok {
		return
	}
	aid, err := uuid.Parse(chi.URLParam(r, "attachmentId"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid attachment id"))
		return
	}
	meta, body, err := s.Emails.GetAttachment(r.Context(), team.ID, emailID, aid)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"id":           meta.ID,
		"filename":     meta.Filename,
		"content_type": meta.ContentType,
		"size_bytes":   meta.SizeBytes,
		"content_id":   meta.ContentID,
		"content":      base64.StdEncoding.EncodeToString(body),
	})
}

// inboundSES accepts SNS notifications for SES inbound (receipt → S3).
// Also supports a direct JSON body for local testing:
// { "team_id": "...", "from": "...", "to": ["..."], "subject": "...", "html": "...", "text": "..." }
func (s *Server) inboundSES(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid body"))
		return
	}
	var envelope snsverify.Envelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Type != "" {
		if err := snsverify.Verify(envelope); err != nil {
			apierr.Write(w, apierr.New(403, "invalid_sns_signature", err.Error()))
			return
		}
		if envelope.Type == "SubscriptionConfirmation" && envelope.SubscribeURL != "" {
			resp, err := http.Get(envelope.SubscribeURL)
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			apierr.WriteJSON(w, 200, map[string]string{"status": "subscribed"})
			return
		}
		if envelope.Type == "Notification" && envelope.Message != "" {
			if err := s.Inbound.HandleSESNNotification(r.Context(), []byte(envelope.Message)); err != nil {
				writeErr(w, err)
				return
			}
			apierr.WriteJSON(w, 200, map[string]string{"status": "ok"})
			return
		}
	}

	// Direct ingest for local/dev
	var direct struct {
		TeamID  string   `json:"team_id"`
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		Text    string   `json:"text"`
		S3Key   string   `json:"s3_key"`
	}
	if err := json.Unmarshal(body, &direct); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	teamID, err := uuid.Parse(direct.TeamID)
	if err != nil {
		apierr.Write(w, apierr.Validation("team_id required"))
		return
	}
	rec, err := s.Inbound.Store(r.Context(), teamID, direct.From, direct.To, direct.Subject, direct.HTML, direct.Text, direct.S3Key, "", nil)
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.Automations != nil {
		_ = s.Automations.Trigger(r.Context(), teamID, "email.received", nil, nil, &rec.ID, map[string]any{
			"received_email_id": rec.ID.String(),
			"from":              rec.From,
			"subject":           rec.Subject,
		})
	}
	if s.Webhooks != nil {
		_ = s.Webhooks.Fanout(r.Context(), teamID, "email.received", map[string]any{
			"received_email_id": rec.ID.String(),
			"from":              rec.From,
			"to":                rec.To,
		})
	}
	apierr.WriteJSON(w, 200, rec)
}
