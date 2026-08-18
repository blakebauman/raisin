package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/billing"
	"github.com/blakebauman/raisin/internal/config"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/domain"
	"github.com/blakebauman/raisin/internal/email"
	"github.com/blakebauman/raisin/internal/events"
	"github.com/blakebauman/raisin/internal/jobs"
	"github.com/blakebauman/raisin/internal/sender"
	"github.com/blakebauman/raisin/internal/storage"
	"github.com/blakebauman/raisin/internal/tracking"
	tmpl "github.com/blakebauman/raisin/internal/template"
	"github.com/blakebauman/raisin/internal/webhook"
)

type Handlers struct {
	Cfg      config.Config
	Pool     *db.Pool
	Sender   sender.Sender
	Webhooks *webhook.Service
	Billing  *billing.Service
	Events   *events.Processor
	Asynq    *asynq.Client
	Storage  storage.Store
	Domains  *domain.Service
	Emails   *email.Service
	IPPools  interface {
		ReserveSend(ctx context.Context, poolID uuid.UUID) (string, error)
		TickAllWarmups(ctx context.Context) (int, error)
	}
}

func (h *Handlers) Mux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TypeEmailSend, h.HandleEmailSend)
	mux.HandleFunc(jobs.TypeWebhookDeliver, h.HandleWebhookDeliver)
	mux.HandleFunc(jobs.TypeBroadcastSend, h.HandleBroadcastSend)
	mux.HandleFunc(jobs.TypeDomainVerify, h.HandleDomainVerify)
	mux.HandleFunc(jobs.TypeAutomationStep, h.HandleAutomationStep)
	mux.HandleFunc(jobs.TypeIPWarmupTick, h.HandleIPWarmupTick)
	return mux
}

