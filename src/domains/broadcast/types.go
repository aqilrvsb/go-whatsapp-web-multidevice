package broadcast

import "time"

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	ID            string
	DeviceID      string
	CampaignID    string
	SequenceID    string
	RecipientPhone string
	Type          string // text, image, video, document
	Content       string
	MediaURL      string
	Caption       string
	ScheduledAt   time.Time
	Status        string
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
