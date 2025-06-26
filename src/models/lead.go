package models

import (
	"time"
)

// Lead represents a lead/contact
type Lead struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Phone     string    `json:"phone" db:"phone"`
	Email     string    `json:"email" db:"email"`
	Niche     string    `json:"niche" db:"niche"` // For matching with campaigns/sequences
	Source    string    `json:"source" db:"source"`
	Status    string    `json:"status" db:"status"`
	Notes     string    `json:"notes" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}