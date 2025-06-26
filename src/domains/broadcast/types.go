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
	Type           string // text, image, video, document
	Content        string
	MediaURL       string
	Caption        string
	ScheduledAt    time.Time
	Status         string
	GroupID        *string // For grouping related messages (e.g., image + text)
	GroupOrder     *int    // Order within the group (1 for image, 2 for text)
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
