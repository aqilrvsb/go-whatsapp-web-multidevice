package usecase

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/sirupsen/logrus"
)

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
		batchSize:       5000,
		processInterval: 15 * time.Second,
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
	
	logrus.Info("Sequence trigger processor started (Direct Broadcast Mode)")
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
	logrus.Debug("Starting trigger processing (Direct Broadcast Mode)...")

	// Process leads with triggers to enroll in sequences
	enrolledCount, err := s.enrollLeadsFromTriggers()
	if err != nil {
		logrus.Errorf("Error enrolling leads: %v", err)
	}

	duration := time.Since(startTime)
	logrus.Debugf("Sequence enrollment completed: enrolled=%d, duration=%v", 
		enrolledCount, duration)
}

// enrollLeadsFromTriggers checks leads for matching sequence triggers
func (s *SequenceTriggerProcessor) enrollLeadsFromTriggers() (int, error) {
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
				SELECT 1 FROM broadcast_messages bm
				WHERE bm.sequence_id = s.id 
				AND bm.recipient_phone = l.phone
				AND bm.status = 'pending'
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

		// Enroll in sequence using direct broadcast method
		if err := s.enrollContactInSequenceDirectBroadcast(sequenceID, lead, entryTrigger); err != nil {
			logrus.Warnf("Error enrolling contact %s: %v", lead.Phone, err)
			continue
		}

		enrolledCount++
	}

	return enrolledCount, nil
}