func (h *Handlers) HandleEmailSend(ctx context.Context, t *asynq.Task) error {
	var p jobs.EmailSendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	emailID, err := uuid.Parse(p.EmailID)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(p.TeamID)
	if err != nil {
		return err
	}

	var status, from, subject string
	var to, cc, bcc, replyTo []string
	var html, text *string
	var domainID *uuid.UUID
	var openTrack, clickTrack bool
	var templateID *uuid.UUID
	var tagsJSON []byte
	err = h.Pool.QueryRow(ctx, `
		SELECT e.status, e.from_addr, e.to_addrs, e.cc_addrs, e.bcc_addrs, e.reply_to,
		       e.subject, e.html, e.text, e.domain_id, e.template_id, e.tags,
		       COALESCE(d.open_tracking, true), COALESCE(d.click_tracking, true)
		FROM emails e
		LEFT JOIN domains d ON d.id = e.domain_id
		WHERE e.id = $1 AND e.team_id = $2
	`, emailID, teamID).Scan(
		&status, &from, &to, &cc, &bcc, &replyTo, &subject, &html, &text, &domainID, &templateID, &tagsJSON, &openTrack, &clickTrack,
	)
	if err != nil {
		return err
	}
	if status == "canceled" || status == "sent" || status == "delivered" {
		return nil
	}

	htmlBody := ""
	textBody := ""
	if html != nil {
		htmlBody = *html
	}
	if text != nil {
		textBody = *text
	}

	if templateID != nil && htmlBody == "" {
		var thtml, tsubj, ttext *string
		_ = h.Pool.QueryRow(ctx, `SELECT html, subject, text FROM templates WHERE id = $1`, *templateID).Scan(&thtml, &tsubj, &ttext)
		if thtml != nil {
			htmlBody = *thtml
		}
		if tsubj != nil && subject == "" {
			subject = *tsubj
		}
		if ttext != nil && textBody == "" {
			textBody = *ttext
		}
	}

	unsubURL := fmt.Sprintf("%s/unsubscribe/%s", h.Cfg.TrackingBaseURL, emailID.String())
	var tagVars map[string]string
	if len(tagsJSON) > 0 {
		var tags map[string]any
		if json.Unmarshal(tagsJSON, &tags) == nil {
			tagVars = map[string]string{}
			for k, v := range tags {
				tagVars[k] = fmt.Sprint(v)
				if k == "topic_id" {
					if tid, ok := v.(string); ok && tid != "" {
						unsubURL = fmt.Sprintf("%s?topic=%s", unsubURL, url.QueryEscape(tid))
					}
				}
			}
		}
	}
	if len(tagVars) > 0 {
		subject = tmpl.Render(subject, tagVars)
		htmlBody = tmpl.Render(htmlBody, tagVars)
		textBody = tmpl.Render(textBody, tagVars)
	}
	htmlBody = tracking.Inject(htmlBody, emailID, h.Cfg.TrackingBaseURL, openTrack, clickTrack)
	headers := map[string]string{
		"List-Unsubscribe":      fmt.Sprintf("<%s>", unsubURL),
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
	}

	var atts []sender.Attachment
	rows, err := h.Pool.Query(ctx, `
		SELECT filename, content_type, s3_key, content_id FROM attachments WHERE email_id = $1
	`, emailID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var filename, ctype, key string
		var contentID *string
		if err := rows.Scan(&filename, &ctype, &key, &contentID); err != nil {
			return err
		}
		var body []byte
		if h.Storage != nil {
			body, _, err = h.Storage.Get(ctx, key)
			if err != nil {
				return fmt.Errorf("load attachment %s: %w", filename, err)
			}
		}
		cid := ""
		if contentID != nil {
			cid = *contentID
		}
		atts = append(atts, sender.Attachment{
			Filename: filename, ContentType: ctype, Content: body, ContentID: cid,
		})
	}

	configSet := h.Cfg.SESConfigurationSet
	if domainID != nil {
		var poolID *uuid.UUID
		_ = h.Pool.QueryRow(ctx, `SELECT ip_pool_id FROM domains WHERE id = $1`, *domainID).Scan(&poolID)
		if poolID != nil && h.IPPools != nil {
			cs, err := h.IPPools.ReserveSend(ctx, *poolID)
			if err != nil {
				_, _ = h.Pool.Exec(ctx, `UPDATE emails SET status = 'failed', updated_at = now() WHERE id = $1`, emailID)
				_ = h.Events.RecordLocalEvent(ctx, teamID, emailID, "email.failed", map[string]any{"error": err.Error()})
				return err
			}
			if cs != "" {
				configSet = cs
			}
		}
	}

	res, err := h.Sender.Send(ctx, sender.Message{
		From: from, To: to, Cc: cc, Bcc: bcc, ReplyTo: replyTo,
		Subject: subject, HTML: htmlBody, Text: textBody,
		Headers: headers, ConfigSet: configSet,
		Tags:        map[string]string{"email_id": emailID.String(), "team_id": teamID.String()},
		Attachments: atts,
	})
	if err != nil {
		_, _ = h.Pool.Exec(ctx, `UPDATE emails SET status = 'failed', updated_at = now() WHERE id = $1`, emailID)
		_ = h.Events.RecordLocalEvent(ctx, teamID, emailID, "email.failed", map[string]any{"error": err.Error()})
		return err
	}

	_, _ = h.Pool.Exec(ctx, `
		UPDATE emails SET status = 'sent', provider_message_id = $2, sent_at = now(), updated_at = now()
		WHERE id = $1
	`, emailID, res.MessageID)
	_ = h.Billing.IncrementUsage(ctx, teamID, 1)
	_ = h.Events.RecordLocalEvent(ctx, teamID, emailID, "email.sent", map[string]any{
		"email_id": emailID.String(), "message_id": res.MessageID, "from": from, "to": to, "subject": subject,
	})

	// In test/mailpit mode, also emit delivered
	if h.Cfg.SenderDriver == "mailpit" {
		_ = h.Events.RecordLocalEvent(ctx, teamID, emailID, "email.delivered", map[string]any{
			"email_id": emailID.String(), "message_id": res.MessageID,
		})
	}
	return nil
}

func (h *Handlers) HandleWebhookDeliver(ctx context.Context, t *asynq.Task) error {
	var p jobs.WebhookDeliverPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	id, err := uuid.Parse(p.WebhookEventID)
	if err != nil {
		return err
	}
	return h.Webhooks.Deliver(ctx, id)
}

