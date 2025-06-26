package models

import (
	"time"
)

// Sequence model for drip campaigns
type Sequence struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Niche       string    `json:"niche" db:"niche"` // Auto-trigger based on lead niche
	TotalDays   int       `json:"total_days" db:"total_days"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SequenceStep model for each message in sequence
type SequenceStep struct {
	ID          string    `json:"id" db:"id"`
	SequenceID  string    `json:"sequence_id" db:"sequence_id"`
	Day         int       `json:"day" db:"day"`
	MessageType string    `json:"message_type" db:"message_type"`
	Content     string    `json:"content" db:"content"`
	MediaURL    string    `json:"media_url" db:"media_url"`
	Caption     string    `json:"caption" db:"caption"`
	SendTime    string    `json:"send_time" db:"send_time"` // HH:MM format
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SequenceContact model for contacts in sequence
type SequenceContact struct {
	ID            string     `json:"id" db:"id"`
	SequenceID    string     `json:"sequence_id" db:"sequence_id"`
	ContactPhone  string     `json:"contact_phone" db:"contact_phone"`
	ContactName   string     `json:"contact_name" db:"contact_name"`
	CurrentDay    int        `json:"current_day" db:"current_day"`
	Status        string     `json:"status" db:"status"` // active, completed, paused
	AddedAt       time.Time  `json:"added_at" db:"added_at"`
	LastMessageAt *time.Time `json:"last_message_at" db:"last_message_at"`
	CompletedAt   *time.Time `json:"completed_at" db:"completed_at"`
}

// SequenceLog model for tracking sent messages
type SequenceLog struct {
	ID           string    `json:"id" db:"id"`
	SequenceID   string    `json:"sequence_id" db:"sequence_id"`
	ContactID    string    `json:"contact_id" db:"contact_id"`
	StepID       string    `json:"step_id" db:"step_id"`
	Day          int       `json:"day" db:"day"`
	Status       string    `json:"status" db:"status"` // sent, delivered, read, failed
	MessageID    string    `json:"message_id" db:"message_id"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	SentAt       time.Time `json:"sent_at" db:"sent_at"`
}