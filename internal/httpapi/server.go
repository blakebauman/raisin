package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/audience"
	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/automation"
	"github.com/blakebauman/raisin/internal/billing"
	"github.com/blakebauman/raisin/internal/broadcast"
	"github.com/blakebauman/raisin/internal/config"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/domain"
	"github.com/blakebauman/raisin/internal/email"
	"github.com/blakebauman/raisin/internal/events"
	"github.com/blakebauman/raisin/internal/inbound"
	"github.com/blakebauman/raisin/internal/ippool"
	"github.com/blakebauman/raisin/internal/metrics"
	"github.com/blakebauman/raisin/internal/oauth"
	"github.com/blakebauman/raisin/internal/suppression"
	"github.com/blakebauman/raisin/internal/template"
	"github.com/blakebauman/raisin/internal/webhook"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	Cfg          config.Config
	Pool         *db.Pool
	Asynq        *asynq.Client
	Redis        *redis.Client
	Emails       *email.Service
	Domains      *domain.Service
	Webhooks     *webhook.Service
	Events       *events.Processor
	Suppressions *suppression.Service
	Audience     *audience.Service
	Templates    *template.Service
	Broadcasts   *broadcast.Service
	Billing      *billing.Service
	Metrics      *metrics.Service
	Inbound      *inbound.Service
	Automations  *automation.Service
	IPPools      *ippool.Service
	OAuth        *oauth.Service
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.Cfg.ConsoleOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "User-Agent", "X-Team-Token"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Request-Id"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierr.WriteJSON(w, 200, map[string]string{"status": "ok"})
	})
	r.Get("/openapi.yaml", s.serveOpenAPIYAML)
	r.Get("/docs", s.serveAPIDocs)
	r.Get("/docs/", s.serveAPIDocs)

	r.Post("/console/token", s.consoleToken)
	r.Post("/console/provision", s.consoleProvision)
	r.Post("/inbound/ses", s.inboundSES) // SNS → SES receipt (no API key)
	r.Get("/team/invites/preview", s.previewTeamInvite)

	r.Group(func(r chi.Router) {
		r.Use(s.authenticate)
		r.Use(s.rateLimit)
		r.Use(s.logAPI)

		r.Route("/emails", func(r chi.Router) {
			r.Post("/", s.sendEmail)
			r.Post("/batch", s.sendBatch)
			r.Get("/", s.listEmails)
			r.Get("/metrics", s.emailMetrics)
			r.Get("/received", s.listReceived)
			r.Get("/received/{id}", s.getReceived)
			r.Get("/received/{id}/attachments", s.listReceivedAttachments)
			r.Get("/received/{id}/attachments/{attachmentId}", s.getReceivedAttachment)
			r.Get("/{id}", s.getEmail)
			r.Patch("/{id}", s.updateEmail)
			r.Post("/{id}/cancel", s.cancelEmail)
			r.Get("/{id}/events", s.listEmailEvents)
			r.Get("/{id}/attachments", s.listAttachments)
			r.Get("/{id}/attachments/{attachmentId}", s.getAttachment)
		})

		r.Route("/domains", func(r chi.Router) {
			r.Post("/", s.createDomain)
			r.Get("/", s.listDomains)
			r.Get("/regions", s.listRegions)
			r.Get("/{id}", s.getDomain)
			r.Patch("/{id}", s.updateDomain)
			r.Post("/{id}/verify", s.verifyDomain)
			r.Post("/{id}/claim", s.claimDomain)
			r.Post("/{id}/claim/confirm", s.confirmClaimDomain)
			r.Post("/{id}/claim/release", s.releaseClaimDomain)
			r.Post("/{id}/bimi", s.setDomainBIMI)
			r.Post("/{id}/receiving", s.setDomainReceiving)
			r.Delete("/{id}", s.deleteDomain)
		})

		r.Route("/api-keys", func(r chi.Router) {
			r.Post("/", s.createAPIKey)
			r.Get("/", s.listAPIKeys)
			r.Delete("/{id}", s.deleteAPIKey)
		})

		r.Route("/webhooks", func(r chi.Router) {
			r.Post("/", s.createWebhook)
			r.Get("/", s.listWebhooks)
			r.Get("/{id}", s.getWebhook)
			r.Patch("/{id}", s.updateWebhook)
			r.Delete("/{id}", s.deleteWebhook)
			r.Get("/{id}/events", s.listWebhookEvents)
			r.Get("/{id}/events/{eventId}/attempts", s.listWebhookAttempts)
		})

		r.Get("/events/stream", s.streamEmailEvents)

		r.Get("/team", s.getTeam)
		r.Patch("/team", s.updateTeam)
		r.Get("/team/members", s.listTeamMembers)
		r.Patch("/team/members/{id}", s.updateTeamMember)
		r.Delete("/team/members/{id}", s.removeTeamMember)
		r.Get("/team/invites", s.listTeamInvites)
		r.Post("/team/invites", s.createTeamInvite)
		r.Delete("/team/invites/{id}", s.revokeTeamInvite)
		r.Post("/team/invites/accept", s.acceptTeamInvite)

		r.Route("/suppressions", func(r chi.Router) {
			r.Post("/", s.addSuppression)
			r.Post("/batch", s.addSuppressionsBatch)
			r.Get("/", s.listSuppressions)
			r.Delete("/{id}", s.removeSuppression)
		})

		r.Route("/contacts", func(r chi.Router) {
			r.Post("/", s.createContact)
			r.Get("/", s.listContacts)
			r.Get("/{id}", s.getContact)
			r.Patch("/{id}", s.updateContact)
			r.Delete("/{id}", s.deleteContact)
			r.Post("/{id}/segments/{segmentId}", s.addContactSegment)
			r.Delete("/{id}/segments/{segmentId}", s.removeContactSegment)
			r.Get("/{id}/segments", s.listContactSegments)
			r.Get("/{id}/topics", s.listContactTopics)
			r.Put("/{id}/topics/{topicId}", s.setContactTopic)
		})

		r.Route("/contact-properties", func(r chi.Router) {
			r.Post("/", s.createProperty)
			r.Get("/", s.listProperties)
			r.Delete("/{id}", s.deleteProperty)
		})

		r.Route("/segments", func(r chi.Router) {
			r.Post("/", s.createSegment)
			r.Get("/", s.listSegments)
			r.Delete("/{id}", s.deleteSegment)
		})

		r.Route("/topics", func(r chi.Router) {
			r.Post("/", s.createTopic)
			r.Get("/", s.listTopics)
			r.Get("/{id}", s.getTopic)
			r.Delete("/{id}", s.deleteTopic)
		})

		r.Route("/templates", func(r chi.Router) {
			r.Post("/", s.createTemplate)
			r.Get("/", s.listTemplates)
			r.Get("/{id}", s.getTemplate)
			r.Patch("/{id}", s.updateTemplate)
			r.Post("/{id}/publish", s.publishTemplate)
			r.Post("/{id}/duplicate", s.duplicateTemplate)
			r.Get("/{id}/preview", s.renderTemplatePreview)
			r.Delete("/{id}", s.deleteTemplate)
		})

		r.Route("/automations", func(r chi.Router) {
			r.Post("/", s.createAutomation)
			r.Get("/", s.listAutomations)
			r.Get("/{id}", s.getAutomation)
			r.Patch("/{id}", s.enableAutomation)
			r.Delete("/{id}", s.deleteAutomation)
			r.Get("/{id}/runs", s.listAutomationRuns)
		})

		r.Route("/ip-pools", func(r chi.Router) {
			r.Post("/", s.createIPPool)
			r.Get("/", s.listIPPools)
			r.Get("/{id}", s.getIPPool)
			r.Post("/{id}/pause", s.pauseIPPool)
			r.Post("/{id}/resume", s.resumeIPPool)
			r.Post("/{id}/warmup/tick", s.tickIPPoolWarmup)
			r.Post("/{id}/assign-domain", s.assignIPPoolDomain)
			r.Delete("/{id}", s.deleteIPPool)
		})

		r.Route("/oauth/apps", func(r chi.Router) {
			r.Post("/", s.createOAuthApp)
			r.Get("/", s.listOAuthApps)
			r.Delete("/{id}", s.deleteOAuthApp)
		})
		r.Post("/oauth/authorize", s.oauthAuthorize)

		r.Route("/broadcasts", func(r chi.Router) {
			r.Post("/", s.createBroadcast)
			r.Get("/", s.listBroadcasts)
			r.Get("/{id}", s.getBroadcast)
			r.Post("/{id}/send", s.sendBroadcast)
			r.Post("/{id}/cancel", s.cancelBroadcast)
			r.Delete("/{id}", s.deleteBroadcast)
		})

		r.Get("/logs", s.listLogs)
		r.Get("/logs/{id}", s.getLog)
		r.Get("/usage", s.getUsage)
		r.Get("/billing/plans", s.getBillingPlans)
		r.Post("/billing/checkout", s.createCheckout)
		r.Post("/billing/portal", s.createBillingPortal)
	})

	r.Post("/billing/webhook", s.stripeWebhook)
	r.Post("/oauth/token", s.oauthToken)
	r.Get("/oauth/apps/public", s.oauthPublicApp)

	return r
}