// enrollContactInSequenceDirectBroadcast creates messages directly in broadcast_messages
func (s *SequenceTriggerProcessor) enrollContactInSequenceDirectBroadcast(sequenceID string, lead models.Lead, trigger string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Track all messages to create
	var allMessages []domainBroadcast.BroadcastMessage
	currentTime := time.Now()
	scheduledAt := currentTime.Add(5 * time.Minute) // Initial 5-minute delay
	
	// Process this sequence and all linked sequences
	currentSequenceID := sequenceID
	processedSequences := make(map[string]bool) // Prevent infinite loops
	
	for currentSequenceID != "" && !processedSequences[currentSequenceID] {
		processedSequences[currentSequenceID] = true
		
		// Get sequence info including min/max delays
		var sequenceName string
		var minDelay, maxDelay int
		err := tx.QueryRow(`
			SELECT name, COALESCE(min_delay_seconds, 5), COALESCE(max_delay_seconds, 15)
			FROM sequences WHERE id = $1
		`, currentSequenceID).Scan(&sequenceName, &minDelay, &maxDelay)
		
		if err != nil {
			logrus.Warnf("Failed to get sequence info for %s: %v", currentSequenceID, err)
			break
		}
		
		// Get all steps for this sequence
		stepsQuery := `
			SELECT id, day_number, trigger, next_trigger, trigger_delay_hours,
				   message_type, content, media_url, 
				   COALESCE(min_delay_seconds, $1) as min_delay,
				   COALESCE(max_delay_seconds, $2) as max_delay
			FROM sequence_steps 
			WHERE sequence_id = $3 
			ORDER BY day_number ASC
		`
		
		rows, err := tx.Query(stepsQuery, minDelay, maxDelay, currentSequenceID)
		if err != nil {
			logrus.Errorf("Failed to get steps for sequence %s: %v", currentSequenceID, err)
			break
		}
		
		var lastStepNextTrigger string
		
		for rows.Next() {
			var step struct {
				ID                string
				DayNumber         int
				Trigger           string
				NextTrigger       sql.NullString
				TriggerDelayHours int
				MessageType       string
				Content           string
				MediaURL          sql.NullString
				MinDelay          int
				MaxDelay          int
			}
			
			err := rows.Scan(&step.ID, &step.DayNumber, &step.Trigger, 
				&step.NextTrigger, &step.TriggerDelayHours,
				&step.MessageType, &step.Content, &step.MediaURL,
				&step.MinDelay, &step.MaxDelay)
			if err != nil {
				logrus.Warnf("Error scanning step: %v", err)
				continue
			}
			
			// Create broadcast message
			msg := domainBroadcast.BroadcastMessage{
				UserID:         lead.UserID,
				DeviceID:       lead.DeviceID,
				SequenceID:     &currentSequenceID,
				SequenceStepID: &step.ID,
				RecipientPhone: lead.Phone,
				RecipientName:  lead.Name,
				Message:        step.Content,
				Content:        step.Content,
				Type:           step.MessageType,
				MinDelay:       step.MinDelay,
				MaxDelay:       step.MaxDelay,
				ScheduledAt:    scheduledAt,
				Status:         "pending",
			}
			
			if step.MediaURL.Valid && step.MediaURL.String != "" {
				msg.MediaURL = step.MediaURL.String
				msg.ImageURL = step.MediaURL.String
			}
			
			allMessages = append(allMessages, msg)
			
			// Calculate next scheduled time
			if step.TriggerDelayHours > 0 {
				scheduledAt = scheduledAt.Add(time.Duration(step.TriggerDelayHours) * time.Hour)
			} else {
				scheduledAt = scheduledAt.Add(24 * time.Hour) // Default 24 hours
			}
			
			// Track the last step's next_trigger for linking
			if step.NextTrigger.Valid {
				lastStepNextTrigger = step.NextTrigger.String
			}
			
			logrus.Debugf("Prepared message for %s - %s Step %d, scheduled at %v",
				lead.Phone, sequenceName, step.DayNumber, msg.ScheduledAt)
		}
		rows.Close()
		
		// Find next linked sequence
		currentSequenceID = ""
		if lastStepNextTrigger != "" {
			var nextSequenceID string
			err := tx.QueryRow(`
				SELECT id FROM sequences 
				WHERE trigger = $1
				LIMIT 1
			`, lastStepNextTrigger).Scan(&nextSequenceID)
			
			if err == nil {
				currentSequenceID = nextSequenceID
				logrus.Infof("Found linked sequence with trigger '%s': %s", 
					lastStepNextTrigger, nextSequenceID)
			}
		}
	}
	
	// Insert all messages into broadcast_messages
	for _, msg := range allMessages {
		insertQuery := `
			INSERT INTO broadcast_messages (
				user_id, device_id, sequence_id, sequence_stepid,
				recipient_phone, recipient_name, message_type,
				content, media_url, status, scheduled_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		
		_, err = tx.Exec(insertQuery,
			msg.UserID, msg.DeviceID, msg.SequenceID, msg.SequenceStepID,
			msg.RecipientPhone, msg.RecipientName, msg.Type,
			msg.Content, msg.MediaURL, msg.Status, msg.ScheduledAt)
		
		if err != nil {
			logrus.Errorf("Failed to insert broadcast message: %v", err)
			return fmt.Errorf("failed to insert broadcast message: %w", err)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	logrus.Infof("✅ Successfully enrolled %s in sequence chain - Created %d messages",
		lead.Phone, len(allMessages))
	
	// Remove trigger from lead after successful enrollment
	s.removeCompletedTriggerFromLead(lead.Phone, trigger)
	
	return nil
}

// removeCompletedTriggerFromLead removes a trigger from lead after completion
func (s *SequenceTriggerProcessor) removeCompletedTriggerFromLead(phone, trigger string) {
	// Simple approach - just clear the trigger field
	updateQuery := `UPDATE leads SET trigger = NULL WHERE phone = $1`
	
	_, err := s.db.Exec(updateQuery, phone)
	if err != nil {
		logrus.Warnf("Failed to remove trigger from lead %s: %v", phone, err)
	}
}

// DeviceLoad represents device workload info
type DeviceLoad struct {
	DeviceID          string
	Status            string
	MessagesHour      int
	MessagesToday     int
	IsAvailable       bool
	CurrentProcessing int
}
