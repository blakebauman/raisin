package httpapi

import (
	"net/http"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/automation"
	"github.com/blakebauman/raisin/internal/domain"
	"github.com/blakebauman/raisin/internal/ippool"
	"github.com/blakebauman/raisin/internal/oauth"
	"github.com/google/uuid"
)

func (s *Server) createAutomation(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req automation.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	a, err := s.Automations.Create(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, a)
}

func (s *Server) listAutomations(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.Automations.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getAutomation(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	a, err := s.Automations.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, a)
}

func (s *Server) enableAutomation(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	a, err := s.Automations.SetEnabled(r.Context(), team.ID, id, body.Enabled)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, a)
}

func (s *Server) deleteAutomation(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.Automations.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) listAutomationRuns(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := s.Automations.ListRuns(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) createIPPool(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req ippool.CreateRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	p, err := s.IPPools.Create(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) listIPPools(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.IPPools.List(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) getIPPool(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := s.IPPools.Get(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) pauseIPPool(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := s.IPPools.Pause(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) tickIPPoolWarmup(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := s.IPPools.TickWarmup(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) resumeIPPool(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := s.IPPools.Resume(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, p)
}

func (s *Server) deleteIPPool(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.IPPools.Delete(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) assignIPPoolDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	poolID, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		DomainID string `json:"domain_id"`
	}
	if err := decode(r, &body); err != nil || body.DomainID == "" {
		apierr.Write(w, apierr.Validation("domain_id required"))
		return
	}
	domainID, err := uuid.Parse(body.DomainID)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid domain_id"))
		return
	}
	if err := s.IPPools.AssignDomain(r.Context(), team.ID, poolID, domainID); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) claimDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	d, err := s.Domains.Claim(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) confirmClaimDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	d, err := s.Domains.ConfirmClaim(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) releaseClaimDomain(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	d, err := s.Domains.ReleaseClaim(r.Context(), team.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) setDomainBIMI(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req domain.BIMIRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	d, err := s.Domains.SetBIMI(r.Context(), team.ID, id, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) setDomainReceiving(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	d, err := s.Domains.EnableReceiving(r.Context(), team.ID, id, body.Enabled)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, d)
}

func (s *Server) listRegions(w http.ResponseWriter, r *http.Request) {
	apierr.WriteJSON(w, 200, map[string]any{"data": domain.AllowedRegions()})
}

func (s *Server) createOAuthApp(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var req oauth.CreateAppRequest
	if err := decode(r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	app, err := s.OAuth.CreateApp(r.Context(), team.ID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, app)
}

func (s *Server) listOAuthApps(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	list, err := s.OAuth.ListApps(r.Context(), team.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{"data": list})
}

func (s *Server) deleteOAuthApp(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.OAuth.DeleteApp(r.Context(), team.ID, id); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]bool{"deleted": true})
}

func (s *Server) oauthPublicApp(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		apierr.Write(w, apierr.Validation("client_id required"))
		return
	}
	app, err := s.OAuth.PublicApp(r.Context(), clientID)
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"name":          app.Name,
		"client_id":     app.ClientID,
		"redirect_uris": app.RedirectURIs,
		"scopes":        app.Scopes,
	})
}

func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	var body struct {
		ClientID    string   `json:"client_id"`
		RedirectURI string   `json:"redirect_uri"`
		Scopes      []string `json:"scopes"`
	}
	if err := decode(r, &body); err != nil {
		apierr.Write(w, apierr.Validation("invalid json"))
		return
	}
	code, err := s.OAuth.CreateAuthorizationCode(r.Context(), oauth.AuthorizeRequest{
		ClientID: body.ClientID, RedirectURI: body.RedirectURI, Scopes: body.Scopes, TeamID: team.ID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]string{"code": code})
}

func (s *Server) oauthToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	grant := r.FormValue("grant_type")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	refreshToken := r.FormValue("refresh_token")
	if grant == "" {
		var body struct {
			GrantType    string `json:"grant_type"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			Code         string `json:"code"`
			RedirectURI  string `json:"redirect_uri"`
			RefreshToken string `json:"refresh_token"`
		}
		_ = decode(r, &body)
		grant = body.GrantType
		clientID = body.ClientID
		clientSecret = body.ClientSecret
		code = body.Code
		redirectURI = body.RedirectURI
		refreshToken = body.RefreshToken
	}
	switch grant {
	case "authorization_code":
		tok, err := s.OAuth.ExchangeCode(r.Context(), clientID, clientSecret, code, redirectURI)
		if err != nil {
			writeErr(w, err)
			return
		}
		apierr.WriteJSON(w, 200, tok)
	case "refresh_token":
		tok, err := s.OAuth.RefreshExchange(r.Context(), clientID, clientSecret, refreshToken)
		if err != nil {
			writeErr(w, err)
			return
		}
		apierr.WriteJSON(w, 200, tok)
	default:
		apierr.Write(w, apierr.Validation("unsupported grant_type"))
	}
}

func (s *Server) renderTemplatePreview(w http.ResponseWriter, r *http.Request) {
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
	html := ""
	if t.HTML != nil {
		html = *t.HTML
	}
	apierr.WriteJSON(w, 200, map[string]any{
		"id":           t.ID,
		"html":         html,
		"react_source": t.ReactSource,
		"editor_json":  t.EditorJSON,
	})
}