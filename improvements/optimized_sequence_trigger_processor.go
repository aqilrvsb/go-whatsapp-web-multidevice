package usecase

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/sirupsen/logrus"
)

// contactFlowJob represents a job for processing a single flow for a contact
type contactFlowJob struct {
	sequenceContactID string
	sequenceID        string
	sequenceStepID    string
	phone             string
	name              string
	currentTrigger    string
	currentStep       int
	messageText       string
	messageType       string
	mediaURL          sql.NullString
	nextTrigger       sql.NullString
	delayHours        int
	preferredDevice   sql.NullString
}

// OptimizedSequenceTriggerProcessor handles trigger-based sequence processing optimized for 3000 devices
type OptimizedSequenceTriggerProcessor struct {
	db              *sql.DB
	broadcastMgr    broadcast.BroadcastManagerInterface
	isRunning       bool
	stopChan        chan bool
	ticker          *time.Ticker
	mutex           sync.Mutex
	batchSize       int
	processInterval time.Duration
	workerCount     int
	
	// Metrics
	totalProcessed  int64
	totalFailed     int64
	totalEnrolled   int64
}

// NewOptimizedSequenceTriggerProcessor creates a new optimized trigger processor
func NewOptimizedSequenceTriggerProcessor(db *sql.DB) *OptimizedSequenceTriggerProcessor {
	return &OptimizedSequenceTriggerProcessor{
		db:              db,
		broadcastMgr:    broadcast.GetBroadcastManager(),
		stopChan:        make(chan bool),
		batchSize:       10000,           // Increased for 3000 devices
		processInterval: 10 * time.Second, // Faster checking
		workerCount:     100,              // More workers for parallel processing
	}
}

// Start begins the trigger processing
func (s *OptimizedSequenceTriggerProcessor) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("sequence trigger processor already running")
	}

	s.ticker = time.NewTicker(s.processInterval)
	s.isRunning = true

	go s.run()
	
	logrus.Info("Optimized Sequence trigger processor started with 3000 device support")
	return nil
}

// Stop halts the trigger processing
func (s *OptimizedSequenceTriggerProcessor) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return
	}

	s.ticker.Stop()
	s.stopChan <- true
	s.isRunning = false

	logrus.Info("Optimized Sequence trigger processor stopped")
}

