package usecase

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// contactJob represents a job for processing a contact message
type contactJob struct {
	contactID        string
	sequenceID       string
	phone            string
	name             string
	currentTrigger   string
	currentStep      int
	messageText      string
	messageType      string
	mediaURL         sql.NullString
	nextTrigger      sql.NullString
	delayHours       int
	preferredDevice  sql.NullString
	minDelaySeconds  int
	maxDelaySeconds  int
	userID           string  // Added to track user
}

// SequenceTriggerProcessor handles trigger-based sequence processing
type SequenceTriggerProcessor struct {
	db              *sql.DB
	broadcastMgr    broadcast.BroadcastManagerInterface
	isRunning       bool
	stopChan        chan bool
	ticker          *time.Ticker
	mutex           sync.Mutex
	batchSize       int
	processInterval time.Duration
}

// NewSequenceTriggerProcessor creates a new trigger processor
func NewSequenceTriggerProcessor(db *sql.DB) *SequenceTriggerProcessor {
	return &SequenceTriggerProcessor{
		db:              db,
		broadcastMgr:    broadcast.GetBroadcastManager(),
		stopChan:        make(chan bool),
		batchSize:       5000,  // Increased for 3000 devices
		processInterval: 15 * time.Second,  // Reduced from 30s for faster processing
	}
}

// Start begins the trigger processing
func (s *SequenceTriggerProcessor) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("sequence trigger processor already running")
	}

	s.ticker = time.NewTicker(s.processInterval)
	s.isRunning = true

	// Also monitor broadcast message results
	go s.monitorBroadcastResults()
	
	go s.run()
	
	logrus.Info("Sequence trigger processor started")
	return nil
}

// Stop halts the trigger processing
func (s *SequenceTriggerProcessor) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return
	}

	s.ticker.Stop()
	s.stopChan <- true
	s.isRunning = false

	logrus.Info("Sequence trigger processor stopped")
}

// run is the main processing loop
func (s *SequenceTriggerProcessor) run() {
	// Process immediately on start
	s.processTriggers()

	for {
		select {
		case <-s.ticker.C:
			s.processTriggers()
		case <-s.stopChan:
			return
		}
	}
}

// processTriggers handles the main trigger processing logic
func (s *SequenceTriggerProcessor) processTriggers() {
	startTime := time.Now()
	logrus.Debug("Starting trigger processing...")

	// Step 1: Process leads with triggers to enroll in sequences
	enrolledCount, err := s.enrollLeadsFromTriggers()
	if err != nil {
		logrus.Errorf("Error enrolling leads: %v", err)
	}

	// Step 2: Clean up stuck processing
	if err := s.cleanupStuckProcessing(); err != nil {
		logrus.Warnf("Error cleaning up stuck processing: %v", err)
	}

	// Step 3: Get device workload for load balancing
	deviceLoads, err := s.getDeviceWorkloads()
	if err != nil {
		logrus.Errorf("Error getting device workloads: %v", err)
		return
	}

	// Step 4: Process sequence contacts in parallel
	processedCount, err := s.processSequenceContacts(deviceLoads)
	if err != nil {
		logrus.Errorf("Error processing sequence contacts: %v", err)
	}

	duration := time.Since(startTime)
	
	// Calculate metrics for monitoring
	totalDevices := len(deviceLoads)
	activeDevices := 0
	for _, load := range deviceLoads {
		if load.IsAvailable {
			activeDevices++
		}
	}
	
	logrus.Infof("Sequence processing completed: enrolled=%d, processed=%d, devices=%d/%d, duration=%v", 
		enrolledCount, processedCount, activeDevices, totalDevices, duration)
	
	// Log performance metrics
	if processedCount > 0 {
		avgTimePerMessage := duration / time.Duration(processedCount)
		messagesPerMinute := float64(processedCount) / duration.Minutes()
		logrus.Infof("Performance: %.2f msg/min, %v avg/msg", messagesPerMinute, avgTimePerMessage)
	}
}

