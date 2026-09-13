package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
)

// ---- User ----

type UserResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        *string   `json:"phone,omitempty"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func FromUser(u *domain.User) UserResponse {
	return UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Phone:        u.Phone,
		IsSuperAdmin: u.IsSuperAdmin,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func FromUsers(us []domain.User) []UserResponse {
	out := make([]UserResponse, len(us))
	for i := range us {
		out[i] = FromUser(&us[i])
	}
	return out
}

// ---- Address ----

type AddressResponse struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	Label             *string   `json:"label,omitempty"`
	RecipientName     string    `json:"recipient_name"`
	Phone             string    `json:"phone"`
	AddressLine1      string    `json:"address_line_1"`
	AddressLine2      *string   `json:"address_line_2,omitempty"`
	City              string    `json:"city"`
	State             *string   `json:"state,omitempty"`
	PostalCode        *string   `json:"postal_code,omitempty"`
	Country           string    `json:"country"`
	IsDefaultShipping bool      `json:"is_default_shipping"`
	IsDefaultBilling  bool      `json:"is_default_billing"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func FromAddress(a *domain.Address) AddressResponse {
	return AddressResponse{
		ID:                a.ID,
		UserID:            a.UserID,
		Label:             a.Label,
		RecipientName:     a.RecipientName,
		Phone:             a.Phone,
		AddressLine1:      a.AddressLine1,
		AddressLine2:      a.AddressLine2,
		City:              a.City,
		State:             a.State,
		PostalCode:        a.PostalCode,
		Country:           a.Country,
		IsDefaultShipping: a.IsDefaultShipping,
		IsDefaultBilling:  a.IsDefaultBilling,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

func FromAddresses(as []domain.Address) []AddressResponse {
	out := make([]AddressResponse, len(as))
	for i := range as {
		out[i] = FromAddress(&as[i])
	}
	return out
}
