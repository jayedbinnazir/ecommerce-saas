package dto

// CreateAddressRequest is the payload for POST /users/:userId/addresses.
type CreateAddressRequest struct {
	Label             *string `json:"label" binding:"omitempty,max=100"`
	RecipientName     string  `json:"recipient_name" binding:"required,max=255"`
	Phone             string  `json:"phone" binding:"required,max=32"`
	AddressLine1      string  `json:"address_line_1" binding:"required,max=255"`
	AddressLine2      *string `json:"address_line_2" binding:"omitempty,max=255"`
	City              string  `json:"city" binding:"required,max=128"`
	State             *string `json:"state" binding:"omitempty,max=128"`
	PostalCode        *string `json:"postal_code" binding:"omitempty,max=32"`
	Country           string  `json:"country" binding:"required,len=2,uppercase"`
	IsDefaultShipping bool    `json:"is_default_shipping"`
	IsDefaultBilling  bool    `json:"is_default_billing"`
}

// UpdateAddressRequest is the payload for PATCH /users/:userId/addresses/:id.
type UpdateAddressRequest struct {
	Label             *string `json:"label" binding:"omitempty,max=100"`
	RecipientName     *string `json:"recipient_name" binding:"omitempty,max=255"`
	Phone             *string `json:"phone" binding:"omitempty,max=32"`
	AddressLine1      *string `json:"address_line_1" binding:"omitempty,max=255"`
	AddressLine2      *string `json:"address_line_2" binding:"omitempty,max=255"`
	City              *string `json:"city" binding:"omitempty,max=128"`
	State             *string `json:"state" binding:"omitempty,max=128"`
	PostalCode        *string `json:"postal_code" binding:"omitempty,max=32"`
	Country           *string `json:"country" binding:"omitempty,len=2,uppercase"`
	IsDefaultShipping *bool   `json:"is_default_shipping"`
	IsDefaultBilling  *bool   `json:"is_default_billing"`
}