// enrollLeadsFromTriggers checks leads for matching sequence triggers
func (s *SequenceTriggerProcessor) enrollLeadsFromTriggers() (int, error) {
	// Simplified query without CTEs to avoid column reference issues
	query := `
		SELECT DISTINCT 
			l.id, l.phone, l.name, l.device_id, l.user_id, 
			s.id as sequence_id, ss.trigger as entry_trigger
		FROM leads l
		CROSS JOIN sequences s
		INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
		WHERE s.is_active = true 
			AND ss.is_entry_point = true
			AND l.trigger IS NOT NULL 
			AND l.trigger != ''
			AND position(ss.trigger in l.trigger) > 0
			AND NOT EXISTS (
				SELECT 1 FROM sequence_contacts sc
				WHERE sc.sequence_id = s.id 
				AND sc.contact_phone = l.phone
				AND sc.current_step = 1
			)
		LIMIT 5000
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query leads for enrollment: %w", err)
	}
	defer rows.Close()

	enrolledCount := 0
	for rows.Next() {
		var lead models.Lead
		var sequenceID, entryTrigger string

		if err := rows.Scan(&lead.ID, &lead.Phone, &lead.Name, &lead.DeviceID, &lead.UserID,
			&sequenceID, &entryTrigger); err != nil {
			logrus.Warnf("Error scanning lead: %v", err)
			continue
		}

		// Enroll in sequence
		if err := s.enrollContactInSequence(sequenceID, lead, entryTrigger); err != nil {
			logrus.Warnf("Error enrolling contact %s: %v", lead.Phone, err)
			continue
		}

		enrolledCount++
	}

	return enrolledCount, nil
}

// enrollContactInSequence adds a contact to a sequence - ONLY creates the first step
func (s *SequenceTriggerProcessor) enrollContactInSequence(sequenceID string, lead models.Lead, trigger string) error {
	// Get only the first step (entry point)
	stepQuery := `
		SELECT id, day_number, trigger, next_trigger, trigger_delay_hours 
		FROM sequence_steps 
		WHERE sequence_id = $1 
		AND is_entry_point = true
		ORDER BY day_number ASC
		LIMIT 1
	`
	
	var step struct {
		ID              string
		DayNumber       int
		Trigger         string
		NextTrigger     sql.NullString
		TriggerDelayHours int
	}
	
	err := s.db.QueryRow(stepQuery, sequenceID).Scan(
		&step.ID, &step.DayNumber, &step.Trigger, 
		&step.NextTrigger, &step.TriggerDelayHours)
	
	if err != nil {
		return fmt.Errorf("failed to get entry step: %w", err)
	}
	
	// Create only the first step record
	insertQuery := `
		INSERT INTO sequence_contacts (
			sequence_id, contact_phone, contact_name, 
			current_step, status, current_trigger,
			next_trigger_time, sequence_stepid, assigned_device_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (sequence_id, contact_phone, sequence_stepid) DO NOTHING
	`
	
	currentTime := time.Now()
	
	logrus.Infof("Enrolling contact %s in sequence %s - creating ONLY step 1", lead.Phone, sequenceID)
	
	_, err = s.db.Exec(insertQuery, 
		sequenceID,          // sequence_id
		lead.Phone,          // contact_phone
		lead.Name,           // contact_name
		step.DayNumber,      // current_step - use actual day_number from sequence_steps
		"active",            // status - first step is active
		step.Trigger,        // current_trigger
		currentTime,         // next_trigger_time - process immediately
		step.ID,             // sequence_stepid
		lead.DeviceID,       // assigned_device_id
	)
	
	if err != nil {
		logrus.Warnf("Failed to enroll contact %s: %v", lead.Phone, err)
		return err
	}
	
	logrus.Infof("Successfully enrolled %s with step 1 only", lead.Phone)
	return nil
}

// getDeviceWorkloads retrieves current device loads for balancing
func (s *SequenceTriggerProcessor) getDeviceWorkloads() (map[string]DeviceLoad, error) {
	query := `
		SELECT 
			d.id,
			d.status,
			COALESCE(dlb.messages_hour, 0) as messages_hour,
			COALESCE(dlb.messages_today, 0) as messages_today,
			COALESCE(dlb.is_available, true) as is_available,
			COUNT(sc.id) as current_processing
		FROM user_devices d
		LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
		LEFT JOIN sequence_contacts sc ON sc.processing_device_id = d.id 
			AND sc.processing_started_at > NOW() - INTERVAL '5 minutes'
		WHERE d.status = 'online' OR d.platform IS NOT NULL AND d.platform != ''
		GROUP BY d.id, d.status, dlb.messages_hour, dlb.messages_today, dlb.is_available
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get device workloads: %w", err)
	}
	defer rows.Close()

	loads := make(map[string]DeviceLoad)
	for rows.Next() {
		var load DeviceLoad
		if err := rows.Scan(&load.DeviceID, &load.Status, &load.MessagesHour,
			&load.MessagesToday, &load.IsAvailable, &load.CurrentProcessing); err != nil {
			continue
		}
		loads[load.DeviceID] = load
	}

	return loads, nil
}

