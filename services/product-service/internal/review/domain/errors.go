package domain

import "errors"

var (
	ErrReviewNotFound = errors.New("review not found")
	ErrInvalidRating  = errors.New("rating must be between 1 and 5")
	ErrNotAuthor      = errors.New("you can only delete your own review")
)
