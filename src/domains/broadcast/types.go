package broadcast

import "time"

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	ID             string
	UserID         string
	DeviceID       string
	CampaignID     *int    // Pointer to allow null
	SequenceID     *string // Pointer to allow null
	RecipientPhone string
	RecipientJID   string  // WhatsApp JID format
	Type           string  // text, image, video, document
	Content        string
	Message        string  // Alias for Content
	MediaURL       string
	ImageURL       string  // Alias for MediaURL
	Caption        string
	ScheduledAt    time.Time
	Status         string
	GroupID        string  // For grouping related messages
	GroupOrder     int     // Order within the group
	RetryCount     int     // Number of retry attempts
	CreatedAt      time.Time
	// Delay settings from campaign/sequence
	MinDelay       int
	MaxDelay       int
}

// WorkerStatus represents the status of a device worker
type WorkerStatus struct {
	DeviceID       string
	Status         string
	QueueSize      int
	ProcessedCount int
	FailedCount    int
	LastActivity   time.Time
}

// BroadcastRequest represents a broadcast request
type BroadcastRequest struct {
	DeviceID   string
	Recipients []string
	Message    BroadcastMessage
}
