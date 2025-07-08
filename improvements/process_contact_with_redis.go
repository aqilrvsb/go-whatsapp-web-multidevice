// Updated processContactFlow with Redis-based device reservation
func (s *OptimizedSequenceTriggerProcessor) processContactFlow(job contactFlowJob, deviceLoads map[string]DeviceLoad) bool {
	ctx := context.Background()
	
	// Use Redis-based atomic device reservation
	deviceID, releaseFunc, err := s.deviceManager.ReserveDeviceAtomic(ctx, job.preferredDevice.String)
	if err != nil {
		logrus.Warnf("No available device for contact %s: %v", job.phone, err)
		s.markContactFailed(job.sequenceContactID, "No available device")
		return false
	}
	
	// IMPORTANT: Always release the device when done
	defer releaseFunc()

	// Claim the contact for processing
	claimQuery := `
		UPDATE sequence_contacts 
		SET processing_device_id = $1, processing_started_at = $2
		WHERE id = $3 AND processing_device_id IS NULL
	`
	
	result, err := s.db.Exec(claimQuery, deviceID, time.Now(), job.sequenceContactID)
	if err != nil {
		logrus.Errorf("Failed to claim contact %s: %v", job.sequenceContactID, err)
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Already claimed by another worker
		return false
	}

	// Get min/max delay settings for the step
	var minDelay, maxDelay int
	delayQuery := `
		SELECT 
			COALESCE(ss.min_delay_seconds, s.min_delay_seconds, 10) as min_delay,
			COALESCE(ss.max_delay_seconds, s.max_delay_seconds, 30) as max_delay
		FROM sequence_steps ss
		JOIN sequences s ON s.id = ss.sequence_id
		WHERE ss.id = $1
	`
	
	err = s.db.QueryRow(delayQuery, job.sequenceStepID).Scan(&minDelay, &maxDelay)
	if err != nil {
		// Use defaults if query fails
		minDelay = 10
		maxDelay = 30
	}

	// Calculate random delay between min and max
	var delay time.Duration
	if minDelay >= maxDelay {
		delay = time.Duration(minDelay) * time.Second
	} else {
		// Random delay between min and max
		delayRange := maxDelay - minDelay
		randomDelay := rand.Intn(delayRange) + minDelay
		delay = time.Duration(randomDelay) * time.Second
	}

	// Apply the delay before sending
	logrus.Debugf("Applying %v delay before sending message to %s on device %s", delay, job.phone, deviceID)
	time.Sleep(delay)

	// Create broadcast message
	broadcastMsg := domainBroadcast.BroadcastMessage{
		DeviceID:       deviceID,
		RecipientPhone: job.phone,
		Message:        job.messageText,
		Content:        job.messageText,
		Type:           job.messageType,
		SequenceStepID: job.sequenceStepID,
	}

	if job.mediaURL.Valid && job.mediaURL.String != "" {
		broadcastMsg.MediaURL = job.mediaURL.String
	}

	// Send to broadcast manager
	if err := s.broadcastMgr.SendMessage(broadcastMsg); err != nil {
		logrus.Errorf("Failed to queue message for %s: %v", job.phone, err)
		s.releaseContact(job.sequenceContactID)
		return false
	}

	// No need to update device load - Redis already tracking it!
	
	// Update contact flow record as sent
	if err := s.updateContactFlowProgress(job.sequenceContactID, job.sequenceStepID, deviceID); err != nil {
		logrus.Errorf("Failed to update contact flow progress: %v", err)
		return false
	}

	// Get device stats for logging
	stats, _ := s.deviceManager.GetDeviceStats(ctx, deviceID)
	logrus.Infof("Sent message to %s via device %s (hour: %d/80, day: %d/800)", 
		job.phone, deviceID, stats.MessagesHour, stats.MessagesToday)

	// Schedule next flow if exists
	if job.nextTrigger.Valid && job.nextTrigger.String != "" {
		if err := s.scheduleNextFlow(job.sequenceID, job.phone, job.nextTrigger.String, job.delayHours); err != nil {
			logrus.Warnf("Failed to schedule next flow: %v", err)
		}
	} else {
		// Sequence complete - handle completion
		s.handleSequenceCompletion(job.sequenceID, job.phone, job.currentTrigger)
	}

	return true
}