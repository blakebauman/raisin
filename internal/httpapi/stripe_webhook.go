package httpapi

import (
	"io"
	"net/http"

	"github.com/raisin-run/raisin/internal/apierr"
	stripewebhook "github.com/stripe/stripe-go/v81/webhook"
)

func (s *Server) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBody = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid body"))
		return
	}
	secret := s.Cfg.StripeWebhookSecret
	if secret == "" {
		apierr.Write(w, apierr.Validation("stripe webhook not configured"))
		return
	}
	event, err := stripewebhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), secret)
	if err != nil {
		apierr.Write(w, apierr.New(400, "invalid_signature", "invalid stripe signature"))
		return
	}
	if err := s.Billing.HandleStripeEvent(r.Context(), string(event.Type), event.Data.Raw); err != nil {
		writeErr(w, err)
		return
	}
	apierr.WriteJSON(w, 200, map[string]string{"received": "true"})
}