// run is the main processing loop
func (s *OptimizedSequenceTriggerProcessor) run() {
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
func (s *OptimizedSequenceTriggerProcessor) processTriggers() {
	startTime := time.Now()
	logrus.Debug("Starting optimized trigger processing...")

	// Step 1: Process leads with triggers to enroll in sequences
	enrolledCount, err := s.enrollLeadsFromTriggers()
	if err != nil {
		logrus.Errorf("Error enrolling leads: %v", err)
	}
	atomic.AddInt64(&s.totalEnrolled, int64(enrolledCount))

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
	processedCount, failedCount, err := s.processSequenceContacts(deviceLoads)
	if err != nil {
		logrus.Errorf("Error processing sequence contacts: %v", err)
	}
	
	atomic.AddInt64(&s.totalProcessed, int64(processedCount))
	atomic.AddInt64(&s.totalFailed, int64(failedCount))

	duration := time.Since(startTime)
	
	// Calculate metrics for monitoring
	totalDevices := len(deviceLoads)
	activeDevices := 0
	for _, load := range deviceLoads {
		if load.IsAvailable {
			activeDevices++
		}
	}
	
	// Get cumulative metrics
	cumProcessed := atomic.LoadInt64(&s.totalProcessed)
	cumFailed := atomic.LoadInt64(&s.totalFailed)
	cumEnrolled := atomic.LoadInt64(&s.totalEnrolled)
	
	logrus.Infof("Sequence processing completed: enrolled=%d (total=%d), processed=%d (total=%d), failed=%d (total=%d), devices=%d/%d, duration=%v", 
		enrolledCount, cumEnrolled, processedCount, cumProcessed, failedCount, cumFailed, activeDevices, totalDevices, duration)
	
	// Log performance metrics
	if processedCount > 0 {
		avgTimePerMessage := duration / time.Duration(processedCount)
		messagesPerMinute := float64(processedCount) / duration.Minutes()
		logrus.Infof("Performance: %.2f msg/min, %v avg/msg", messagesPerMinute, avgTimePerMessage)
	}
}

// enrollLeadsFromTriggers checks leads for matching sequence triggers and creates flow records
func (s *OptimizedSequenceTriggerProcessor) enrollLeadsFromTriggers() (int, error) {
	// First, get sequences that are active and should run now
	activeSequencesQuery := `
		SELECT DISTINCT 
			s.id, s.trigger, s.schedule_time
		FROM sequences s
		WHERE s.is_active = true 
			AND s.status = 'active'
	`

	seqRows, err := s.db.Query(activeSequencesQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to query active sequences: %w", err)
	}
	defer seqRows.Close()

	type activeSequence struct {
		ID           string
		Trigger      string
		ScheduleTime sql.NullString
	}

	var activeSequences []activeSequence
	for seqRows.Next() {
		var seq activeSequence
		if err := seqRows.Scan(&seq.ID, &seq.Trigger, &seq.ScheduleTime); err != nil {
			continue
		}
		
		// Check if sequence should run based on schedule time
		if seq.ScheduleTime.Valid && seq.ScheduleTime.String != "" {
			// Parse schedule time and check if it's time to run
			if !s.isTimeToRun(seq.ScheduleTime.String) {
				continue
			}
		}
		
		activeSequences = append(activeSequences, seq)
	}

	if len(activeSequences) == 0 {
		return 0, nil
	}

	enrolledCount := 0
	
	// Process each active sequence
	for _, seq := range activeSequences {
		// Find leads matching this sequence trigger
		leadsQuery := `
			SELECT DISTINCT 
				l.id, l.phone, l.name, l.device_id, l.user_id
			FROM leads l
			WHERE l.trigger IS NOT NULL 
				AND l.trigger != ''
				AND position($1 in l.trigger) > 0
				AND NOT EXISTS (
					SELECT 1 FROM sequence_contacts sc
					WHERE sc.sequence_id = $2 AND sc.contact_phone = l.phone
				)
			LIMIT 1000
		`

		rows, err := s.db.Query(leadsQuery, seq.Trigger, seq.ID)
		if err != nil {
			logrus.Warnf("Error querying leads for sequence %s: %v", seq.ID, err)
			continue
		}

		for rows.Next() {
			var lead models.Lead
			if err := rows.Scan(&lead.ID, &lead.Phone, &lead.Name, &lead.DeviceID, &lead.UserID); err != nil {
				continue
			}

			// Enroll lead and create flow records
			if err := s.enrollContactWithFlowRecords(seq.ID, lead, seq.Trigger); err != nil {
				logrus.Warnf("Error enrolling contact %s: %v", lead.Phone, err)
				continue
			}

			enrolledCount++
		}
		rows.Close()
	}

	return enrolledCount, nil
}

// enrollContactWithFlowRecords creates individual records for each flow/step
func (s *OptimizedSequenceTriggerProcessor) enrollContactWithFlowRecords(sequenceID string, lead models.Lead, trigger string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all steps for this sequence
	stepsQuery := `
		SELECT id, trigger, next_trigger, trigger_delay_hours, is_entry_point, day_number
		FROM sequence_steps
		WHERE sequence_id = $1
		ORDER BY day_number ASC
	`

	rows, err := tx.Query(stepsQuery, sequenceID)
	if err != nil {
		return fmt.Errorf("failed to get sequence steps: %w", err)
	}
	defer rows.Close()

	// Create a record for each step
	entryTrigger := ""
	stepCount := 0
	
	for rows.Next() {
		var stepID, stepTrigger string
		var nextTrigger sql.NullString
		var delayHours, dayNumber int
		var isEntryPoint bool

		if err := rows.Scan(&stepID, &stepTrigger, &nextTrigger, &delayHours, &isEntryPoint, &dayNumber); err != nil {
			continue
		}

		// Keep track of entry trigger
		if isEntryPoint && stepTrigger == trigger {
			entryTrigger = stepTrigger
		}

		// Create sequence_contacts record for this step
		insertQuery := `
			INSERT INTO sequence_contacts (
				sequence_id, sequence_stepid, contact_phone, contact_name,
				current_step, current_day, current_trigger,
				next_trigger_time, status, enrolled_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (sequence_id, contact_phone) DO NOTHING
		`

		// Calculate when this step should be triggered
		var nextTriggerTime time.Time
		if stepCount == 0 && isEntryPoint {
			// First step (entry point) should be processed immediately
			nextTriggerTime = time.Now()
		} else {
			// Future steps should wait
			nextTriggerTime = time.Now().Add(100 * 365 * 24 * time.Hour) // Far future
		}

		status := "pending"
		if stepCount == 0 && isEntryPoint {
			status = "active"
		}

		_, err = tx.Exec(insertQuery, 
			sequenceID, stepID, lead.Phone, lead.Name,
			dayNumber, dayNumber, stepTrigger,
			nextTriggerTime, status, time.Now(), time.Now())
		
		if err != nil {
			logrus.Warnf("Failed to insert flow record for step %s: %v", stepID, err)
			continue
		}

		stepCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.Debugf("Enrolled %s in sequence %s with %d flow records", lead.Phone, sequenceID, stepCount)
	return nil
}

// processSequenceContacts processes contacts ready for their next message
func (s *OptimizedSequenceTriggerProcessor) processSequenceContacts(deviceLoads map[string]DeviceLoad) (int, int, error) {
	// Get contacts ready for processing with their specific step info
	query := `
		SELECT 
			sc.id, sc.sequence_id, sc.sequence_stepid, sc.contact_phone, sc.contact_name,
			sc.current_trigger, sc.current_step,
			ss.content, ss.message_type, ss.media_url,
			ss.next_trigger, ss.trigger_delay_hours,
			l.device_id as preferred_device_id,
			s.schedule_time
		FROM sequence_contacts sc
		JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
		JOIN sequences s ON s.id = sc.sequence_id
		LEFT JOIN leads l ON l.phone = sc.contact_phone
		WHERE sc.status = 'active'
			AND s.is_active = true
			AND s.status = 'active'
			AND sc.next_trigger_time <= $1
			AND sc.processing_device_id IS NULL
		ORDER BY sc.next_trigger_time ASC
		LIMIT $2
	`

	rows, err := s.db.Query(query, time.Now(), s.batchSize)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get contacts for processing: %w", err)
	}
	defer rows.Close()

	// Process in parallel with worker pool
	jobs := make(chan contactFlowJob, s.batchSize)
	results := make(chan bool, s.batchSize)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				success := s.processContactFlow(job, deviceLoads)
				results <- success
			}
		}()
	}

	// Queue jobs
	go func() {
		for rows.Next() {
			var job contactFlowJob
			var scheduleTime sql.NullString
			
			if err := rows.Scan(&job.sequenceContactID, &job.sequenceID, &job.sequenceStepID, 
				&job.phone, &job.name, &job.currentTrigger, &job.currentStep, 
				&job.messageText, &job.messageType, &job.mediaURL,
				&job.nextTrigger, &job.delayHours, &job.preferredDevice, &scheduleTime); err != nil {
				continue
			}
			
			// Check schedule time
			if scheduleTime.Valid && scheduleTime.String != "" {
				if !s.isTimeToRun(scheduleTime.String) {
					continue
				}
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

	// Count successes and failures
	processedCount := 0
	failedCount := 0
	for success := range results {
		if success {
			processedCount++
		} else {
			failedCount++
		}
	}

	return processedCount, failedCount, nil
}

// processContactFlow handles a single contact's flow/step message
func (s *OptimizedSequenceTriggerProcessor) processContactFlow(job contactFlowJob, deviceLoads map[string]DeviceLoad) bool {
	// Select best device for this contact
	deviceID := s.selectDeviceForContact(job.preferredDevice.String, deviceLoads)
	if deviceID == "" {
		logrus.Warnf("No available device for contact %s", job.phone)
		s.markContactFailed(job.sequenceContactID, "No available device")
		return false
	}

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
	logrus.Debugf("Applying %v delay before sending message to %s", delay, job.phone)
	time.Sleep(delay)

	// Create broadcast message
	broadcastMsg := domainBroadcast.BroadcastMessage{
		DeviceID:       deviceID,
		RecipientPhone: job.phone,
		Message:        job.messageText,
		Content:        job.messageText,
		Type:           job.messageType,
		SequenceStepID: job.sequenceStepID, // Track which step sent this
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

	// Update device load counter
	s.updateDeviceLoad(deviceID)

	// Update contact flow record as sent
	if err := s.updateContactFlowProgress(job.sequenceContactID, job.sequenceStepID, deviceID); err != nil {
		logrus.Errorf("Failed to update contact flow progress: %v", err)
		return false
	}

	// Schedule next flow if exists
	if job.nextTrigger.Valid && job.nextTrigger.String != "" {
		if err := s.scheduleNextFlow(job.sequenceID, job.phone, job.nextTrigger.String, job.delayHours); err != nil {
			logrus.Warnf("Failed to schedule next flow: %v", err)
		}
	} else {
		// Sequence complete - handle completion
		s.handleSequenceCompletion(job.sequenceID, job.phone, job.currentTrigger)
	}

	logrus.Debugf("Processed flow for contact %s, step %s, device %s", job.phone, job.sequenceStepID, deviceID)
	return true
}

// updateContactFlowProgress updates the flow record after successful send
func (s *OptimizedSequenceTriggerProcessor) updateContactFlowProgress(contactID, stepID, deviceID string) error {
	query := `
		UPDATE sequence_contacts 
		SET status = 'sent',
			completed_at = $1,
			processing_device_id = $2,
			processing_started_at = NULL
		WHERE id = $3
	`
	
	_, err := s.db.Exec(query, time.Now(), deviceID, contactID)
	return err
}

// scheduleNextFlow activates the next flow record for a contact
func (s *OptimizedSequenceTriggerProcessor) scheduleNextFlow(sequenceID, phone, nextTrigger string, delayHours int) error {
	nextTime := time.Now().Add(time.Duration(delayHours) * time.Hour)
	
	query := `
		UPDATE sequence_contacts 
		SET status = 'active',
			next_trigger_time = $1
		WHERE sequence_id = $2 
			AND contact_phone = $3 
			AND current_trigger = $4
			AND status = 'pending'
	`
	
	_, err := s.db.Exec(query, nextTime, sequenceID, phone, nextTrigger)
	return err
}

// handleSequenceCompletion handles when a contact completes all flows
func (s *OptimizedSequenceTriggerProcessor) handleSequenceCompletion(sequenceID, phone, lastTrigger string) {
	// Mark all remaining flows as completed
	query := `
		UPDATE sequence_contacts 
		SET status = 'completed',
			completed_at = $1
		WHERE sequence_id = $2 
			AND contact_phone = $3 
			AND status IN ('pending', 'active')
	`
	
	s.db.Exec(query, time.Now(), sequenceID, phone)
	
	// Remove trigger from lead
	s.removeCompletedTriggerFromLead(phone, lastTrigger)
	
	// Check if there's a chained sequence
	var nextSequenceTrigger sql.NullString
	checkQuery := `
		SELECT ss.next_trigger 
		FROM sequence_steps ss
		WHERE ss.sequence_id = $1 
			AND ss.trigger = $2
			AND ss.next_trigger IS NOT NULL
			AND ss.next_trigger != ''
			AND NOT EXISTS (
				SELECT 1 FROM sequence_steps ss2 
				WHERE ss2.sequence_id = $1 
				AND ss2.trigger = ss.next_trigger
			)
	`
	
	if err := s.db.QueryRow(checkQuery, sequenceID, lastTrigger).Scan(&nextSequenceTrigger); err == nil && nextSequenceTrigger.Valid {
		// Update lead with new sequence trigger
		s.updateLeadTrigger(phone, lastTrigger, nextSequenceTrigger.String)
		logrus.Infof("Lead %s completed sequence and moved to new trigger: %s", phone, nextSequenceTrigger.String)
	}
}

// Helper methods remain similar but optimized...

// selectDeviceForContact chooses the best device using advanced load balancing
func (s *OptimizedSequenceTriggerProcessor) selectDeviceForContact(preferredDeviceID string, loads map[string]DeviceLoad) string {
	// Try preferred device first if it's not overloaded
	if preferredDeviceID != "" {
		if load, ok := loads[preferredDeviceID]; ok && load.CanAcceptMore() && load.MessagesHour < 50 {
			return preferredDeviceID
		}
	}

	// Find least loaded device with capacity
	var bestDevice string
	minScore := float64(^uint(0) >> 1) // Max int

	for deviceID, load := range loads {
		if !load.CanAcceptMore() {
			continue
		}

		// Calculate load score (lower is better)
		score := float64(load.MessagesHour)*0.7 + float64(load.CurrentProcessing)*0.3
		
		if score < minScore {
			bestDevice = deviceID
			minScore = score
		}
	}

	return bestDevice
}

// isTimeToRun checks if current time matches schedule time
func (s *OptimizedSequenceTriggerProcessor) isTimeToRun(scheduleTime string) bool {
	if scheduleTime == "" {
		return true
	}

	// Parse schedule time (format: "HH:MM")
	parts := strings.Split(scheduleTime, ":")
	if len(parts) != 2 {
		return true
	}

	now := time.Now()
	currentHour := now.Hour()
	currentMin := now.Minute()

	var schedHour, schedMin int
	fmt.Sscanf(parts[0], "%d", &schedHour)
	fmt.Sscanf(parts[1], "%d", &schedMin)

	// Check if we're within 10 minutes of scheduled time
	schedMinutes := schedHour*60 + schedMin
	currentMinutes := currentHour*60 + currentMin

	diff := schedMinutes - currentMinutes
	if diff < 0 {
		diff = -diff
	}

	return diff <= 10 // Within 10 minutes window
}

// markContactFailed marks a contact flow as failed immediately
func (s *OptimizedSequenceTriggerProcessor) markContactFailed(contactID string, reason string) {
	query := `
		UPDATE sequence_contacts 
		SET status = 'failed',
			completed_at = $1,
			processing_device_id = NULL,
			processing_started_at = NULL
		WHERE id = $2
	`
	
	if _, err := s.db.Exec(query, time.Now(), contactID); err != nil {
		logrus.Errorf("Failed to mark contact as failed: %v", err)
	} else {
		logrus.Warnf("Contact flow %s marked as failed: %s", contactID, reason)
	}
}

// releaseContact releases a contact from processing
func (s *OptimizedSequenceTriggerProcessor) releaseContact(contactID string) {
	query := `
		UPDATE sequence_contacts 
		SET processing_device_id = NULL,
			processing_started_at = NULL,
			status = 'failed',
			completed_at = $1
		WHERE id = $2
	`
	s.db.Exec(query, time.Now(), contactID)
}

// removeCompletedTriggerFromLead removes trigger from lead when sequence completes
func (s *OptimizedSequenceTriggerProcessor) removeCompletedTriggerFromLead(phone, trigger string) {
	var currentTriggers sql.NullString
	err := s.db.QueryRow("SELECT trigger FROM leads WHERE phone = $1", phone).Scan(&currentTriggers)
	if err != nil || !currentTriggers.Valid {
		return
	}

	triggers := strings.Split(currentTriggers.String, ",")
	newTriggers := []string{}
	for _, t := range triggers {
		t = strings.TrimSpace(t)
		if t != trigger && t != "" {
			newTriggers = append(newTriggers, t)
		}
	}

	newTriggerStr := ""
	if len(newTriggers) > 0 {
		newTriggerStr = strings.Join(newTriggers, ",")
	}

	s.db.Exec("UPDATE leads SET trigger = NULLIF($1, '') WHERE phone = $2", newTriggerStr, phone)
}

// updateLeadTrigger updates lead trigger for sequence chaining
func (s *OptimizedSequenceTriggerProcessor) updateLeadTrigger(phone, oldTrigger, newTrigger string) {
	var currentTriggers sql.NullString
	err := s.db.QueryRow("SELECT trigger FROM leads WHERE phone = $1", phone).Scan(&currentTriggers)
	if err != nil {
		return
	}

	var newTriggerStr string
	if currentTriggers.Valid && currentTriggers.String != "" {
		triggers := strings.Split(currentTriggers.String, ",")
		for i, t := range triggers {
			t = strings.TrimSpace(t)
			if t == oldTrigger {
				triggers[i] = newTrigger
			}
		}
		newTriggerStr = strings.Join(triggers, ",")
	} else {
		newTriggerStr = newTrigger
	}

	_, err = s.db.Exec("UPDATE leads SET trigger = $1 WHERE phone = $2", newTriggerStr, phone)
	if err != nil {
		logrus.Errorf("Failed to update lead trigger for %s: %v", phone, err)
	} else {
		logrus.Infof("Updated lead %s trigger from %s to %s", phone, oldTrigger, newTrigger)
	}
}

// cleanupStuckProcessing releases contacts stuck in processing
func (s *OptimizedSequenceTriggerProcessor) cleanupStuckProcessing() error {
	query := `
		UPDATE sequence_contacts
		SET processing_device_id = NULL,
			processing_started_at = NULL,
			status = 'failed',
			completed_at = $1
		WHERE processing_device_id IS NOT NULL
			AND processing_started_at < $2
			AND status = 'active'
	`
	
	cutoffTime := time.Now().Add(-5 * time.Minute)
	_, err := s.db.Exec(query, time.Now(), cutoffTime)
	return err
}

// getDeviceWorkloads retrieves current device loads with optimized query
func (s *OptimizedSequenceTriggerProcessor) getDeviceWorkloads() (map[string]DeviceLoad, error) {
	query := `
		SELECT 
			d.id,
			d.status,
			COALESCE(dlb.messages_hour, 0) as messages_hour,
			COALESCE(dlb.messages_today, 0) as messages_today,
			COALESCE(dlb.is_available, true) as is_available,
			COUNT(DISTINCT sc.id) as current_processing
		FROM user_devices d
		LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
		LEFT JOIN sequence_contacts sc ON sc.processing_device_id = d.id 
			AND sc.processing_started_at > NOW() - INTERVAL '5 minutes'
			AND sc.status = 'active'
		WHERE d.status = 'online'
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

// updateDeviceLoad updates the device load counter after sending a message
func (s *OptimizedSequenceTriggerProcessor) updateDeviceLoad(deviceID string) {
	query := `
		INSERT INTO device_load_balance (device_id, messages_hour, messages_today, updated_at)
		VALUES ($1, 1, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (device_id) DO UPDATE
		SET messages_hour = device_load_balance.messages_hour + 1,
			messages_today = device_load_balance.messages_today + 1,
			updated_at = CURRENT_TIMESTAMP
	`
	
	if _, err := s.db.Exec(query, deviceID); err != nil {
		logrus.Warnf("Failed to update device load for %s: %v", deviceID, err)
	}
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
		d.MessagesHour < 80 &&     // WhatsApp limit ~100/hour
		d.MessagesToday < 800 &&   // Daily limit ~1000
		d.CurrentProcessing < 100  // Increased for 3000 device support
}
