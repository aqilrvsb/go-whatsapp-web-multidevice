// ProcessCampaigns checks and processes ready campaigns (NEW FUNCTION)
func (p *DirectBroadcastProcessor) ProcessCampaigns() (int, error) {
	// Query campaigns that are ready to send - FIXED FOR TIMEZONE
	query := `
		SELECT c.id, c.user_id, c.title, c.message, c.niche, 
			COALESCE(c.target_status, 'all') AS target_status, 
			COALESCE(c.image_url, '') AS image_url, 
			c.min_delay_seconds, c.max_delay_seconds
		FROM campaigns c
		WHERE c.status = 'pending'
		AND (
			(c.scheduled_at IS NOT NULL AND c.scheduled_at <= NOW())
			OR
			(c.scheduled_at IS NULL AND 
			 STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= DATE_ADD(NOW(), INTERVAL 8 HOUR))
		)
		LIMIT 10
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query campaigns: %w", err)
	}
	defer rows.Close()

	processedCount := 0
	for rows.Next() {
		var campaign models.Campaign
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.TargetStatus, &campaign.ImageURL,
			&campaign.MinDelaySeconds, &campaign.MaxDelaySeconds,
		)
		if err != nil {
			logrus.Errorf("Failed to scan campaign: %v", err)
			continue
		}

		logrus.Infof("Processing campaign: %s (ID: %d)", campaign.Title, campaign.ID)
		
		// Process campaign using same direct enrollment approach
		enrolledCount, err := p.processCampaignDirect(&campaign)
		if err != nil {
			logrus.Errorf("Failed to process campaign %s: %v", campaign.Title, err)
			continue
		}

		// Update campaign status
		if enrolledCount > 0 {
			p.db.Exec("UPDATE campaigns SET status = 'triggered', updated_at = NOW() WHERE id = ?", campaign.ID)
			logrus.Infof("Campaign %s triggered: %d messages queued", campaign.Title, enrolledCount)
		} else {
			p.db.Exec("UPDATE campaigns SET status = 'finished', updated_at = NOW() WHERE id = ?", campaign.ID)
			logrus.Infof("Campaign %s finished: No matching leads found", campaign.Title)
		}

		processedCount++
	}

	return processedCount, nil
}