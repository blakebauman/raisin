package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/raisin-run/raisin/internal/db"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
)

type Service struct {
	Pool      *db.Pool
	SecretKey string
}

type Usage struct {
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	EmailsSent  int       `json:"emails_sent"`
	Quota       int       `json:"quota"`
	Remaining   int       `json:"remaining"`
	Status      string    `json:"billing_status"`
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

func (s *Service) CreateCheckout(ctx context.Context, teamID uuid.UUID, teamName, successURL, cancelURL string) (string, error) {
	if s.SecretKey == "" {
		return "", fmt.Errorf("stripe not configured")
	}
	stripe.Key = s.SecretKey

	var customerID *string
	_ = s.Pool.QueryRow(ctx, `SELECT stripe_customer_id FROM teams WHERE id = $1`, teamID).Scan(&customerID)

	if customerID == nil || *customerID == "" {
		c, err := customer.New(&stripe.CustomerParams{
			Name: stripe.String(teamName),
			Metadata: map[string]string{"team_id": teamID.String()},
		})
		if err != nil {
			return "", err
		}
		_, _ = s.Pool.Exec(ctx, `UPDATE teams SET stripe_customer_id = $2 WHERE id = $1`, teamID, c.ID)
		customerID = &c.ID
	}

	sess, err := session.New(&stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   customerID,
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Raisin Pro"),
					},
					UnitAmount: stripe.Int64(2000),
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String("month"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{"team_id": teamID.String()},
	})
	if err != nil {
		return "", err
	}
	return sess.URL, nil
}

func (s *Service) MarkActive(ctx context.Context, teamID uuid.UUID, subscriptionID string, quota int) error {
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

// HandleStripeEvent activates or cancels a team from a verified Stripe event payload.
func (s *Service) HandleStripeEvent(ctx context.Context, eventType string, data json.RawMessage) error {
	switch eventType {
	case "checkout.session.completed":
		var sess struct {
			Metadata       map[string]string `json:"metadata"`
			Subscription   string            `json:"subscription"`
			ClientReference string           `json:"client_reference_id"`
		}
		if err := json.Unmarshal(data, &sess); err != nil {
			return err
		}
		teamStr := sess.Metadata["team_id"]
		if teamStr == "" {
			teamStr = sess.ClientReference
		}
		teamID, err := uuid.Parse(teamStr)
		if err != nil {
			return fmt.Errorf("missing team_id in checkout session")
		}
		sub := sess.Subscription
		if sub == "" {
			sub = "sub_pending"
		}
		return s.MarkActive(ctx, teamID, sub, 100000)
	case "customer.subscription.deleted":
		var sub struct {
			Metadata map[string]string `json:"metadata"`
			ID       string            `json:"id"`
		}
		if err := json.Unmarshal(data, &sub); err != nil {
			return err
		}
		if teamStr := sub.Metadata["team_id"]; teamStr != "" {
			teamID, err := uuid.Parse(teamStr)
			if err == nil {
				return s.MarkCanceled(ctx, teamID)
			}
		}
		_, err := s.Pool.Exec(ctx, `
			UPDATE teams SET billing_status = 'canceled', updated_at = now()
			WHERE stripe_subscription_id = $1
		`, sub.ID)
		return err
	default:
		return nil
	}
}
