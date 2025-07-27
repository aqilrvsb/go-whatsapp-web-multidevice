package usecase

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// DirectBroadcastProcessor handles direct sequence enrollment without sequence_contacts
type DirectBroadcastProcessor struct {
	db        *sql.DB
	batchSize int
	mu        sync.Mutex
}

// NewDirectBroadcastProcessor creates new processor
func NewDirectBroadcastProcessor(db *sql.DB) *DirectBroadcastProcessor {
	return &DirectBroadcastProcessor{
		db:        db,
		batchSize: 100,
	}
}

// ProcessDirectEnrollments finds leads with triggers and enrolls them directly
func (p *DirectBroadcastProcessor) ProcessDirectEnrollments() (int, error) {
	// Find leads with triggers that match active sequence entry points
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
			AND l.device_id IS NOT NULL 
			AND l.user_id IS NOT NULL
			AND position(ss.trigger in l.trigger) > 0
			AND NOT EXISTS (
				SELECT 1 FROM broadcast_messages bm
				WHERE bm.sequence_id = s.id 
				AND bm.recipient_phone = l.phone
				AND bm.status IN ('pending', 'sent')
			)
		LIMIT $1
	`

	rows, err := p.db.Query(query, p.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to query leads: %w", err)
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

		// Validate UUIDs are not empty strings
		if lead.DeviceID == "" || lead.UserID == "" {
			logrus.Warnf("Skipping lead %s - empty device_id or user_id", lead.Phone)
			continue
		}

		// Enroll directly to broadcast_messages
		if err := p.enrollDirectBroadcast(sequenceID, lead, entryTrigger); err != nil {
			logrus.Warnf("Error enrolling %s: %v", lead.Phone, err)
			continue
		}

		enrolledCount++
	}

	return enrolledCount, nil
}

// enrollDirectBroadcast creates all messages directly in broadcast_messages
func (p *DirectBroadcastProcessor) enrollDirectBroadcast(sequenceID string, lead models.Lead, trigger string) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get broadcast repository
	broadcastRepo := repository.GetBroadcastRepository()
	
	// Track all messages to create
	var allMessages []domainBroadcast.BroadcastMessage
	scheduledAt := time.Now().Add(5 * time.Minute) // First message in 5 minutes
	currentSequenceID := sequenceID
	processedSequences := make(map[string]bool)

	// Process sequence chain (COLD → WARM → HOT)
	for currentSequenceID != "" {
		// Prevent infinite loops
		if processedSequences[currentSequenceID] {
			break
		}
		processedSequences[currentSequenceID] = true

		// Get sequence info
		var sequenceName string
		var minDelay, maxDelay int
		err := p.db.QueryRow(`
			SELECT name, COALESCE(min_delay_seconds, 5), COALESCE(max_delay_seconds, 15)
			FROM sequences WHERE id = $1
		`, currentSequenceID).Scan(&sequenceName, &minDelay, &maxDelay)
		
		if err != nil {
			logrus.Warnf("Failed to get sequence %s: %v", currentSequenceID, err)
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
		
		rows, err := p.db.Query(stepsQuery, minDelay, maxDelay, currentSequenceID)
		if err != nil {
			return fmt.Errorf("failed to get steps: %w", err)
		}

		var lastStepNextTrigger string
		for rows.Next() {
			var step struct {
				ID                string
				DayNumber         int
				Trigger           sql.NullString
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

			// Handle media URL
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

			// Track last step's next trigger
			if step.NextTrigger.Valid {
				lastStepNextTrigger = step.NextTrigger.String
			}

			logrus.Debugf("Prepared message for %s - %s Step %d, scheduled at %v",
				lead.Phone, sequenceName, step.DayNumber, msg.ScheduledAt)
		}
		rows.Close()

		// Find next linked sequence (don't check active status for linked sequences)
		currentSequenceID = ""
		if lastStepNextTrigger != "" && !strings.Contains(lastStepNextTrigger, "_day") {
			var nextSequenceID string
			err := p.db.QueryRow(`
				SELECT s.id FROM sequences s
				INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
				WHERE ss.is_entry_point = true AND ss.trigger = $1
				LIMIT 1
			`, lastStepNextTrigger).Scan(&nextSequenceID)
			
			if err == nil {
				currentSequenceID = nextSequenceID
				logrus.Infof("Found linked sequence with trigger '%s': %s", 
					lastStepNextTrigger, nextSequenceID)
			}
		}
	}

	// Insert all messages using repository (handles UUIDs properly)
	for _, msg := range allMessages {
		// Repository will handle ID generation and NULL values
		err := broadcastRepo.QueueMessage(msg)
		if err != nil {
			logrus.Errorf("Failed to queue message for %s: %v", msg.RecipientPhone, err)
			return fmt.Errorf("failed to queue message: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	logrus.Infof("✅ Direct enrollment successful for %s - Created %d messages",
		lead.Phone, len(allMessages))

	// Remove trigger from lead after successful enrollment
	p.removeCompletedTrigger(lead.Phone, trigger)

	return nil
}

// removeCompletedTrigger removes trigger from lead after enrollment
func (p *DirectBroadcastProcessor) removeCompletedTrigger(phone, trigger string) {
	_, err := p.db.Exec("UPDATE leads SET trigger = NULL WHERE phone = $1", phone)
	if err != nil {
		logrus.Errorf("Failed to remove trigger for %s: %v", phone, err)
	}
}
