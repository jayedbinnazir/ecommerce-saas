// Package services stores in-app notifications and, when asked, forwards an
// email to mail-service.
package services

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/notification-service/internal/mailclient"
	"github.com/jayedbinnazir/notification-service/internal/notification/domain"
	"github.com/jayedbinnazir/notification-service/internal/notification/dto"
	"github.com/jayedbinnazir/notification-service/internal/platform"
)

type Service struct {
	repo domain.Repository
	mail *mailclient.Client
}

func New(repo domain.Repository, mail *mailclient.Client) *Service {
	return &Service{repo: repo, mail: mail}
}

// Create stores a notification. If the request carries an email + template it
// also asks mail-service to send it (best-effort — a mail failure is logged).
func (s *Service) Create(ctx context.Context, req dto.CreateRequest) (*domain.Notification, error) {
	if strings.TrimSpace(req.Type) == "" {
		return nil, domain.ErrInvalidType
	}

	n := &domain.Notification{
		UserID: req.UserID,
		Type:   strings.TrimSpace(req.Type),
		Title:  strings.TrimSpace(req.Title),
		Body:   strings.TrimSpace(req.Body),
	}
	if req.TenantID != nil && *req.TenantID != "" {
		tid := uuid.MustParse(*req.TenantID)
		n.TenantID = &tid
	}
	if req.Data != nil {
		if raw, err := json.Marshal(req.Data); err == nil {
			n.Data = raw
		}
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	if req.Email != "" && req.EmailTemplate != "" {
		if err := s.mail.Send(ctx, req.Email, req.EmailTemplate, req.Data); err != nil {
			log.Printf("[notification] mail to %s (%s) failed: %v", req.Email, req.EmailTemplate, err)
		}
	}
	return n, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page platform.Page) ([]domain.Notification, error) {
	return s.repo.ListByUser(ctx, domain.ListFilter{UserID: userID, UnreadOnly: unreadOnly, Page: page})
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead marks read: all of the user's unread notifications when all is true,
// otherwise just the given ids. Returns how many rows changed.
func (s *Service) MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, all bool) (int, error) {
	if all {
		return s.repo.MarkAllRead(ctx, userID)
	}
	return s.repo.MarkRead(ctx, userID, ids)
}
