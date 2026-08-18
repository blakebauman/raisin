package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/stripe/stripe-go/v81"
	portalsession "github.com/stripe/stripe-go/v81/billingportal/session"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
)

type Service struct {
	Pool            *db.Pool
	SecretKey       string
	PricePro        string // monthly Price ID (price_…)
	PriceProAnnual  string // optional annual Price ID
	ProMonthlyQuota int    // quota granted on active Pro
}

type Usage struct {
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	EmailsSent  int       `json:"emails_sent"`
	Quota       int       `json:"quota"`
	Remaining   int       `json:"remaining"`
	Status      string    `json:"billing_status"`
}

type PlanInfo struct {
	Configured     bool   `json:"configured"`
	Name           string `json:"name"`
	Interval       string `json:"interval"` // month | year
	HasAnnual      bool   `json:"has_annual"`
	MonthlyQuota   int    `json:"monthly_quota"`
	CheckoutReady  bool   `json:"checkout_ready"`
	PortalReady    bool   `json:"portal_ready"`
}

func (s *Service) Plans() PlanInfo {
	ready := s.SecretKey != "" && s.PricePro != ""
	quota := s.ProMonthlyQuota
	if quota <= 0 {
		quota = 100000
	}
	return PlanInfo{
		Configured:    ready,
		Name:          "Raisin Pro",
		Interval:      "month",
		HasAnnual:     s.PriceProAnnual != "",
		MonthlyQuota:  quota,
		CheckoutReady: ready,
		PortalReady:   s.SecretKey != "",
	}
}

func (s *Service) priceIDFor(interval string) (string, error) {
	interval = strings.ToLower(strings.TrimSpace(interval))
	if interval == "" || interval == "month" || interval == "monthly" {
		if s.PricePro == "" {
			return "", fmt.Errorf("STRIPE_PRICE_PRO is not configured")
		}
		return s.PricePro, nil
	}
	if interval == "year" || interval == "annual" || interval == "yearly" {
		if s.PriceProAnnual == "" {
			return "", fmt.Errorf("STRIPE_PRICE_PRO_ANNUAL is not configured")
		}
		return s.PriceProAnnual, nil
	}
	return "", fmt.Errorf("interval must be month or year")
}

func (s *Service) CurrentUsage(ctx context.Context, teamID uuid.UUID, quota int, status string) (*Usage, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	var sent int
	_ = s.Pool.QueryRow(ctx, `
		SELECT COALESCE(emails_sent, 0) FROM usage_periods WHERE team_id = $1 AND period_start = $2
	`, teamID, start).Scan(&sent)
	rem := quota - sent
	if rem < 0 {
		rem = 0
	}
	return &Usage{
		PeriodStart: start,
		PeriodEnd:   end,
		EmailsSent:  sent,
		Quota:       quota,
		Remaining:   rem,
		Status:      status,
	}, nil
}

func (s *Service) IncrementUsage(ctx context.Context, teamID uuid.UUID, n int) error {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO usage_periods (team_id, period_start, period_end, emails_sent)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (team_id, period_start) DO UPDATE SET emails_sent = usage_periods.emails_sent + $4
	`, teamID, start, end, n)
	return err
}

func (s *Service) PauseIfOverQuota(ctx context.Context, teamID uuid.UUID, quota int) error {
	u, err := s.CurrentUsage(ctx, teamID, quota, "")
	if err != nil {
		return err
	}
	if u.EmailsSent >= quota {
		_, err = s.Pool.Exec(ctx, `UPDATE teams SET billing_status = 'paused', updated_at = now() WHERE id = $1`, teamID)
		return err
	}
	return nil
}

func (s *Service) ensureCustomer(ctx context.Context, teamID uuid.UUID, teamName string) (string, error) {
	var customerID *string
	_ = s.Pool.QueryRow(ctx, `SELECT stripe_customer_id FROM teams WHERE id = $1`, teamID).Scan(&customerID)
	if customerID != nil && *customerID != "" {
		return *customerID, nil
	}
	c, err := customer.New(&stripe.CustomerParams{
		Name: stripe.String(teamName),
		Metadata: map[string]string{"team_id": teamID.String()},
	})
	if err != nil {
		return "", err
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE teams SET stripe_customer_id = $2 WHERE id = $1`, teamID, c.ID)
	return c.ID, nil
}