// processSequenceContacts processes contacts ready for their next message
func (s *SequenceTriggerProcessor) processSequenceContacts(deviceLoads map[string]DeviceLoad) (int, error) {
	// Debug: Log how many active contacts we're about to process
	var activeCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM sequence_contacts WHERE status = 'active' AND next_trigger_time <= $1`, time.Now()).Scan(&activeCount)
	logrus.Infof("Found %d active sequence contacts ready for processing", activeCount)
	
	// CRITICAL: Add extra logging to debug the issue
	var debugInfo []string
	debugRows, _ := s.db.Query(`
		SELECT sc.contact_phone, sc.current_step, sc.status, sc.next_trigger_time, ss.day_number 
		FROM sequence_contacts sc 
		JOIN sequence_steps ss ON ss.id = sc.sequence_stepid 
		WHERE sc.status = 'active' 
		ORDER BY sc.contact_phone, ss.day_number 
		LIMIT 10
	`)
	defer debugRows.Close()
	for debugRows.Next() {
		var phone string
		var step int
		var status string
		var nextTime time.Time
		var dayNum int
		debugRows.Scan(&phone, &step, &status, &nextTime, &dayNum)
		debugInfo = append(debugInfo, fmt.Sprintf("%s: step=%d, day=%d, status=%s, next=%v", 
			phone, step, dayNum, status, nextTime.Format("15:04:05")))
	}
	if len(debugInfo) > 0 {
		logrus.Infof("Active contacts sample: %v", debugInfo)
	}
	
	// Get contacts ready for processing
	query := `
		SELECT 
			sc.id, sc.sequence_id, sc.contact_phone, sc.contact_name,
			sc.current_trigger, sc.current_step,
			ss.content, ss.message_type, ss.media_url,
			ss.next_trigger, ss.trigger_delay_hours,
			COALESCE(sc.assigned_device_id, l.device_id) as preferred_device_id,
			COALESCE(ss.min_delay_seconds, 5) as min_delay_seconds,
			COALESCE(ss.max_delay_seconds, 15) as max_delay_seconds,
			l.user_id
		FROM sequence_contacts sc
		JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
		JOIN sequences s ON s.id = sc.sequence_id
		LEFT JOIN leads l ON l.phone = sc.contact_phone
		WHERE sc.status = 'active'
			AND s.is_active = true
			AND sc.next_trigger_time <= $1
			AND sc.processing_device_id IS NULL
		ORDER BY sc.next_trigger_time ASC
		LIMIT $2
	`
	
	// CRITICAL: Log the query parameters
	currentTimeStr := time.Now().Format("2006-01-02 15:04:05")
	logrus.Infof("Processing sequence contacts: status='active', time<='%s', limit=%d", 
		currentTimeStr, s.batchSize)
	
	// SAFETY CHECK: Log any suspicious active records
	var suspiciousCount int
	s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM sequence_contacts 
		WHERE status = 'active' 
		AND next_trigger_time > $1
	`, time.Now()).Scan(&suspiciousCount)
	
	if suspiciousCount > 0 {
		logrus.Warnf("WARNING: Found %d active records with future trigger times - these should NOT be active yet!", suspiciousCount)
	}

	rows, err := s.db.Query(query, time.Now(), s.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get contacts for processing: %w", err)
	}
	defer rows.Close()

	// Process in parallel with worker pool
	jobs := make(chan contactJob, s.batchSize)
	results := make(chan bool, s.batchSize)
	
	// Start workers - increase for 3000 devices
	numWorkers := 50  // Increased from 10
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				success := s.processContact(job, deviceLoads)
				results <- success
			}
		}()
	}

	// Queue jobs
	go func() {
		for rows.Next() {
			var job contactJob
			if err := rows.Scan(&job.contactID, &job.sequenceID, &job.phone, &job.name,
				&job.currentTrigger, &job.currentStep, &job.messageText, &job.messageType,
				&job.mediaURL, &job.nextTrigger, &job.delayHours, &job.preferredDevice,
				&job.minDelaySeconds, &job.maxDelaySeconds, &job.userID); err != nil {
				continue
			}
			jobs <- job
		}
		close(jobs)
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Count successes
	processedCount := 0
	for success := range results {
		if success {
			processedCount++
		}
	}

	return processedCount, nil
}

