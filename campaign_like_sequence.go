// processCampaignDirect enrolls campaign leads directly to broadcast_messages
func (p *DirectBroadcastProcessor) processCampaignDirect(campaign *models.Campaign) (int, error) {
	// Get broadcast repository
	broadcastRepo := repository.GetBroadcastRepository()
	
	// Find matching leads - without device status check
	query := `
		SELECT DISTINCT 
			l.id, l.phone, l.name, l.device_id, l.user_id
		FROM leads l
		INNER JOIN user_devices ud ON l.device_id = ud.id
		WHERE ud.user_id = ?
		AND l.niche LIKE CONCAT('%', ?, '%')
		AND (? = 'all' OR l.target_status = ?)
		AND NOT EXISTS (
			SELECT 1 FROM broadcast_messages bm
			WHERE bm.campaign_id = ?
			AND bm.recipient_phone = l.phone
			AND bm.status IN ('pending', 'processing', 'queued', 'sent')
		)
		LIMIT 1000
	`
	
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "all"
	}
	
	rows, err := p.db.Query(query, campaign.UserID, campaign.Niche, targetStatus, targetStatus, campaign.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to query leads: %w", err)
	}
	defer rows.Close()
	
	enrolledCount := 0
	
	// EXACTLY LIKE SEQUENCES: Schedule first message 5 minutes from now
	scheduledAt := time.Now().Add(5 * time.Minute)
	
	for rows.Next() {
		var leadID, phone, name, deviceID, userID string
		if err := rows.Scan(&leadID, &phone, &name, &deviceID, &userID); err != nil {
			continue
		}
		
		// Create broadcast message EXACTLY LIKE SEQUENCES
		msg := domainBroadcast.BroadcastMessage{
			UserID:         userID,
			DeviceID:       deviceID,
			CampaignID:     &campaign.ID,
			RecipientPhone: phone,
			RecipientName:  name,
			Type:           "text",
			Message:        campaign.Message,
			Content:        campaign.Message,
			MediaURL:       campaign.ImageURL,
			MinDelay:       campaign.MinDelaySeconds,
			MaxDelay:       campaign.MaxDelaySeconds,
			ScheduledAt:    scheduledAt,  // CHANGED: Use same scheduling as sequences
			Status:         "pending",     // ADDED: Explicitly set status like sequences
		}
		
		// Handle image URL like sequences
		if campaign.ImageURL != "" {
			msg.MediaURL = campaign.ImageURL
			msg.ImageURL = campaign.ImageURL
			msg.Type = "image"  // ADDED: Set type to image if there's media
		}
		
		if err := broadcastRepo.QueueMessage(msg); err != nil {
			logrus.Debugf("Failed to queue message for %s: %v", phone, err)
		} else {
			enrolledCount++
		}
	}
	
	return enrolledCount, nil
}