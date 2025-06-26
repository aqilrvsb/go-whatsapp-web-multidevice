package repository

import (
	"database/sql"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/google/uuid"
)

type BroadcastRepository struct {
	db *sql.DB
}

var broadcastRepo *BroadcastRepository

// GetBroadcastRepository returns broadcast repository instance
func GetBroadcastRepository() *BroadcastRepository {
	if broadcastRepo == nil {
		broadcastRepo = &BroadcastRepository{
			db: database.GetDB(),
		}
	}
	return broadcastRepo
}

// QueueMessage adds a message to the queue
func (r *BroadcastRepository) QueueMessage(msg domainBroadcast.BroadcastMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	
	query := `
		INSERT INTO broadcast_messages 
		(id, user_id, device_id, campaign_id, sequence_id, recipient_phone, 
		 message_type, content, media_url, status, scheduled_at, created_at, group_id, group_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	// Get user_id - use from message if provided, otherwise get from device
	var userID string
	if msg.UserID != "" {
		userID = msg.UserID
	} else {
		err := r.db.QueryRow("SELECT user_id FROM user_devices WHERE id = $1", msg.DeviceID).Scan(&userID)
		if err != nil {
			return err
		}
	}
	
	// Handle nullable fields
	var campaignID interface{}
	if msg.CampaignID != nil {
		campaignID = *msg.CampaignID
	} else {
		campaignID = nil
	}
	
	var sequenceID interface{}
	if msg.SequenceID != nil {
		sequenceID = *msg.SequenceID
	} else {
		sequenceID = nil
	}
	
	var groupID interface{}
	if msg.GroupID != nil {
		groupID = *msg.GroupID
	} else {
		groupID = nil
	}
	
	var groupOrder interface{}
	if msg.GroupOrder != nil {
		groupOrder = *msg.GroupOrder
	} else {
		groupOrder = nil
	}
	
	_, err := r.db.Exec(query, msg.ID, userID, msg.DeviceID, campaignID,
		sequenceID, msg.RecipientPhone, msg.Type, msg.Content, 
		msg.MediaURL, "pending", msg.ScheduledAt, time.Now(), groupID, groupOrder)
		
	return err
}

// GetPendingMessages gets pending messages for a device
func (r *BroadcastRepository) GetPendingMessages(deviceID string, limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT id, user_id, device_id, campaign_id, sequence_id, recipient_phone, 
		       message_type, content, media_url, scheduled_at, group_id, group_order
		FROM broadcast_messages
		WHERE device_id = $1 AND status = 'pending'
		AND (scheduled_at IS NULL OR scheduled_at <= $2)
		ORDER BY group_id NULLS LAST, group_order NULLS LAST, created_at ASC
		LIMIT $3
	`
	
	rows, err := r.db.Query(query, deviceID, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var messages []domainBroadcast.BroadcastMessage
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var userID sql.NullString
		var campaignID sql.NullInt64
		var sequenceID, groupID sql.NullString
		var groupOrder sql.NullInt64
		var scheduledAt sql.NullTime
		
		err := rows.Scan(&msg.ID, &userID, &msg.DeviceID, &campaignID, &sequenceID,
			&msg.RecipientPhone, &msg.Type, &msg.Content, &msg.MediaURL, &scheduledAt,
			&groupID, &groupOrder)
		if err != nil {
			continue
		}
		
		if userID.Valid {
			msg.UserID = userID.String
		}
		if campaignID.Valid {
			campaignIDInt := int(campaignID.Int64)
			msg.CampaignID = &campaignIDInt
		}
		if sequenceID.Valid {
			msg.SequenceID = &sequenceID.String
		}
		if groupID.Valid {
			msg.GroupID = &groupID.String
		}
		if groupOrder.Valid {
			groupOrderInt := int(groupOrder.Int64)
			msg.GroupOrder = &groupOrderInt
		}
		if scheduledAt.Valid {
			msg.ScheduledAt = scheduledAt.Time
		}
		
		messages = append(messages, msg)
	}
	
	return messages, nil
}

// UpdateMessageStatus updates message status
func (r *BroadcastRepository) UpdateMessageStatus(messageID, status, errorMsg string) error {
	query := `
		UPDATE broadcast_messages 
		SET status = $1, error_message = $2, sent_at = CASE WHEN $1 = 'sent' THEN $3 ELSE sent_at END
		WHERE id = $4
	`
	
	_, err := r.db.Exec(query, status, errorMsg, time.Now(), messageID)
	return err
}

// GetBroadcastStats gets broadcast statistics
func (r *BroadcastRepository) GetBroadcastStats(deviceID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get counts by status
	query := `
		SELECT status, COUNT(*) as count
		FROM broadcast_messages
		WHERE device_id = $1 AND created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
		GROUP BY status
	`
	
	rows, err := r.db.Query(query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			statusCounts[status] = count
		}
	}
	
	stats["status_counts"] = statusCounts
	stats["total_24h"] = statusCounts["sent"] + statusCounts["failed"] + statusCounts["pending"]
	
	return stats, nil
}