func (h *Handlers) HandleBroadcastSend(ctx context.Context, t *asynq.Task) error {
	var p jobs.BroadcastSendPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	bid, _ := uuid.Parse(p.BroadcastID)
	tid, _ := uuid.Parse(p.TeamID)

	var segmentID, topicID *uuid.UUID
	var from, subject string
	var html, text *string
	err := h.Pool.QueryRow(ctx, `
		SELECT segment_id, topic_id, from_addr, subject, html, text FROM broadcasts WHERE id = $1 AND team_id = $2
	`, bid, tid).Scan(&segmentID, &topicID, &from, &subject, &html, &text)
	if err != nil {
		return err
	}
	_, _ = h.Pool.Exec(ctx, `UPDATE broadcasts SET status = 'sending', updated_at = now() WHERE id = $1`, bid)

	htmlBody, textBody := "", ""
	if html != nil {
		htmlBody = *html
	}
	if text != nil {
		textBody = *text
	}

	var emails []string
	if topicID != nil {
		emails, err = h.topicRecipientEmails(ctx, tid, *topicID, segmentID)
		if err != nil {
			return err
		}
	} else if segmentID != nil {
		rows, err := h.Pool.Query(ctx, `
			SELECT c.email FROM contacts c
			JOIN segment_members sm ON sm.contact_id = c.id
			WHERE sm.segment_id = $1 AND c.unsubscribed = FALSE AND c.team_id = $2
			  AND NOT EXISTS (
			    SELECT 1 FROM suppressions s WHERE s.team_id = c.team_id AND s.email = c.email
			  )
		`, *segmentID, tid)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e string
			if err := rows.Scan(&e); err != nil {
				return err
			}
			emails = append(emails, e)
		}
	} else {
		rows, err := h.Pool.Query(ctx, `
			SELECT c.email FROM contacts c
			WHERE c.team_id = $1 AND c.unsubscribed = FALSE
			  AND NOT EXISTS (
			    SELECT 1 FROM suppressions s WHERE s.team_id = c.team_id AND s.email = c.email
			  )
		`, tid)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e string
			if err := rows.Scan(&e); err != nil {
				return err
			}
			emails = append(emails, e)
		}
	}

	emailSvc := &email.Service{Pool: h.Pool, Client: h.Asynq}
	team, err := auth.LoadTeam(ctx, h.Pool, tid)
	if err != nil {
		return err
	}
	baseTags := map[string]string{}
	if topicID != nil {
		baseTags["topic_id"] = topicID.String()
	}
	for _, addr := range emails {
		tags := h.mergeContactTags(ctx, tid, addr, baseTags)
		_, err := emailSvc.Send(ctx, team, email.SendRequest{
			From: from, To: []string{addr}, Subject: subject, HTML: htmlBody, Text: textBody, Tags: tags,
		}, "")
		if err != nil {
			log.Printf("broadcast send to %s: %v", addr, err)
		}
	}
	_, _ = h.Pool.Exec(ctx, `UPDATE broadcasts SET status = 'sent', sent_at = now(), updated_at = now() WHERE id = $1`, bid)
	return nil
}

