package domain

import (
	"time"

	"github.com/google/uuid"
)

// Image is a product photo stored in object storage (S3). URL is the browser
// URL; StorageKey is the S3 object key, kept so the file can be deleted.
// Table "product_images".
type Image struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	ProductID  uuid.UUID `json:"product_id"  db:"product_id"`
	URL        string    `json:"url"         db:"url"`
	StorageKey string    `json:"-"           db:"storage_key"`
	Alt        *string   `json:"alt,omitempty" db:"alt"`
	Position   int       `json:"position"    db:"position"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}
