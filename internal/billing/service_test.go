package billing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestHandleStripeEventCheckoutActivates(t *testing.T) {
	// Pure parsing path: ensure unpaid completed is ignored without DB.
	s := &Service{ProMonthlyQuota: 50000}
	team := uuid.New()
	payload, _ := json.Marshal(map[string]any{
		"mode":            "subscription",
		"payment_status":  "unpaid",
		"subscription":    "sub_x",
		"metadata":        map[string]string{"team_id": team.String()},
		"client_reference_id": team.String(),
	})
	if err := s.HandleStripeEvent(context.Background(), "checkout.session.completed", payload); err != nil {
		t.Fatalf("unpaid completed should no-op: %v", err)
	}
}

func TestSubscriptionStatusMapping(t *testing.T) {
	cases := map[string]string{
		"active":   "active",
		"trialing": "active",
		"past_due": "past_due",
		"canceled": "canceled",
		"bogus":    "",
	}
	for in, want := range cases {
		if got := subscriptionStatusToBilling(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestPriceIDFor(t *testing.T) {
	s := &Service{PricePro: "price_month", PriceProAnnual: "price_year"}
	id, err := s.priceIDFor("month")
	if err != nil || id != "price_month" {
		t.Fatalf("month: %v %s", err, id)
	}
	id, err = s.priceIDFor("annual")
	if err != nil || id != "price_year" {
		t.Fatalf("annual: %v %s", err, id)
	}
	s2 := &Service{}
	if _, err := s2.priceIDFor("month"); err == nil {
		t.Fatal("expected missing price error")
	}
}

func TestPlansConfigured(t *testing.T) {
	s := &Service{SecretKey: "sk_test", PricePro: "price_x", ProMonthlyQuota: 100000}
	p := s.Plans()
	if !p.Configured || !p.CheckoutReady || p.MonthlyQuota != 100000 {
		t.Fatalf("unexpected plans: %+v", p)
	}
}
