package domain

import "errors"

var (
	ErrUnknownTemplate  = errors.New("no such email template")
	ErrInvalidRecipient = errors.New("recipient must be a valid email address")
	ErrSendFailed       = errors.New("the mail relay rejected the message")
)