// processContact handles a single contact's message
func (s *SequenceTriggerProcessor) processContact(job contactJob, deviceLoads map[string]DeviceLoad) bool {
	// CRITICAL: Double-check that this contact is really ready to process
	var nextTriggerTime time.Time
	var status string
	err := s.db.QueryRow(`
		SELECT next_trigger_time, status 
		FROM sequence_contacts 
		WHERE id = $1
	`, job.contactID).Scan(&nextTriggerTime, &status)
	
	if err != nil {
		logrus.Errorf("Failed to verify contact %s: %v", job.contactID, err)
		return false
	}
	
	// CRITICAL: Ensure the record is still active
	if status != "active" {
		logrus.Warnf("Contact %s is not active anymore (status=%s), skipping", job.contactID, status)
		return false
	}
	
	// CRITICAL: Ensure we're not processing too early
	if nextTriggerTime.After(time.Now()) {
		logrus.Warnf("Contact %s not ready yet - next trigger time is %v (in %v)", 
			job.phone, nextTriggerTime, time.Until(nextTriggerTime))
		return false
	}
	
	// Select best device for this contact
	deviceID := s.selectDeviceForContact(job.preferredDevice.String, deviceLoads)
	if deviceID == "" {
		logrus.Warnf("No available device for contact %s", job.phone)
		return false
	}

	// Claim the contact for processing
	claimQuery := `
		UPDATE sequence_contacts 
		SET processing_device_id = $1, processing_started_at = $2
		WHERE id = $3 AND processing_device_id IS NULL
	`
	
	result, err := s.db.Exec(claimQuery, deviceID, time.Now(), job.contactID)
	if err != nil {
		logrus.Errorf("Failed to claim contact %s: %v", job.contactID, err)
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Already claimed by another worker
		return false
	}

	// Queue message to broadcast system
	// Create broadcast message with all required fields like campaigns
	broadcastMsg := domainBroadcast.BroadcastMessage{
		UserID:         job.userID,           // Added - required for tracking
		DeviceID:       deviceID,
		SequenceID:     &job.sequenceID,      // Added - link to sequence
		RecipientPhone: job.phone,
		RecipientName:  job.name,
		Message:        job.messageText,
		Content:        job.messageText,
		Type:           job.messageType,
		MinDelay:       job.minDelaySeconds,
		MaxDelay:       job.maxDelaySeconds,
		ScheduledAt:    time.Now(),           // Added - when queued
		Status:         "pending",            // Added - initial status
	}

	if job.mediaURL.Valid && job.mediaURL.String != "" {
		broadcastMsg.MediaURL = job.mediaURL.String
		broadcastMsg.ImageURL = job.mediaURL.String
	}

	// Queue to database like campaigns (NOT direct to manager)
	broadcastRepo := repository.GetBroadcastRepository()
	if err := broadcastRepo.QueueMessage(broadcastMsg); err != nil {
		logrus.Errorf("Failed to queue sequence message for %s: %v", job.phone, err)
		s.releaseContact(job.contactID)
		return false
	}
	
	logrus.Debugf("Queued sequence message for %s to database", job.phone)

	// Update contact with next trigger
	if err := s.updateContactProgress(job.contactID, job.nextTrigger, job.delayHours); err != nil {
		logrus.Errorf("Failed to update contact progress: %v", err)
		return false
	}

	// Handle sequence completion or continuation
	if !job.nextTrigger.Valid || job.nextTrigger.String == "" {
		// Sequence is complete - remove current trigger from lead
		s.removeCompletedTriggerFromLead(job.phone, job.currentTrigger)
		logrus.Infof("Sequence completed for lead %s, removed trigger: %s", job.phone, job.currentTrigger)
	} else {
		// Check if next trigger points to another sequence
		if !strings.Contains(job.nextTrigger.String, "_day") {
			// This looks like a sequence trigger, not a day trigger
			// Update lead's trigger to the new sequence trigger
			s.updateLeadTrigger(job.phone, job.currentTrigger, job.nextTrigger.String)
			logrus.Infof("Lead %s transitioning from %s to new sequence trigger: %s", 
				job.phone, job.currentTrigger, job.nextTrigger.String)
		}
	}

	logrus.Debugf("Processed contact %s with trigger %s", job.phone, job.currentTrigger)
	return true
}

