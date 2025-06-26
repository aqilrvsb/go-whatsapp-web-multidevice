package models

import (
	"time"
)

// Sequence model for drip campaigns - simplified like campaigns
type Sequence struct {
	ID              string         `json:"id" db:"id"`
	UserID          string         `json:"user_id" db:"user_id"`
	Name            string         `json:"name" db:"name"`
	Description     string         `json:"description" db:"description"`
	Niche           string         `json:"niche" db:"niche"`
	Status          string         `json:"status" db:"status"` // draft, active, paused
	ContactsCount   int            `json:"contacts_count" db:"contacts_count"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	Steps           []SequenceStep `json:"steps,omitempty"`
}

// SequenceStep model for each day in sequence - simplified
type SequenceStep struct {
	ID               string    `json:"id" db:"id"`
	SequenceID       string    `json:"sequence_id" db:"sequence_id"`
	DayNumber        int       `json:"day_number" db:"day_number"`
	Content          string    `json:"content" db:"content"`
	ImageURL         string    `json:"image_url" db:"image_url"`
	MinDelaySeconds  int       `json:"min_delay_seconds" db:"min_delay_seconds"`
	MaxDelaySeconds  int       `json:"max_delay_seconds" db:"max_delay_seconds"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// SequenceContact model for tracking individual progress
type SequenceContact struct {
	ID             string     `json:"id" db:"id"`
	SequenceID     string     `json:"sequence_id" db:"sequence_id"`
	ContactPhone   string     `json:"contact_phone" db:"contact_phone"`
	ContactName    string     `json:"contact_name" db:"contact_name"`
	CurrentStep    int        `json:"current_step" db:"current_step"`
	Status         string     `json:"status" db:"status"` // active, completed, paused
	EnrolledAt     time.Time  `json:"enrolled_at" db:"enrolled_at"`
	LastSentAt     *time.Time `json:"last_sent_at" db:"last_sent_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
}
