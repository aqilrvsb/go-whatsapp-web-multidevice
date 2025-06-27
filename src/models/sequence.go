package models

import (
	"time"
)

// Sequence model for drip campaigns - simplified like campaigns
type Sequence struct {
	ID              string         `json:"id" db:"id"`
	UserID          string         `json:"user_id" db:"user_id"`
	DeviceID        *string        `json:"device_id" db:"device_id"` // Nullable - sequences use all user devices
	Name            string         `json:"name" db:"name"`
	Description     string         `json:"description" db:"description"`
	Niche           string         `json:"niche" db:"niche"`
	TargetStatus    string         `json:"target_status" db:"target_status"` // prospect, customer, all
	Status          string         `json:"status" db:"status"` // draft, active, paused
	TotalDays       int            `json:"total_days" db:"total_days"` // Added
	IsActive        bool           `json:"is_active" db:"is_active"`   // Added
	ScheduleTime    string         `json:"schedule_time" db:"schedule_time"`
	MinDelaySeconds int            `json:"min_delay_seconds" db:"min_delay_seconds"`
	MaxDelaySeconds int            `json:"max_delay_seconds" db:"max_delay_seconds"`
	ContactsCount   int            `json:"contacts_count" db:"contacts_count"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	Steps           []SequenceStep `json:"steps,omitempty"`
}

// SequenceStep model for each day in sequence - simplified
type SequenceStep struct {
	ID               string    `json:"id" db:"id"`
	SequenceID       string    `json:"sequence_id" db:"sequence_id"`
	Day              int       `json:"day" db:"day"`
	DayNumber        int       `json:"day_number" db:"day_number"`
	MessageType      string    `json:"message_type" db:"message_type"`
	SendTime         string    `json:"send_time" db:"send_time"`
	ScheduleTime     string    `json:"schedule_time" db:"schedule_time"`
	Content          string    `json:"content" db:"content"`
	MediaURL         string    `json:"media_url" db:"media_url"`
	ImageURL         string    `json:"image_url" db:"image_url"`
	Caption          string    `json:"caption" db:"caption"`
	MinDelaySeconds  int       `json:"min_delay_seconds" db:"min_delay_seconds"`
	MaxDelaySeconds  int       `json:"max_delay_seconds" db:"max_delay_seconds"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// SequenceContact model for tracking individual progress
type SequenceContact struct {
	ID             string     `json:"id" db:"id"`
	SequenceID     string     `json:"sequence_id" db:"sequence_id"`
	ContactPhone   string     `json:"contact_phone" db:"contact_phone"`
	ContactName    string     `json:"contact_name" db:"contact_name"`
	CurrentStep    int        `json:"current_step" db:"current_step"`
	CurrentDay     int        `json:"current_day" db:"current_day"`
	Status         string     `json:"status" db:"status"` // active, completed, paused
	AddedAt        time.Time  `json:"added_at" db:"added_at"`
	LastMessageAt  *time.Time `json:"last_message_at" db:"last_message_at"`
	EnrolledAt     time.Time  `json:"enrolled_at" db:"enrolled_at"`
	LastSentAt     *time.Time `json:"last_sent_at" db:"last_sent_at"`
	NextSendAt     *time.Time `json:"next_send_at" db:"next_send_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
}

// SequenceLog model for tracking message history
type SequenceLog struct {
	ID           string    `json:"id" db:"id"`
	SequenceID   string    `json:"sequence_id" db:"sequence_id"`
	ContactID    string    `json:"contact_id" db:"contact_id"`
	StepID       string    `json:"step_id" db:"step_id"`
	Day          int       `json:"day" db:"day"`
	Status       string    `json:"status" db:"status"` // sent, failed
	MessageID    string    `json:"message_id" db:"message_id"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	SentAt       time.Time `json:"sent_at" db:"sent_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
