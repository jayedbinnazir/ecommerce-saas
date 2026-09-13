// Package services holds the billing logic. Payment for a subscription is
// stubbed: subscribing/renewing simply records/extends an ACTIVE subscription
// for one interval, with no real card charge.
package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/billing/domain"
	"github.com/jayedbinnazir/payment-service/internal/billing/dto"
	"github.com/jayedbinnazir/payment-service/internal/billing/repository"
	"github.com/jayedbinnazir/payment-service/internal/platform"
)

type Service struct {
	db    *sql.DB
	plans domain.PlanRepository
	subs  domain.SubscriptionRepository
}

func New(db *sql.DB, plans domain.PlanRepository, subs domain.SubscriptionRepository) *Service {
	return &Service{db: db, plans: plans, subs: subs}
}

// ListPlans returns the plans a customer can pick from.
func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx, true)
}

// GetMySubscription returns the caller's active subscription.
func (s *Service) GetMySubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return s.subs.GetActiveByUser(ctx, userID)
}

// Subscribe records or renews the caller's subscription for the chosen plan:
//
//   - no subscription ever existed for this user   -> create a fresh ACTIVE row
//   - an ACTIVE, not-yet-lapsed subscription exists -> RENEWAL: extend
//     current_period_end by one more interval (from the current end, so
//     paid-for time is never lost), same row.
//   - a CANCELED/EXPIRED/PAST_DUE subscription exists -> reactivate that same
//     row: ACTIVE again, a fresh period starting now.
//
// idempotencyKey, when non-empty (the Idempotency-Key header), makes a
// retried/duplicate renewal request a no-op instead of extending the period
// twice: the row is locked for the duration of the check-and-update, so two
// concurrent requests with the same key serialize and the second one sees
// its key already recorded.
func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req dto.SubscribeRequest, idempotencyKey string) (*domain.Subscription, error) {
	plan, err := s.plans.GetByCode(ctx, req.PlanCode)
	if err != nil {
		return nil, err
	}
	if !plan.Active {
		return nil, domain.ErrPlanInactive
	}

	var result *domain.Subscription
	err = platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		txSubs := repository.NewSubscriptionRepository(tx)
		now := time.Now().UTC()

		existing, err := txSubs.GetLatestByUserForUpdate(ctx, userID)
		if err != nil && !errors.Is(err, domain.ErrSubscriptionNotFound) {
			return err
		}

		if existing == nil {
			sub := &domain.Subscription{
				UserID:             userID,
				PlanID:             plan.ID,
				Status:             domain.SubActive,
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   plan.Interval.AddTo(now),
			}
			if idempotencyKey != "" {
				sub.LastRenewalKey = &idempotencyKey
			}
			if err := txSubs.Create(ctx, sub); err != nil {
				return err
			}
			result = sub
			return nil
		}

		if idempotencyKey != "" && existing.LastRenewalKey != nil && *existing.LastRenewalKey == idempotencyKey {
			result = existing // duplicate renewal request already applied -- no-op
			return nil
		}

		if existing.Status == domain.SubActive && now.Before(existing.CurrentPeriodEnd) {
			existing.PlanID = plan.ID
			existing.CurrentPeriodEnd = plan.Interval.AddTo(existing.CurrentPeriodEnd)
		} else {
			existing.Status = domain.SubActive
			existing.PlanID = plan.ID
			existing.CurrentPeriodStart = now
			existing.CurrentPeriodEnd = plan.Interval.AddTo(now)
		}
		if idempotencyKey != "" {
			existing.LastRenewalKey = &idempotencyKey
		}
		if err := txSubs.Update(ctx, existing); err != nil {
			return err
		}
		result = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// MarkPastDue records that a recurring-charge attempt for the user's active
// subscription failed. It is the internal entry point a real Stripe
// "invoice.payment_failed" webhook would call once real recurring billing
// exists; today it is invoked directly (X-Internal-Key) since Subscribe/renew
// never performs a real charge to fail on its own.
//
// This is a grace-period transition, not a suspension: current_period_end is
// left untouched (so a still-current PAST_DUE subscription is indistinguishable
// from ACTIVE except that IsEntitled treats it as not-entitled for *new*
// tenant creation), and nothing about the user's existing tenant(s) changes --
// tenant data, memberships and orders are never touched here.
func (s *Service) MarkPastDue(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sub.Status = domain.SubPastDue
	if err := s.subs.Update(ctx, sub); err != nil {
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
