package repository

import (
	"database/sql"
	"fmt"
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
	
	// ISSUE 3 FIX: Check for duplicates before inserting
	// For SEQUENCES: Check based on sequence_stepid, recipient_phone, and device_id
	if msg.SequenceStepID != nil && *msg.SequenceStepID != "" {
		duplicateCheck := `
			SELECT COUNT(*) 
			FROM broadcast_messages 
			WHERE sequence_stepid = ? 
			AND recipient_phone = ? 
			AND device_id = ?
			AND status IN ('pending', 'sent', 'queued', 'processing')
		`
		
		var count int
		err := r.db.QueryRow(duplicateCheck, *msg.SequenceStepID, msg.RecipientPhone, msg.DeviceID).Scan(&count)
		if err != nil {
			logrus.Warnf("Error checking sequence duplicates: %v", err)
		} else if count > 0 {
			logrus.Infof("Skipping duplicate sequence message for %s - sequence_step %s already exists", 
				msg.RecipientPhone, *msg.SequenceStepID)
			return nil // Skip duplicate
		}
	}
	
	// For CAMPAIGNS: Check based on campaign_id, recipient_phone, and device_id
	if msg.CampaignID != nil && *msg.CampaignID > 0 {
		duplicateCheck := `
			SELECT COUNT(*) 
			FROM broadcast_messages 
			WHERE campaign_id = ? 
			AND recipient_phone = ? 
			AND device_id = ?
			AND status IN ('pending', 'sent', 'queued', 'processing')
		`
		
		var count int
		err := r.db.QueryRow(duplicateCheck, *msg.CampaignID, msg.RecipientPhone, msg.DeviceID).Scan(&count)
		if err != nil {
			logrus.Warnf("Error checking campaign duplicates: %v", err)
		} else if count > 0 {
			logrus.Infof("Skipping duplicate campaign message for %s - campaign %d already exists", 
				msg.RecipientPhone, *msg.CampaignID)
			return nil // Skip duplicate
		}
	}
	
	query := `
		INSERT INTO broadcast_messages(id, user_id, device_id, device_name, campaign_id, sequence_id, sequence_stepid, recipient_phone, recipient_name,
		 message_type, content, media_url, status, scheduled_at, created_at, group_id, group_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Get user_id and device_name from user_devices table
	var userID, deviceName string
	if msg.UserID != "" {
		userID = msg.UserID
		// Still need to fetch device_name
		err := r.db.QueryRow("SELECT device_name FROM user_devices WHERE id = ?", msg.DeviceID).Scan(&deviceName)
		if err != nil {
			// If device_name not found, use device_id as fallback
			// This is for platform devices (Whacenter/Wablas)
			logrus.Infof("Device %s not found in user_devices, using device_id as device_name (platform device)", msg.DeviceID)
			deviceName = msg.DeviceID
		}
	} else {
		// Try to fetch user_id and device_name from user_devices
		err := r.db.QueryRow("SELECT user_id, device_name FROM user_devices WHERE id = ?", msg.DeviceID).Scan(&userID, &deviceName)
		if err != nil {
			// PLATFORM FIX: If device not found in user_devices, it might be a platform device
			// For platform devices (Whacenter/Wablas), use device_id as device_name
			logrus.Infof("Device %s not found in user_devices, using device_id as device_name (platform device)", msg.DeviceID)
			userID = ""
			deviceName = msg.DeviceID
			// Don't return error - continue with insert
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
	if msg.SequenceID != nil && *msg.SequenceID != "" {
		sequenceID = *msg.SequenceID
	} else {
		sequenceID = nil
	}
	
	var sequenceStepID interface{}
	if msg.SequenceStepID != nil && *msg.SequenceStepID != "" {
		sequenceStepID = *msg.SequenceStepID
	} else {
		sequenceStepID = nil
	}
	
	var groupID interface{}
	if msg.GroupID != nil && *msg.GroupID != "" {
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
	
	_, err := r.db.Exec(query, msg.ID, userID, msg.DeviceID, deviceName, campaignID,
		sequenceID, sequenceStepID, msg.RecipientPhone, msg.RecipientName, msg.Type, msg.Content,
		msg.MediaURL, "pending", msg.ScheduledAt, time.Now(), groupID, groupOrder)

	return err
}

// GetPendingMessages gets pending messages for a device with campaign/sequence delays
func (r *BroadcastRepository) GetPendingMessages(deviceID string, limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT bm.id, bm.user_id, bm.device_id, bm.device_name, bm.campaign_id, bm.sequence_id,
			bm.recipient_phone, bm.recipient_name, bm.message_type, bm.content AS message, bm.media_url,
			bm.scheduled_at, bm.group_id, bm.group_order,
			COALESCE(
				c.min_delay_seconds,
				ss.min_delay_seconds,
				s.min_delay_seconds,
				10
			) AS min_delay,
			COALESCE(
				c.max_delay_seconds,
				ss.max_delay_seconds,
				s.max_delay_seconds,
				30
			) AS max_delay
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN sequences s ON bm.sequence_id = s.id
		LEFT JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
		WHERE bm.device_id = ?
		AND bm.status = 'pending'
		AND bm.scheduled_at IS NOT NULL
		AND bm.scheduled_at <= ?
		AND bm.scheduled_at >= DATE_SUB(?, INTERVAL 3 HOUR)
		ORDER BY bm.scheduled_at ASC, bm.group_id, bm.group_order
		LIMIT ?
	`
	now := time.Now()
	rows, err := r.db.Query(query, deviceID, now, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domainBroadcast.BroadcastMessage
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var userID, deviceName sql.NullString
		var campaignID sql.NullInt64
		var sequenceID, groupID sql.NullString
		var groupOrder sql.NullInt64
		var scheduledAt sql.NullTime

		err := rows.Scan(&msg.ID, &userID, &msg.DeviceID, &deviceName, &campaignID, &sequenceID,
			&msg.RecipientPhone, &msg.RecipientName, &msg.Type, &msg.Content, &msg.MediaURL, &scheduledAt,
			&groupID, &groupOrder, &msg.MinDelay, &msg.MaxDelay)
		if err != nil {
			continue
		}

		if deviceName.Valid {
			msg.DeviceName = deviceName.String
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
		
		// Set ImageURL for backward compatibility
		msg.ImageURL = msg.MediaURL
		msg.Message = msg.Content
		
		messages = append(messages, msg)
	}
	
	return messages, nil
}

// UpdateMessageStatus updates message status
func (r *BroadcastRepository) UpdateMessageStatus(messageID, status, errorMsg string) error {
	query := `
		UPDATE broadcast_messages SET status = ?, 
		    error_message = ?, 
		    sent_at = CASE WHEN ? = 'sent' THEN NOW() ELSE sent_at END,
		    updated_at = NOW()
		WHERE id = ?
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
		SELECT status, COUNT(*) AS count
		FROM broadcast_messages
		WHERE device_id = ? AND created_at > DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 24 HOUR)
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
		SELECT COUNT(*) AS total,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) AS sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pending
		FROM broadcast_messages
		WHERE user_id = ?
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
		SELECT id, user_id, device_id, device_name, campaign_id, sequence_id, recipient_phone,
		       recipient_name, message_type, content, media_url, status, scheduled_at,
		       created_at, group_id, group_order
		FROM broadcast_messages
		WHERE status = 'pending'
		AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domainBroadcast.BroadcastMessage
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var deviceName sql.NullString
		var campaignID sql.NullInt64
		var sequenceID sql.NullString
		var recipientName sql.NullString
		var scheduledAt sql.NullTime
		var groupID sql.NullString
		var groupOrder sql.NullInt64

		err := rows.Scan(&msg.ID, &msg.UserID, &msg.DeviceID, &deviceName, &campaignID, &sequenceID,
			&msg.RecipientPhone, &recipientName, &msg.Type, &msg.Content, &msg.MediaURL,
			&msg.Status, &scheduledAt, &msg.CreatedAt, &groupID, &groupOrder)
		if err != nil {
			continue
		}

		// Set device name
		if deviceName.Valid {
			msg.DeviceName = deviceName.String
		}

		// Set recipient name
		if recipientName.Valid {
			msg.RecipientName = recipientName.String
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
		
		// Set ImageURL for backward compatibility
		msg.ImageURL = msg.MediaURL
		msg.Message = msg.Content
		
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
		AND (scheduled_at IS NULL OR scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR))
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

// GetDB returns the database connection
func (r *BroadcastRepository) GetDB() *sql.DB {
	return r.db
}

// GetPendingMessagesAndLock - Atomically fetch and lock messages using UPDATE-then-SELECT for MySQL 5.7
func (r *BroadcastRepository) GetPendingMessagesAndLock(deviceID string, limit int) ([]domainBroadcast.BroadcastMessage, error) {
	// Generate unique worker ID for this operation
	workerID := fmt.Sprintf("%s_%d_%s", deviceID, time.Now().UnixNano(), uuid.New().String()[:8])
	
	// Debug: Check why messages aren't being claimed
	var debugInfo struct {
		TotalPending int
		WithinWindow int
		ServerNow    string
		MalaysiaNow  string
		WindowStart  string
		WindowEnd    string
	}
	
	err := r.db.QueryRow(`
		SELECT 
			COUNT(*) as total_pending,
			SUM(CASE 
				WHEN scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR) 
				AND scheduled_at >= DATE_ADD(DATE_SUB(NOW(), INTERVAL 1 DAY), INTERVAL 8 HOUR)
				THEN 1 ELSE 0 
			END) as within_window,
			NOW() as server_now,
			DATE_ADD(NOW(), INTERVAL 8 HOUR) as malaysia_now,
			DATE_ADD(DATE_SUB(NOW(), INTERVAL 1 DAY), INTERVAL 8 HOUR) as window_start,
			DATE_ADD(NOW(), INTERVAL 8 HOUR) as window_end
		FROM broadcast_messages
		WHERE device_id = ?
		AND status = 'pending'
		AND processing_worker_id IS NULL
		AND scheduled_at IS NOT NULL
	`, deviceID).Scan(&debugInfo.TotalPending, &debugInfo.WithinWindow, 
		&debugInfo.ServerNow, &debugInfo.MalaysiaNow, 
		&debugInfo.WindowStart, &debugInfo.WindowEnd)
	
	if err == nil && debugInfo.TotalPending > 0 && debugInfo.WithinWindow == 0 {
		logrus.Warnf("🕐 Device %s time window mismatch: %d pending but 0 within window. Window: %s to %s", 
			deviceID, debugInfo.TotalPending, debugInfo.WindowStart, debugInfo.WindowEnd)
	}
	
	// STEP 1: Atomically claim messages by updating their status (MySQL 5.7 compatible)
	result, err := r.db.Exec(`
		UPDATE broadcast_messages 
		SET status = 'processing',
			processing_worker_id = ?,
			processing_started_at = DATE_ADD(NOW(), INTERVAL 8 HOUR),
			updated_at = NOW()
		WHERE device_id = ? 
		AND status = 'pending'
		AND processing_worker_id IS NULL
		AND scheduled_at IS NOT NULL
		AND scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR)
		ORDER BY scheduled_at ASC, group_id, group_order
		LIMIT ?
	`, workerID, deviceID, limit)
	
	if err != nil {
		return nil, fmt.Errorf("failed to claim messages: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// No messages to process
		return []domainBroadcast.BroadcastMessage{}, nil
	}
	
	logrus.Infof("Worker %s claimed %d messages for device %s", workerID, rowsAffected, deviceID)
	
	// STEP 2: Fetch the messages we just claimed
	query := `
		SELECT bm.id, bm.user_id, bm.device_id, bm.device_name, bm.campaign_id, bm.sequence_id,
			bm.recipient_phone, bm.recipient_name, bm.message_type, bm.content AS message, bm.media_url,
			bm.scheduled_at, bm.group_id, bm.group_order, bm.sequence_stepid,
			COALESCE(
				c.min_delay_seconds,
				ss.min_delay_seconds,
				s.min_delay_seconds,
				10
			) AS min_delay,
			COALESCE(
				c.max_delay_seconds,
				ss.max_delay_seconds,
				s.max_delay_seconds,
				30
			) AS max_delay
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN sequences s ON bm.sequence_id = s.id
		LEFT JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
		WHERE bm.processing_worker_id = ?
		ORDER BY bm.scheduled_at ASC, bm.group_id, bm.group_order
	`

	rows, err := r.db.Query(query, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domainBroadcast.BroadcastMessage

	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var userID, deviceName sql.NullString
		var campaignID sql.NullInt64
		var sequenceID, groupID, sequenceStepID sql.NullString
		var groupOrder sql.NullInt64
		var scheduledAt sql.NullTime

		err := rows.Scan(&msg.ID, &userID, &msg.DeviceID, &deviceName, &campaignID, &sequenceID,
			&msg.RecipientPhone, &msg.RecipientName, &msg.Type, &msg.Content, &msg.MediaURL, &scheduledAt,
			&groupID, &groupOrder, &sequenceStepID, &msg.MinDelay, &msg.MaxDelay)
		if err != nil {
			continue
		}

		if deviceName.Valid {
			msg.DeviceName = deviceName.String
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
		if sequenceStepID.Valid {
			msg.SequenceStepID = &sequenceStepID.String
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
		
		// Set ImageURL for backward compatibility
		msg.ImageURL = msg.MediaURL
		msg.Message = msg.Content
		
		messages = append(messages, msg)
	}
	
	return messages, nil
}
