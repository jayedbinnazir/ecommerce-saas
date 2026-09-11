// Package services holds the billing logic. Payment for a subscription is
// stubbed: subscribing simply records an ACTIVE subscription for one interval.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/billing/domain"
	"github.com/jayedbinnazir/payment-service/internal/billing/dto"
)

type Service struct {
	plans domain.PlanRepository
	subs  domain.SubscriptionRepository
}

func New(plans domain.PlanRepository, subs domain.SubscriptionRepository) *Service {
	return &Service{plans: plans, subs: subs}
}

// ListPlans returns the plans a customer can pick from.
func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx, true)
}

// GetMySubscription returns the caller's active subscription.
func (s *Service) GetMySubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return s.subs.GetActiveByUser(ctx, userID)
}

// Subscribe records a new ACTIVE subscription for the chosen plan.
func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req dto.SubscribeRequest) (*domain.Subscription, error) {
	plan, err := s.plans.GetByCode(ctx, req.PlanCode)
	if err != nil {
		return nil, err
	}
	if !plan.Active {
		return nil, domain.ErrPlanInactive
	}

	now := time.Now().UTC()
	sub := &domain.Subscription{
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             domain.SubActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   plan.Interval.AddTo(now),
	}
	if err := s.subs.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// Cancel marks the caller's active subscription as canceled. Access continues
// until the end of the paid period.
func (s *Service) Cancel(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sub.Status = domain.SubCanceled
	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// Status is the internal entitlement check user-management calls before store
// creation. It never errors on "no subscription" — it returns active=false.
func (s *Service) Status(ctx context.Context, userID uuid.UUID) (dto.SubscriptionStatusResponse, error) {
	sub, err := s.subs.GetActiveByUser(ctx, userID)
	if errors.Is(err, domain.ErrSubscriptionNotFound) {
		return dto.SubscriptionStatusResponse{Active: false}, nil
	}
	if err != nil {
		return dto.SubscriptionStatusResponse{}, err
	}

	resp := dto.SubscriptionStatusResponse{
		Active:           sub.IsEntitled(time.Now().UTC()),
		Status:           string(sub.Status),
		CurrentPeriodEnd: &sub.CurrentPeriodEnd,
	}
	if plan, err := s.plans.GetByID(ctx, sub.PlanID); err == nil {
		resp.PlanCode = plan.Code
	}
	return resp, nil
}
