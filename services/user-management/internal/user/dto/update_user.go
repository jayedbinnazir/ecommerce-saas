package dto

// UpdateUserRequest is the payload for PATCH /users/:userId. All fields optional;
// nil means "leave unchanged". Password changes go through the auth module.
type UpdateUserRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=255"`
	Email *string `json:"email" binding:"omitempty,email"`
	Phone *string `json:"phone" binding:"omitempty,max=32"`
}

func (r UpdateUserRequest) IsEmpty() bool {
	return r.Name == nil && r.Email == nil && r.Phone == nil
}
