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
	"github.com/sirupsen/logrus"
)

// contactJob represents a job for processing a contact message
type contactJob struct {
	contactID        string
	sequenceID       string
	phone            string
	name             string
	currentTrigger   string
	currentDay       int
	messageText      string
	messageType      string
	mediaURL         sql.NullString
	nextTrigger      sql.NullString
	delayHours       int
	preferredDevice  sql.NullString
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
		batchSize:       1000,
		processInterval: 30 * time.Second, // Process every 30 seconds
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
	logrus.Infof("Trigger processing completed: enrolled=%d, processed=%d, duration=%v", 
		enrolledCount, processedCount, duration)
}

// enrollLeadsFromTriggers checks leads for matching sequence triggers
func (s *SequenceTriggerProcessor) enrollLeadsFromTriggers() (int, error) {
	query := `
		WITH active_sequences AS (
			SELECT s.id, ss.trigger as entry_trigger
			FROM sequences s
			JOIN sequence_steps ss ON ss.sequence_id = s.id
			WHERE s.is_active = true AND ss.is_entry_point = true
		),
		leads_to_process AS (
			SELECT DISTINCT l.id, l.phone, l.name, l.device_id
			FROM leads l
			WHERE l.trigger IS NOT NULL AND l.trigger != ''
		)
		SELECT l.*, a.id as sequence_id, a.entry_trigger
		FROM leads_to_process l
		CROSS JOIN active_sequences a
		WHERE position(a.entry_trigger in l.trigger) > 0
			AND NOT EXISTS (
				SELECT 1 FROM sequence_contacts sc
				WHERE sc.sequence_id = a.id AND sc.contact_phone = l.phone
			)
		LIMIT 1000
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

		if err := rows.Scan(&lead.ID, &lead.Phone, &lead.Name, &lead.DeviceID, 
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

// enrollContactInSequence adds a contact to a sequence
func (s *SequenceTriggerProcessor) enrollContactInSequence(sequenceID string, lead models.Lead, trigger string) error {
	query := `
		INSERT INTO sequence_contacts (
			sequence_id, contact_phone, contact_name, 
			current_step, current_day, current_trigger,
			next_trigger_time, status, enrolled_at
		) VALUES ($1, $2, $3, 1, 1, $4, $5, 'active', $6)
		ON CONFLICT (sequence_id, contact_phone) DO NOTHING
	`

	_, err := s.db.Exec(query, sequenceID, lead.Phone, lead.Name, 
		trigger, time.Now(), time.Now())
	
	if err != nil {
		return fmt.Errorf("failed to enroll contact: %w", err)
	}

	logrus.Debugf("Enrolled %s in sequence %s with trigger %s", lead.Phone, sequenceID, trigger)
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

// processSequenceContacts processes contacts ready for their next message
func (s *SequenceTriggerProcessor) processSequenceContacts(deviceLoads map[string]DeviceLoad) (int, error) {
	// Get contacts ready for processing
	query := `
		SELECT 
			sc.id, sc.sequence_id, sc.contact_phone, sc.contact_name,
			sc.current_trigger, sc.current_day,
			ss.message_text, ss.message_type, ss.media_url,
			ss.next_trigger, ss.trigger_delay_hours,
			l.device_id as preferred_device_id
		FROM sequence_contacts sc
		JOIN sequence_steps ss ON ss.trigger = sc.current_trigger
		JOIN sequences s ON s.id = sc.sequence_id
		LEFT JOIN leads l ON l.phone = sc.contact_phone
		WHERE sc.status = 'active'
			AND s.is_active = true
			AND sc.next_trigger_time <= $1
			AND sc.processing_device_id IS NULL
		ORDER BY s.priority DESC, sc.next_trigger_time ASC
		LIMIT $2
	`

	rows, err := s.db.Query(query, time.Now(), s.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get contacts for processing: %w", err)
	}
	defer rows.Close()

	// Process in parallel with worker pool
	jobs := make(chan contactJob, s.batchSize)
	results := make(chan bool, s.batchSize)
	
	// Start workers
	numWorkers := 10
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
				&job.currentTrigger, &job.currentDay, &job.messageText, &job.messageType,
				&job.mediaURL, &job.nextTrigger, &job.delayHours, &job.preferredDevice); err != nil {
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
	// Create broadcast message
	broadcastMsg := domainBroadcast.BroadcastMessage{
		DeviceID:       deviceID,
		RecipientPhone: job.phone,
		Message:        job.messageText,
		Content:        job.messageText,
		Type:           job.messageType,
	}

	if job.mediaURL.Valid && job.mediaURL.String != "" {
		broadcastMsg.MediaURL = job.mediaURL.String
	}

	// Send to broadcast manager
	if err := s.broadcastMgr.SendMessage(broadcastMsg); err != nil {
		logrus.Errorf("Failed to queue message for %s: %v", job.phone, err)
		s.releaseContact(job.contactID)
		return false
	}

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
	// Try preferred device first
	if preferredDeviceID != "" {
		if load, ok := loads[preferredDeviceID]; ok && load.CanAcceptMore() {
			return preferredDeviceID
		}
	}

	// Find least loaded device
	var bestDevice string
	minLoad := int(^uint(0) >> 1) // Max int

	for deviceID, load := range loads {
		if load.CanAcceptMore() && load.CurrentProcessing < minLoad {
			bestDevice = deviceID
			minLoad = load.CurrentProcessing
		}
	}

	return bestDevice
}

// updateContactProgress moves contact to next step
func (s *SequenceTriggerProcessor) updateContactProgress(contactID string, nextTrigger sql.NullString, delayHours int) error {
	if !nextTrigger.Valid || nextTrigger.String == "" {
		// Sequence complete
		query := `
			UPDATE sequence_contacts 
			SET status = 'completed', 
				completed_at = $1,
				current_trigger = NULL,
				processing_device_id = NULL,
				processing_started_at = NULL
			WHERE id = $2
		`
		_, err := s.db.Exec(query, time.Now(), contactID)
		return err
	}

	// Move to next step
	nextTime := time.Now().Add(time.Duration(delayHours) * time.Hour)
	query := `
		UPDATE sequence_contacts 
		SET current_trigger = $1,
			next_trigger_time = $2,
			current_day = current_day + 1,
			last_sent_at = $3,
			processing_device_id = NULL,
			processing_started_at = NULL,
			retry_count = 0
		WHERE id = $4
	`
	
	_, err := s.db.Exec(query, nextTrigger.String, nextTime, time.Now(), contactID)
	return err
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