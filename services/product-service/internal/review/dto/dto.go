// Package dto holds the request/response payloads for the review module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/review/domain"
)

type CreateReviewRequest struct {
	Rating int     `json:"rating" binding:"required,min=1,max=5"`
	Title  *string `json:"title" binding:"omitempty,max=140"`
	Body   *string `json:"body" binding:"omitempty,max=4000"`
}

type ReviewResponse struct {
	ID         uuid.UUID `json:"id"`
	ProductID  uuid.UUID `json:"product_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Rating     int       `json:"rating"`
	Title      *string   `json:"title,omitempty"`
	Body       *string   `json:"body,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func FromReview(r *domain.Review) ReviewResponse {
	return ReviewResponse{
		ID:         r.ID,
		ProductID:  r.ProductID,
		CustomerID: r.CustomerID,
		Rating:     r.Rating,
		Title:      r.Title,
		Body:       r.Body,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func FromReviews(rs []domain.Review) []ReviewResponse {
	out := make([]ReviewResponse, len(rs))
	for i := range rs {
		out[i] = FromReview(&rs[i])
	}
	return out
}

type SummaryResponse struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

func FromSummary(s domain.Summary) SummaryResponse {
	return SummaryResponse{Average: s.Average, Count: s.Count}
}