// selectDeviceForContact chooses the best device for sending
func (s *SequenceTriggerProcessor) selectDeviceForContact(preferredDeviceID string, loads map[string]DeviceLoad) string {
	// STRICT DEVICE MATCHING - Like campaigns, only use the device that owns the lead
	if preferredDeviceID != "" {
		if load, ok := loads[preferredDeviceID]; ok && load.CanAcceptMore() {
			return preferredDeviceID
		}
		// Device not available - do NOT fall back to other devices
		logrus.Debugf("Preferred device %s not available for processing", preferredDeviceID)
		return ""
	}

	// No preferred device - this shouldn't happen if leads are properly assigned
	logrus.Warnf("No preferred device for contact - skipping")
	return ""
}

// updateContactProgress updates the current record as completed and activates the next step
func (s *SequenceTriggerProcessor) updateContactProgress(contactID string, nextTrigger sql.NullString, delayHours int) error {
	// First, mark current contact record as completed
	query := `
		UPDATE sequence_contacts 
		SET status = 'completed', 
			processing_device_id = NULL,
			processing_started_at = NULL,
			completed_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(query, contactID)
	if err != nil {
		return fmt.Errorf("failed to mark contact as completed: %w", err)
	}
	
	// If there's a next trigger, create the next step record
	if nextTrigger.Valid && nextTrigger.String != "" {
		// Get the contact info from current record
		var phone, sequenceID string
		err := s.db.QueryRow(`
			SELECT contact_phone, sequence_id 
			FROM sequence_contacts 
			WHERE id = $1
		`, contactID).Scan(&phone, &sequenceID)
		
		if err != nil {
			return fmt.Errorf("failed to get contact info: %w", err)
		}
		
		// Find the next step details from sequence_steps
		var nextStep struct {
			ID              string
			DayNumber       int
			Trigger         string
			NextTrigger     sql.NullString
			TriggerDelayHours int
		}
		err = s.db.QueryRow(`
			SELECT id, day_number, trigger, next_trigger, trigger_delay_hours
			FROM sequence_steps
			WHERE sequence_id = $1 AND trigger = $2
			LIMIT 1
		`, sequenceID, nextTrigger.String).Scan(
			&nextStep.ID, &nextStep.DayNumber, &nextStep.Trigger,
			&nextStep.NextTrigger, &nextStep.TriggerDelayHours)
		
		if err != nil {
			logrus.Warnf("Next step not found for trigger %s: %v", nextTrigger.String, err)
			return nil
		}
		
		// Calculate next trigger time based on NOW + delay hours
		nextTime := time.Now().Add(time.Duration(nextStep.TriggerDelayHours) * time.Hour)
		
		logrus.Infof("Creating next step: phone=%s, step=%d, trigger=%s, will process at %v", 
			phone, nextStep.DayNumber, nextStep.Trigger, nextTime)
		
		// Create the next step record
		insertQuery := `
			INSERT INTO sequence_contacts (
				sequence_id, contact_phone, contact_name, 
				current_step, status, current_trigger,
				next_trigger_time, sequence_stepid, assigned_device_id, user_id
			) 
			SELECT 
				sequence_id, contact_phone, contact_name,
				$1, 'active', $2, $3, $4, assigned_device_id, user_id
			FROM sequence_contacts
			WHERE id = $5
			ON CONFLICT (sequence_id, contact_phone, sequence_stepid) DO NOTHING
		`
		
		_, err = s.db.Exec(insertQuery, 
			nextStep.DayNumber,  // current_step - use actual day_number
			nextStep.Trigger,    // current_trigger
			nextTime,            // next_trigger_time
			nextStep.ID,         // sequence_stepid
			contactID,           // source record id
		)
		
		if err != nil {
			return fmt.Errorf("failed to create next step: %w", err)
		}
		
		logrus.Infof("Successfully created next step %d for %s", nextStep.DayNumber, phone)
		
	} else {
		// No next trigger - sequence is complete
		logrus.Infof("Sequence completed for contact %s - no more steps", contactID)
		
		// Get contact info for trigger removal
		var phone, currentTrigger string
		err := s.db.QueryRow(`
			SELECT contact_phone, current_trigger 
			FROM sequence_contacts 
			WHERE id = $1
		`, contactID).Scan(&phone, &currentTrigger)
		
		if err == nil {
			// Remove the completed trigger from lead
			s.removeCompletedTriggerFromLead(phone, currentTrigger)
			logrus.Infof("Removed trigger %s from lead %s", currentTrigger, phone)
		}
	}
	
	return nil
}

// releaseContact releases a contact from processing
func (s *SequenceTriggerProcessor) releaseContact(contactID string) {
	query := `
		UPDATE sequence_contacts 
		SET processing_device_id = NULL,
			processing_started_at = NULL,
			retry_count = retry_count + 1
		WHERE id = $1
	`
	s.db.Exec(query, contactID)
}

// removeCompletedTriggerFromLead removes trigger from lead when sequence completes
func (s *SequenceTriggerProcessor) removeCompletedTriggerFromLead(phone, trigger string) {
	// Get current triggers
	var currentTriggers sql.NullString
	err := s.db.QueryRow("SELECT trigger FROM leads WHERE phone = $1", phone).Scan(&currentTriggers)
	if err != nil || !currentTriggers.Valid {
		return
	}

	// Remove the trigger
	triggers := strings.Split(currentTriggers.String, ",")
	newTriggers := []string{}
	for _, t := range triggers {
		t = strings.TrimSpace(t)
		if t != trigger && t != "" {
			newTriggers = append(newTriggers, t)
		}
	}

	// Update lead
	newTriggerStr := ""
	if len(newTriggers) > 0 {
		newTriggerStr = strings.Join(newTriggers, ",")
	}

	s.db.Exec("UPDATE leads SET trigger = NULLIF($1, '') WHERE phone = $2", newTriggerStr, phone)
}

// updateLeadTrigger updates lead trigger from old to new (for sequence chaining)
func (s *SequenceTriggerProcessor) updateLeadTrigger(phone, oldTrigger, newTrigger string) {
	// Get current triggers
	var currentTriggers sql.NullString
	err := s.db.QueryRow("SELECT trigger FROM leads WHERE phone = $1", phone).Scan(&currentTriggers)
	if err != nil {
		return
	}

	var newTriggerStr string
	if currentTriggers.Valid && currentTriggers.String != "" {
		// Replace old trigger with new one
		triggers := strings.Split(currentTriggers.String, ",")
		for i, t := range triggers {
			t = strings.TrimSpace(t)
			if t == oldTrigger {
				triggers[i] = newTrigger
			}
		}
		newTriggerStr = strings.Join(triggers, ",")
	} else {
		// No existing triggers, just set the new one
		newTriggerStr = newTrigger
	}

	// Update lead with new trigger
	_, err = s.db.Exec("UPDATE leads SET trigger = $1 WHERE phone = $2", newTriggerStr, phone)
	if err != nil {
		logrus.Errorf("Failed to update lead trigger for %s: %v", phone, err)
	} else {
		logrus.Infof("Updated lead %s trigger from %s to %s", phone, oldTrigger, newTrigger)
	}
}

// cleanupStuckProcessing releases contacts stuck in processing
func (s *SequenceTriggerProcessor) cleanupStuckProcessing() error {
	query := `
		UPDATE sequence_contacts
		SET processing_device_id = NULL,
			processing_started_at = NULL,
			retry_count = retry_count + 1
		WHERE processing_device_id IS NOT NULL
			AND processing_started_at < $1
	`
	
	cutoffTime := time.Now().Add(-5 * time.Minute)
	_, err := s.db.Exec(query, cutoffTime)
	return err
}

// DeviceLoad represents current device workload
type DeviceLoad struct {
	DeviceID          string
	Status            string
	MessagesHour      int
	MessagesToday     int
	IsAvailable       bool
	CurrentProcessing int
}

// CanAcceptMore checks if device can handle more messages
func (d DeviceLoad) CanAcceptMore() bool {
	return d.IsAvailable && 
		d.Status == "online" &&
		d.MessagesHour < 80 && // WhatsApp limit ~100/hour
		d.MessagesToday < 800 && // Daily limit ~1000
		d.CurrentProcessing < 50 // Don't overload single device
}

// monitorBroadcastResults monitors broadcast messages and updates sequence_contacts accordingly
func (s *SequenceTriggerProcessor) monitorBroadcastResults() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check for failed messages and handle them
			query := `
				UPDATE sequence_contacts sc
				SET status = 'failed',
					last_error = bm.error_message,
					retry_count = sc.retry_count + 1
				FROM broadcast_messages bm
				WHERE bm.sequence_id = sc.sequence_id
					AND bm.recipient_phone = sc.contact_phone
					AND bm.status = 'failed'
					AND sc.status = 'active'
					AND sc.processing_device_id IS NOT NULL
					AND bm.created_at > NOW() - INTERVAL '5 minutes'
			`
			
			result, err := s.db.Exec(query)
			if err == nil {
				if affected, _ := result.RowsAffected(); affected > 0 {
					logrus.Warnf("Marked %d sequence contacts as failed due to broadcast failures", affected)
				}
			}
			
		case <-s.stopChan:
			return
		}
	}
}