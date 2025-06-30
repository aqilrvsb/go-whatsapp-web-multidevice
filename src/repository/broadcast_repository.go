package repository

import (
	"database/sql"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

// GetPendingMessages gets pending messages for a device with campaign/sequence delays
func (r *BroadcastRepository) GetPendingMessages(deviceID string, limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT 
			bm.id, bm.user_id, bm.device_id, bm.campaign_id, bm.sequence_id, 
			bm.recipient_phone, bm.message_type, bm.content as message, bm.media_url as image_url, 
			bm.scheduled_at, bm.group_id, bm.group_order,
			COALESCE(c.min_delay_seconds, s.min_delay_seconds, 10) as min_delay,
			COALESCE(c.max_delay_seconds, s.max_delay_seconds, 30) as max_delay
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN sequences s ON bm.sequence_id = s.id
		WHERE bm.device_id = $1 AND bm.status = 'pending'
		AND (bm.scheduled_at IS NULL OR bm.scheduled_at <= $2)
		ORDER BY bm.group_id NULLS LAST, bm.group_order NULLS LAST, bm.created_at ASC
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
			&groupID, &groupOrder, &msg.MinDelay, &msg.MaxDelay)
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
		SET status = $1, 
		    error_message = $2, 
		    sent_at = CASE WHEN $3 = 'sent' THEN NOW() ELSE sent_at END,
		    updated_at = NOW()
		WHERE id = $4
	`
	
	result, err := r.db.Exec(query, status, errorMsg, status, messageID)
	if err != nil {
		logrus.Errorf("Failed to update message status: %v", err)
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.Warnf("No rows updated for message ID: %s", messageID)
	} else {
		logrus.Infof("Updated message %s status to %s", messageID, status)
	}
	
	return nil
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

// GetUserBroadcastStats gets broadcast statistics for a user
func (r *BroadcastRepository) GetUserBroadcastStats(userID string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending
		FROM broadcast_messages
		WHERE user_id = $1
	`
	
	var total, sent, failed, pending int
	err := r.db.QueryRow(query, userID).Scan(&total, &sent, &failed, &pending)
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"total_messages": total,		"sent_messages": sent,
		"failed_messages": failed,
		"pending_messages": pending,
		"success_rate": float64(sent) / float64(max(1, total)) * 100,
	}
	
	return stats, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetAllPendingMessages gets all pending messages across all devices
func (r *BroadcastRepository) GetAllPendingMessages(limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT id, user_id, device_id, campaign_id, sequence_id, recipient_phone, 
		       message_type, content, media_url, status, scheduled_at, created_at,
		       group_id, group_order
		FROM broadcast_messages
		WHERE status = 'pending' 
		AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var messages []domainBroadcast.BroadcastMessage
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var campaignID sql.NullInt64
		var sequenceID sql.NullString
		var scheduledAt sql.NullTime
		var groupID sql.NullString
		var groupOrder sql.NullInt64
		
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.DeviceID, &campaignID, &sequenceID,
			&msg.RecipientPhone, &msg.Type, &msg.Content, &msg.MediaURL, &msg.Status,
			&scheduledAt, &msg.CreatedAt, &groupID, &groupOrder)
		if err != nil {
			continue
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

// GetDevicesWithPendingMessages gets all device IDs that have pending messages
func (r *BroadcastRepository) GetDevicesWithPendingMessages() ([]string, error) {
	query := `
		SELECT DISTINCT device_id 
		FROM broadcast_messages 
		WHERE status = 'pending' 
		AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY device_id
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var devices []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			return nil, err
		}
		devices = append(devices, deviceID)
	}
	
	return devices, rows.Err()
}