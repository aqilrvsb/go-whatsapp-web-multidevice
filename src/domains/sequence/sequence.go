package sequence

import (
	"time"
)

// ISequenceUsecase interface for sequence operations
type ISequenceUsecase interface {
	CreateSequence(request CreateSequenceRequest) (SequenceResponse, error)
	GetSequences(userID string) ([]SequenceResponse, error)
	GetSequenceByID(sequenceID string) (SequenceDetailResponse, error)
	UpdateSequence(sequenceID string, request UpdateSequenceRequest) error
	DeleteSequence(sequenceID string) error
	
	// Contact management
	AddContactsToSequence(sequenceID string, contacts []string) error
	RemoveContactFromSequence(sequenceID string, contactID string) error
	GetSequenceContacts(sequenceID string) ([]SequenceContactResponse, error)
	
	// Execution
	StartSequence(sequenceID string) error
	PauseSequence(sequenceID string) error
	ProcessSequences() error // Called by cron job
}

// CreateSequenceRequest for creating new sequence
type CreateSequenceRequest struct {
	Name        string                   `json:"name" validate:"required"`
	Description string                   `json:"description"`
	UserID      string                   `json:"user_id"`
	DeviceID    string                   `json:"device_id" validate:"required"`
	Niche       string                   `json:"niche"` // Auto-trigger based on lead niche
	Steps       []SequenceStepRequest    `json:"steps" validate:"required,min=1"`
	IsActive    bool                     `json:"is_active"`
}

// SequenceStepRequest for each step in sequence
type SequenceStepRequest struct {
	Day         int    `json:"day" validate:"required,min=1"`
	MessageType string `json:"message_type" validate:"required,oneof=text image video document"`
	Content     string `json:"content" validate:"required"`
	MediaURL    string `json:"media_url"`
	Caption     string `json:"caption"`
	SendTime    string `json:"send_time" validate:"required"` // HH:MM format
}

// UpdateSequenceRequest for updating sequence
type UpdateSequenceRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Steps       []SequenceStepRequest `json:"steps"`
	IsActive    bool                  `json:"is_active"`
}

// SequenceResponse basic sequence info
type SequenceResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	Niche        string    `json:"niche"`
	TotalSteps   int       `json:"total_steps"`
	TotalDays    int       `json:"total_days"`
	IsActive     bool      `json:"is_active"`
	ContactCount int       `json:"contact_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SequenceDetailResponse with full details
type SequenceDetailResponse struct {
	SequenceResponse
	Steps []SequenceStepResponse `json:"steps"`
	Stats SequenceStats         `json:"stats"`
}

// SequenceStepResponse for each step
type SequenceStepResponse struct {
	ID          string `json:"id"`
	SequenceID  string `json:"sequence_id"`
	Day         int    `json:"day"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
	MediaURL    string `json:"media_url"`
	Caption     string `json:"caption"`
	SendTime    string `json:"send_time"`
}

// SequenceContactResponse for contacts in sequence
type SequenceContactResponse struct {
	ID            string    `json:"id"`
	ContactPhone  string    `json:"contact_phone"`
	ContactName   string    `json:"contact_name"`
	CurrentDay    int       `json:"current_day"`
	Status        string    `json:"status"` // active, completed, paused
	AddedAt       time.Time `json:"added_at"`
	LastMessageAt time.Time `json:"last_message_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// SequenceStats statistics
type SequenceStats struct {
	TotalContacts    int `json:"total_contacts"`
	ActiveContacts   int `json:"active_contacts"`
	CompletedContacts int `json:"completed_contacts"`
	PausedContacts   int `json:"paused_contacts"`
	MessagesSent     int `json:"messages_sent"`
	MessagesDelivered int `json:"messages_delivered"`
	MessagesRead     int `json:"messages_read"`
}