func (s *Service) CreateCheckout(ctx context.Context, teamID uuid.UUID, teamName, successURL, cancelURL, interval string) (string, error) {
	if s.SecretKey == "" {
		return "", fmt.Errorf("stripe not configured")
	}
	priceID, err := s.priceIDFor(interval)
	if err != nil {
		return "", err
	}
	stripe.Key = s.SecretKey

	customerID, err := s.ensureCustomer(ctx, teamID, teamName)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Mode:               stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:           stripe.String(customerID),
		ClientReferenceID:  stripe.String(teamID.String()),
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		AllowPromotionCodes: stripe.Bool(true),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{"team_id": teamID.String()},
		},
		Metadata: map[string]string{"team_id": teamID.String()},
	}
	sess, err := session.New(params)
	if err != nil {
		return "", err
	}
	return sess.URL, nil
}

func (s *Service) CreatePortal(ctx context.Context, teamID uuid.UUID, returnURL string) (string, error) {
	if s.SecretKey == "" {
		return "", fmt.Errorf("stripe not configured")
	}
	stripe.Key = s.SecretKey

	var customerID *string
	_ = s.Pool.QueryRow(ctx, `SELECT stripe_customer_id FROM teams WHERE id = $1`, teamID).Scan(&customerID)
	if customerID == nil || *customerID == "" {
		return "", fmt.Errorf("no Stripe customer for this team; complete checkout first")
	}
	sess, err := portalsession.New(&stripe.BillingPortalSessionParams{
		Customer:  customerID,
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return "", err
	}
	return sess.URL, nil
}

func (s *Service) proQuota() int {
	if s.ProMonthlyQuota > 0 {
		return s.ProMonthlyQuota
	}
	return 100000
}

func (s *Service) MarkStatus(ctx context.Context, teamID uuid.UUID, status, subscriptionID string, setQuota bool) error {
	if setQuota && status == "active" {
		_, err := s.Pool.Exec(ctx, `
			UPDATE teams SET billing_status = $2, stripe_subscription_id = COALESCE(NULLIF($3,''), stripe_subscription_id),
			       monthly_quota = $4, updated_at = now()
			WHERE id = $1
		`, teamID, status, subscriptionID, s.proQuota())
		return err
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE teams SET billing_status = $2,
		       stripe_subscription_id = COALESCE(NULLIF($3,''), stripe_subscription_id),
		       updated_at = now()
		WHERE id = $1
	`, teamID, status, subscriptionID)
	return err
}

func (s *Service) MarkActive(ctx context.Context, teamID uuid.UUID, subscriptionID string, quota int) error {
	if quota <= 0 {
		quota = s.proQuota()
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE teams SET billing_status = 'active', stripe_subscription_id = $2, monthly_quota = $3, updated_at = now()
		WHERE id = $1
	`, teamID, subscriptionID, quota)
	return err
}

func (s *Service) MarkCanceled(ctx context.Context, teamID uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE teams SET billing_status = 'canceled', updated_at = now() WHERE id = $1
	`, teamID)
	return err
}

func teamIDFromMeta(meta map[string]string, fallback string) (uuid.UUID, error) {
	teamStr := ""
	if meta != nil {
		teamStr = meta["team_id"]
	}
	if teamStr == "" {
		teamStr = fallback
	}
	return uuid.Parse(teamStr)
}

func subscriptionStatusToBilling(status string) string {
	switch status {
	case "active", "trialing":
		return "active"
	case "past_due", "unpaid":
		return "past_due"
	case "canceled", "incomplete_expired":
		return "canceled"
	case "paused":
		return "paused"
	default:
		return ""
	}
}

// HandleStripeEvent activates or cancels a team from a verified Stripe event payload.
func (s *Service) HandleStripeEvent(ctx context.Context, eventType string, data json.RawMessage) error {
	switch eventType {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		var sess struct {
			Metadata          map[string]string `json:"metadata"`
			Subscription      string            `json:"subscription"`
			ClientReferenceID string            `json:"client_reference_id"`
			PaymentStatus     string            `json:"payment_status"`
			Mode              string            `json:"mode"`
		}
		if err := json.Unmarshal(data, &sess); err != nil {
			return err
		}
		if sess.Mode != "" && sess.Mode != "subscription" {
			return nil
		}
		// unpaid async checkout waits for async_payment_succeeded
		if eventType == "checkout.session.completed" && sess.PaymentStatus == "unpaid" {
			return nil
		}
		if sess.PaymentStatus != "" && sess.PaymentStatus != "paid" && sess.PaymentStatus != "no_payment_required" {
			return nil
		}
		teamID, err := teamIDFromMeta(sess.Metadata, sess.ClientReferenceID)
		if err != nil {
			return fmt.Errorf("missing team_id in checkout session")
		}
		sub := sess.Subscription
		if sub == "" {
			sub = "sub_pending"
		}
		return s.MarkActive(ctx, teamID, sub, s.proQuota())

	case "customer.subscription.updated", "customer.subscription.created":
		var sub struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(data, &sub); err != nil {
			return err
		}
		billingStatus := subscriptionStatusToBilling(sub.Status)
		if billingStatus == "" {
			return nil
		}
		if teamID, err := teamIDFromMeta(sub.Metadata, ""); err == nil {
			return s.MarkStatus(ctx, teamID, billingStatus, sub.ID, billingStatus == "active")
		}
		_, err := s.Pool.Exec(ctx, `
			UPDATE teams SET billing_status = $2, updated_at = now(),
			       monthly_quota = CASE WHEN $2 = 'active' THEN GREATEST(monthly_quota, $3) ELSE monthly_quota END
			WHERE stripe_subscription_id = $1
		`, sub.ID, billingStatus, s.proQuota())
		return err

	case "customer.subscription.deleted":
		var sub struct {
			Metadata map[string]string `json:"metadata"`
			ID       string            `json:"id"`
		}
		if err := json.Unmarshal(data, &sub); err != nil {
			return err
		}
		if teamID, err := teamIDFromMeta(sub.Metadata, ""); err == nil {
			return s.MarkCanceled(ctx, teamID)
		}
		_, err := s.Pool.Exec(ctx, `
			UPDATE teams SET billing_status = 'canceled', updated_at = now()
			WHERE stripe_subscription_id = $1
		`, sub.ID)
		return err

	case "invoice.payment_failed":
		var inv struct {
			Subscription string            `json:"subscription"`
			Metadata     map[string]string `json:"metadata"`
			Customer     string            `json:"customer"`
		}
		if err := json.Unmarshal(data, &inv); err != nil {
			return err
		}
		if teamID, err := teamIDFromMeta(inv.Metadata, ""); err == nil {
			return s.MarkStatus(ctx, teamID, "past_due", inv.Subscription, false)
		}
		if inv.Subscription != "" {
			_, err := s.Pool.Exec(ctx, `
				UPDATE teams SET billing_status = 'past_due', updated_at = now()
				WHERE stripe_subscription_id = $1
			`, inv.Subscription)
			return err
		}
		if inv.Customer != "" {
			_, err := s.Pool.Exec(ctx, `
				UPDATE teams SET billing_status = 'past_due', updated_at = now()
				WHERE stripe_customer_id = $1
			`, inv.Customer)
			return err
		}
		return nil

	default:
		return nil
	}
}