func oauthScopeAllowed(method, path string, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, s := range scopes {
		if s == "*" || s == "full_access" {
			return true
		}
	}
	need := oauthRequiredScope(method, path)
	if need == "" {
		// Unscoped surface (webhooks, broadcasts, api-keys, billing, …) requires full_access
		return false
	}
	for _, s := range scopes {
		if s == need {
			return true
		}
		if writeImpliesRead(need, s) {
			return true
		}
	}
	return false
}

func oauthRequiredScope(method, path string) string {
	mutating := method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut || method == http.MethodDelete
	switch {
	case strings.HasPrefix(path, "/emails") && method == http.MethodPost && !strings.Contains(path, "/received"):
		return "emails:send"
	case strings.HasPrefix(path, "/emails"), strings.HasPrefix(path, "/events"):
		return "emails:read"
	case strings.HasPrefix(path, "/domains") && mutating:
		return "domains:write"
	case strings.HasPrefix(path, "/domains"):
		return "domains:read"
	case strings.HasPrefix(path, "/contacts") && mutating:
		return "contacts:write"
	case strings.HasPrefix(path, "/contacts"), strings.HasPrefix(path, "/segments"), strings.HasPrefix(path, "/topics"), strings.HasPrefix(path, "/contact-properties"):
		return "contacts:read"
	case strings.HasPrefix(path, "/templates") && mutating:
		return "templates:write"
	case strings.HasPrefix(path, "/templates"):
		return "templates:read"
	case strings.HasPrefix(path, "/automations") && mutating:
		return "automations:write"
	case strings.HasPrefix(path, "/automations"):
		return "automations:read"
	case strings.HasPrefix(path, "/ip-pools") && mutating:
		return "ips:write"
	case strings.HasPrefix(path, "/ip-pools"):
		return "ips:read"
	default:
		return ""
	}
}

