package models

import (
	"encoding/json"
	"time"
)

// Campaign represents a marketing campaign
type Campaign struct {
	ID              int        `json:"id" db:"id"`
	UserID          string     `json:"user_id" db:"user_id"`
	DeviceID        string     `json:"device_id" db:"device_id"`
	Title           string     `json:"title" db:"title"`
	Niche           string     `json:"niche" db:"niche"`
	Message         string     `json:"message" db:"message"`
	ImageURL        string     `json:"image_url" db:"image_url"`
	CampaignDate    string     `json:"campaign_date" db:"campaign_date"`
	ScheduledDate   string     `json:"scheduled_date" db:"scheduled_date"`
	ScheduledTime   *time.Time `json:"-" db:"scheduled_time"`
	MinDelaySeconds int        `json:"min_delay_seconds" db:"min_delay_seconds"`
	MaxDelaySeconds int        `json:"max_delay_seconds" db:"max_delay_seconds"`
	Status          string     `json:"status" db:"status"` // pending, sent, failed
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// MarshalJSON customizes JSON output for Campaign
func (c Campaign) MarshalJSON() ([]byte, error) {
	type Alias Campaign
	var scheduledTimeStr string
	if c.ScheduledTime != nil {
		scheduledTimeStr = c.ScheduledTime.Format("15:04")
	}
	return json.Marshal(&struct {
		ScheduledTime string `json:"scheduled_time"`
		*Alias
	}{
		ScheduledTime: scheduledTimeStr,
		Alias:         (*Alias)(&c),
	})
}
