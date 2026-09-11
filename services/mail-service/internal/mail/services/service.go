// Package services renders a template and sends it, logging the attempt.
package services

import (
	"context"
	"strings"

	"github.com/jayedbinnazir/mail-service/internal/mail/domain"
	"github.com/jayedbinnazir/mail-service/internal/mail/dto"
	"github.com/jayedbinnazir/mail-service/internal/mail/templates"
	"github.com/jayedbinnazir/mail-service/internal/smtp"
)

type Service struct {
	repo   domain.Repository
	sender smtp.Sender
}

func New(repo domain.Repository, sender smtp.Sender) *Service {
	return &Service{repo: repo, sender: sender}
}

// Send renders req.Template with req.Data and delivers it. The attempt is logged
// whether it succeeds or fails; a delivery failure is returned as ErrSendFailed.
func (s *Service) Send(ctx context.Context, req dto.SendRequest) (*domain.Mail, error) {
	to := strings.TrimSpace(req.To)

	rendered, err := templates.Render(req.Template, req.Data)
	if err != nil {
		return nil, err // domain.ErrUnknownTemplate or a template parse error
	}

	mail := &domain.Mail{
		ToAddr:   to,
		Subject:  rendered.Subject,
		Template: req.Template,
		Status:   domain.StatusSent,
	}

	if sendErr := s.sender.Send(ctx, to, rendered.Subject, rendered.HTML); sendErr != nil {
		msg := sendErr.Error()
		mail.Status = domain.StatusFailed
		mail.Error = &msg
		_ = s.repo.Create(ctx, mail)
		return mail, domain.ErrSendFailed
	}

	if err := s.repo.Create(ctx, mail); err != nil {
		return nil, err
	}
	return mail, nil
}

func (s *Service) ListByRecipient(ctx context.Context, toAddr string, limit int) ([]domain.Mail, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListByRecipient(ctx, strings.TrimSpace(toAddr), limit)
}

// Templates lists the available template names.
func (s *Service) Templates() []string { return templates.Names() }
