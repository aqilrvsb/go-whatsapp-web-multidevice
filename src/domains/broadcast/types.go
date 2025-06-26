package broadcast

import (
	"time"
)

// BroadcastMessage represents a message to be sent
type BroadcastMessage struct {
	ID           string
	DeviceID     string
	Type         string // campaign, sequence
	ReferenceID  string
	Phone        string
	Content      string
	MediaURL     string
	Caption      string
	Priority     int
	ScheduledAt  time.Time
	RetryCount   int
}