func (h *Handlers) topicRecipientEmails(ctx context.Context, teamID, topicID uuid.UUID, segmentID *uuid.UUID) ([]string, error) {
	q := `
		SELECT c.email
		FROM contacts c
		CROSS JOIN topics t
		LEFT JOIN topic_subscriptions ts ON ts.topic_id = t.id AND ts.contact_id = c.id
		WHERE t.id = $1 AND c.team_id = $2 AND t.team_id = $2
		  AND c.unsubscribed = FALSE
		  AND COALESCE(ts.subscribed, t.default_subscription = 'opt_out') = TRUE
		  AND NOT EXISTS (
		    SELECT 1 FROM suppressions s WHERE s.team_id = c.team_id AND s.email = c.email
		  )
	`
	args := []any{topicID, teamID}
	if segmentID != nil {
		q += `
		  AND EXISTS (
		    SELECT 1 FROM segment_members sm
		    WHERE sm.contact_id = c.id AND sm.segment_id = $3
		  )`
		args = append(args, *segmentID)
	}
	rows, err := h.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// mergeContactTags copies base and adds contact first_name/last_name/email/properties for {{var}} merge.
func (h *Handlers) mergeContactTags(ctx context.Context, teamID uuid.UUID, addr string, base map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	emailAddr := strings.ToLower(strings.TrimSpace(addr))
	if i := strings.Index(emailAddr, "<"); i >= 0 {
		if j := strings.Index(emailAddr, ">"); j > i {
			emailAddr = strings.TrimSpace(emailAddr[i+1 : j])
		}
	}
	out["email"] = emailAddr
	var first, last *string
	var props []byte
	err := h.Pool.QueryRow(ctx, `
		SELECT first_name, last_name, properties FROM contacts
		WHERE team_id = $1 AND email = $2
	`, teamID, emailAddr).Scan(&first, &last, &props)
	if err != nil {
		return out
	}
	if first != nil {
		out["first_name"] = *first
	}
	if last != nil {
		out["last_name"] = *last
	}
	if len(props) > 0 {
		var m map[string]any
		if json.Unmarshal(props, &m) == nil {
			for k, v := range m {
				if _, exists := out[k]; !exists {
					out[k] = fmt.Sprint(v)
				}
			}
		}
	}
	return out
}

func (h *Handlers) HandleDomainVerify(ctx context.Context, t *asynq.Task) error {
	var p jobs.DomainVerifyPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	domainID, err := uuid.Parse(p.DomainID)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(p.TeamID)
	if err != nil {
		return err
	}
	if h.Domains == nil {
		return nil
	}
	_, err = h.Domains.Verify(ctx, teamID, domainID)
	return err
}

func (h *Handlers) HandleIPWarmupTick(ctx context.Context, t *asynq.Task) error {
	_ = t
	if h.IPPools == nil {
		return nil
	}
	n, err := h.IPPools.TickAllWarmups(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		log.Printf("ip warmup tick advanced %d pools", n)
	}
	return nil
}

// TrackingHTTP serves open/click + unsubscribe endpoints on the worker.
func (h *Handlers) TrackingHTTP() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/t/o/", h.handleOpen)
	mux.HandleFunc("/t/c/", h.handleClick)
	mux.HandleFunc("/unsubscribe/", h.handleUnsubscribe)
	mux.HandleFunc("/test/events/", h.handleTestEvent) // synthesize events in test mode
	return mux
}

func (h *Handlers) handleOpen(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/t/o/"):]
	emailID, err := uuid.Parse(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var teamID uuid.UUID
	_ = h.Pool.QueryRow(r.Context(), `SELECT team_id FROM emails WHERE id = $1`, emailID).Scan(&teamID)
	if teamID != uuid.Nil {
		_ = h.Events.RecordLocalEvent(r.Context(), teamID, emailID, "email.opened", map[string]any{
			"email_id": emailID.String(),
		})
	}
	w.Header().Set("Content-Type", "image/gif")
	// 1x1 transparent GIF
	_, _ = w.Write([]byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff,
		0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
	})
}