func writeImpliesRead(need, have string) bool {
	pairs := map[string]string{
		"emails:read":      "emails:send",
		"domains:read":     "domains:write",
		"contacts:read":    "contacts:write",
		"templates:read":   "templates:write",
		"automations:read": "automations:write",
		"ips:read":         "ips:write",
	}
	return pairs[need] == have
}


func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua == "" {
			apierr.Write(w, apierr.New(403, "missing_user_agent", "User-Agent header is required"))
			return
		}
		// Console JWT
		if tok := r.Header.Get("X-Team-Token"); tok != "" {
			claims, err := auth.ParseConsoleJWT(s.Cfg.JWTSecret, tok)
			if err != nil {
				apierr.Write(w, apierr.Forbidden)
				return
			}
			team, err := auth.LoadTeam(r.Context(), s.Pool, claims.TeamID)
			if err != nil {
				apierr.Write(w, apierr.Forbidden)
				return
			}
			ctx := context.WithValue(r.Context(), auth.TeamKey, team)
			ctx = context.WithValue(ctx, auth.ConsoleClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		authz := r.Header.Get("Authorization")
		if authz == "" {
			apierr.Write(w, apierr.Unauthorized)
			return
		}
		if s.OAuth != nil && strings.Contains(authz, "ra_atk_") {
			team, scopes, err := s.OAuth.LookupAccessToken(r.Context(), authz)
			if err == nil {
				ctx := context.WithValue(r.Context(), auth.TeamKey, team)
				ctx = context.WithValue(ctx, auth.OAuthScopesKey, scopes)
				if !oauthScopeAllowed(r.Method, r.URL.Path, scopes) {
					apierr.Write(w, apierr.New(403, "insufficient_scope", "token lacks required scope"))
					return
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		team, key, err := auth.LookupAPIKey(r.Context(), s.Pool, authz)
		if err != nil {
			apierr.Write(w, apierr.Forbidden)
			return
		}
		ctx := context.WithValue(r.Context(), auth.TeamKey, team)
		ctx = context.WithValue(ctx, auth.APIKeyCtxKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	const limit = 60
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		team := auth.TeamFromContext(r.Context())
		if team == nil || s.Redis == nil {
			next.ServeHTTP(w, r)
			return
		}
		key := "rl:" + team.ID.String() + ":" + time.Now().UTC().Format("20060102150405")
		n, err := s.Redis.Incr(r.Context(), key).Result()
		remaining := limit - int(n)
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().UTC().Add(time.Second).Unix(), 10))
		if err == nil {
			if n == 1 {
				s.Redis.Expire(r.Context(), key, 2*time.Second)
			}
			if n > int64(limit) {
				apierr.Write(w, apierr.RateLimited)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Long-lived SSE must not block api_logs until disconnect.
		if r.URL.Path == "/events/stream" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		dur := time.Since(start).Milliseconds()
		reqID := middleware.GetReqID(r.Context())
		team := auth.TeamFromContext(r.Context())
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", dur,
			"request_id", reqID,
		}
		if team != nil {
			attrs = append(attrs, "team_id", team.ID.String())
			_, _ = s.Pool.Exec(r.Context(), `
				INSERT INTO api_logs (team_id, method, path, status_code, request_id, duration_ms)
				VALUES ($1,$2,$3,$4,$5,$6)
			`, team.ID, r.Method, r.URL.Path, ww.Status(), reqID, dur)
		}
		if ww.Status() >= 500 {
			slog.Error("request", attrs...)
		} else if ww.Status() >= 400 {
			slog.Warn("request", attrs...)
		} else {
			slog.Info("request", attrs...)
		}
	})
}

func (s *Server) consoleToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TeamID string `json:"team_id"`
		UserID string `json:"user_id"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	if body.Secret != s.Cfg.JWTSecret {
		apierr.Write(w, apierr.Forbidden)
		return
	}
	tid, err := uuid.Parse(body.TeamID)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid team_id"))
		return
	}
	uid := uuid.Nil
	if body.UserID != "" {
		uid, _ = uuid.Parse(body.UserID)
	}
	tok, err := auth.IssueConsoleJWT(s.Cfg.JWTSecret, tid, uid, "owner", time.Hour)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]string{"token": tok})
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func teamOrWrite(w http.ResponseWriter, r *http.Request) *auth.Team {
	t := auth.TeamFromContext(r.Context())
	if t == nil {
		apierr.Write(w, apierr.Unauthorized)
	}
	return t
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func writeErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	apierr.Write(w, err)
}

func asStrings(v any) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func normalizeRecipients(to any) []string {
	return asStrings(to)
}

func strHeader(r *http.Request, k string) string {
	return strings.TrimSpace(r.Header.Get(k))
}
