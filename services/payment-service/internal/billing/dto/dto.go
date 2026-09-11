// Package dto holds the request/response payloads for the billing module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/billing/domain"
)

// SubscribeRequest is the (stubbed) checkout payload — no card details.
type SubscribeRequest struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

type PlanResponse struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Interval   string    `json:"interval"`    // MONTH | YEAR
	PriceCents int64     `json:"price_cents"` // what is charged each period
	// MonthlyEquivalentCents lets the UI show "$25/mo billed yearly".
	MonthlyEquivalentCents int64  `json:"monthly_equivalent_cents"`
	Currency               string `json:"currency"`
}

func FromPlan(p *domain.Plan) PlanResponse {
	monthly := p.PriceCents
	if p.Interval == domain.IntervalYear {
		monthly = p.PriceCents / 12
	}
	return PlanResponse{
		ID:                     p.ID,
		Code:                   p.Code,
		Name:                   p.Name,
		Interval:               string(p.Interval),
		PriceCents:             p.PriceCents,
		MonthlyEquivalentCents: monthly,
		Currency:               p.Currency,
	}
}

func FromPlans(ps []domain.Plan) []PlanResponse {
	out := make([]PlanResponse, len(ps))
	for i := range ps {
		out[i] = FromPlan(&ps[i])
	}
	return out
}

type SubscriptionResponse struct {
	ID                 uuid.UUID `json:"id"`
	PlanID             uuid.UUID `json:"plan_id"`
	PlanCode           string    `json:"plan_code,omitempty"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
}

func FromSubscription(s *domain.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:                 s.ID,
		PlanID:             s.PlanID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
	}
}

// SubscriptionStatusResponse is the internal shape user-management reads before
// letting a user create a store.
type SubscriptionStatusResponse struct {
	Active           bool       `json:"active"`
	PlanCode         string     `json:"plan_code,omitempty"`
	Status           string     `json:"status,omitempty"`
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
}