func (h *Handlers) handleClick(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/t/c/"):]
	emailID, err := uuid.Parse(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	dest := r.URL.Query().Get("u")
	if dest == "" {
		http.NotFound(w, r)
		return
	}
	decoded, err := url.QueryUnescape(dest)
	if err != nil {
		decoded = dest
	}
	var teamID uuid.UUID
	_ = h.Pool.QueryRow(r.Context(), `SELECT team_id FROM emails WHERE id = $1`, emailID).Scan(&teamID)
	if teamID != uuid.Nil {
		_ = h.Events.RecordLocalEvent(r.Context(), teamID, emailID, "email.clicked", map[string]any{
			"email_id": emailID.String(), "link": decoded,
		})
	}
	http.Redirect(w, r, decoded, http.StatusFound)
}

func (h *Handlers) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/unsubscribe/"):]
	if i := strings.IndexByte(idStr, '?'); i >= 0 {
		idStr = idStr[:i]
	}
	idStr = strings.Trim(idStr, "/")
	emailID, err := uuid.Parse(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	topicParam := r.URL.Query().Get("topic")
	if topicParam == "" {
		_ = r.ParseForm()
		topicParam = r.Form.Get("topic")
	}
	var topicID uuid.UUID
	if topicParam != "" {
		topicID, err = uuid.Parse(topicParam)
		if err != nil {
			http.Error(w, "invalid topic", http.StatusBadRequest)
			return
		}
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.writeUnsubscribeConfirm(w, emailID, topicID)
		return
	case http.MethodPost:
		// RFC 8058 one-click or confirm form
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var teamID uuid.UUID
	var toAddrs []string
	_ = h.Pool.QueryRow(r.Context(), `SELECT team_id, to_addrs FROM emails WHERE id = $1`, emailID).Scan(&teamID, &toAddrs)
	if teamID == uuid.Nil {
		http.NotFound(w, r)
		return
	}

	for _, a := range toAddrs {
		if topicID != uuid.Nil {
			_, _ = h.Pool.Exec(r.Context(), `
				INSERT INTO topic_subscriptions (topic_id, contact_id, subscribed, updated_at)
				SELECT $1, c.id, FALSE, now()
				FROM contacts c
				WHERE c.team_id = $2 AND c.email = lower($3)
				ON CONFLICT (topic_id, contact_id) DO UPDATE
				SET subscribed = FALSE, updated_at = now()
			`, topicID, teamID, a)
		} else {
			_, _ = h.Pool.Exec(r.Context(), `
				UPDATE contacts SET unsubscribed = TRUE, updated_at = now()
				WHERE team_id = $1 AND email = lower($2)
			`, teamID, a)
			_, _ = h.Pool.Exec(r.Context(), `
				INSERT INTO suppressions (team_id, email, reason)
				VALUES ($1, lower($2), 'unsubscribe')
				ON CONFLICT (team_id, email) DO UPDATE SET reason = EXCLUDED.reason
			`, teamID, a)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	msg := "You have been unsubscribed."
	if topicID != uuid.Nil {
		msg = "You have been unsubscribed from this topic."
	}
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><h1>Unsubscribed</h1><p>` + msg + `</p></body></html>`))
}

func (h *Handlers) writeUnsubscribeConfirm(w http.ResponseWriter, emailID, topicID uuid.UUID) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	action := "/unsubscribe/" + emailID.String()
	topicField := ""
	label := "Unsubscribe from all email"
	if topicID != uuid.Nil {
		topicField = `<input type="hidden" name="topic" value="` + topicID.String() + `">`
		label = "Unsubscribe from this topic"
		action += "?topic=" + url.QueryEscape(topicID.String())
	}
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Unsubscribe</title></head><body>
<h1>Confirm unsubscribe</h1>
<p>Click the button below to confirm. Link scanners will not unsubscribe you.</p>
<form method="POST" action="` + action + `">
` + topicField + `
<input type="hidden" name="List-Unsubscribe" value="One-Click">
<button type="submit">` + label + `</button>
</form>
</body></html>`))
}


func (h *Handlers) handleTestEvent(w http.ResponseWriter, r *http.Request) {
	// POST /test/events/{email_id}?type=email.bounced
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	idStr := r.URL.Path[len("/test/events/"):]
	emailID, err := uuid.Parse(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "email.bounced"
	}
	var teamID uuid.UUID
	var testMode bool
	err = h.Pool.QueryRow(r.Context(), `
		SELECT e.team_id, t.test_mode FROM emails e JOIN teams t ON t.id = e.team_id WHERE e.id = $1
	`, emailID).Scan(&teamID, &testMode)
	if err != nil || !testMode {
		http.Error(w, "not found or not test mode", 404)
		return
	}
	_ = h.Events.RecordLocalEvent(r.Context(), teamID, emailID, typ, map[string]any{"email_id": emailID.String(), "test": true})
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

var _ = time.Now
