// FIX FOR SEQUENCE MESSAGE GREETING ISSUE
// The problem: GetPendingMessages was not populating the Message field correctly
// This caused the greeting processor to not work for sequence messages

// In src/repository/broadcast_repository.go, update the GetPendingMessages function:

// GetPendingMessages gets pending messages for a device with campaign/sequence delays
func (r *BroadcastRepository) GetPendingMessages(deviceID string, limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT 
			bm.id, bm.user_id, bm.device_id, bm.campaign_id, bm.sequence_id, 
			bm.recipient_phone, bm.recipient_name, bm.message_type, bm.content, bm.media_url, 
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
			&msg.RecipientPhone, &msg.RecipientName, &msg.Type, &msg.Content, &msg.MediaURL, &scheduledAt,
			&groupID, &groupOrder, &msg.MinDelay, &msg.MaxDelay)
		if err != nil {
			continue
		}
		
		// CRITICAL FIX: Ensure Message field is populated for the greeting processor
		msg.Message = msg.Content
		msg.ImageURL = msg.MediaURL // Also ensure ImageURL alias is set
		
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
