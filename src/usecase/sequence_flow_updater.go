package usecase

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type SequenceFlowUpdater struct {
	db *sql.DB
}

func NewSequenceFlowUpdater() *SequenceFlowUpdater {
	return &SequenceFlowUpdater{
		db: database.GetDB(),
	}
}

// FlowUpdate updates sequences with new steps without creating duplicates
// Logic: For each device+phone combination, find their highest day and create missing days
func (s *SequenceFlowUpdater) FlowUpdate(sequenceID string) (int, int, error) {
	logrus.Infof("🚀 Starting Flow Update for sequence: %s", sequenceID)
	
	// Get sequence info
	var sequenceName, scheduleTime string
	err := s.db.QueryRow("SELECT name, COALESCE(schedule_time, '09:00') FROM sequences WHERE id = ?", sequenceID).Scan(&sequenceName, &scheduleTime)
	if err != nil {
		return 0, 0, fmt.Errorf("sequence not found: %w", err)
	}
	
	// Step 1: Get template's current maximum day
	var templateMaxDay int
	err = s.db.QueryRow(`
		SELECT COALESCE(MAX(day_number), 0) 
		FROM sequence_steps 
		WHERE sequence_id = ?
	`, sequenceID).Scan(&templateMaxDay)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get template days: %w", err)
	}
	
	logrus.Infof("📋 Sequence '%s' template has %d days", sequenceName, templateMaxDay)
	
	if templateMaxDay == 0 {
		return 0, 0, fmt.Errorf("template has no steps")
	}
	
	// Step 2: Find all device+phone combinations and their highest day
	// Group by device_id AND recipient_phone to track each device-lead pair separately
	leadsQuery := `
		SELECT 
			bm.recipient_phone,
			bm.recipient_name,
			bm.device_id,
			bm.user_id,
			MAX(ss.day_number) as highest_day
		FROM broadcast_messages bm
		JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
		WHERE bm.sequence_id = ?
		GROUP BY bm.device_id, bm.recipient_phone, bm.recipient_name, bm.user_id
	`
	
	rows, err := s.db.Query(leadsQuery, sequenceID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get leads progress: %w", err)
	}
	defer rows.Close()
	
	type leadProgress struct {
		Phone      string
		Name       string
		DeviceID   string
		UserID     string
		HighestDay int
	}
	
	var leads []leadProgress
	for rows.Next() {
		var lead leadProgress
		err := rows.Scan(&lead.Phone, &lead.Name, &lead.DeviceID, &lead.UserID, &lead.HighestDay)
		if err != nil {
			logrus.Warnf("⚠️ Error scanning lead: %v", err)
			continue
		}
		leads = append(leads, lead)
	}
	
	if len(leads) == 0 {
		logrus.Warnf("⚠️ No leads found for sequence %s", sequenceName)
		return 0, 0, nil
	}
	
	logrus.Infof("📊 Found %d device-lead combinations to check", len(leads))
	
	// Step 3: Process each lead (device + phone combination)
	totalLeadsUpdated := 0
	totalMessagesCreated := 0
	broadcastRepo := repository.GetBroadcastRepository()
	
	for _, lead := range leads {
		// Skip if already up-to-date
		if lead.HighestDay >= templateMaxDay {
			logrus.Debugf("✅ Device %s + Phone %s already at day %d (template: %d) - skipping", 
				lead.DeviceID[:8], lead.Phone, lead.HighestDay, templateMaxDay)
			continue
		}
		
		// Calculate days to create
		startFromDay := lead.HighestDay + 1
		daysToCreate := templateMaxDay - lead.HighestDay
		
		logrus.Infof("📝 Device %s + Phone %s: Creating days %d to %d (%d messages)", 
			lead.DeviceID[:8], lead.Phone, startFromDay, templateMaxDay, daysToCreate)
		
		// Get steps from (highest_day + 1) to templateMaxDay
		stepsQuery := `
			SELECT 
				id, 
				day_number, 
				message_type, 
				COALESCE(content, '') as content,
				COALESCE(message_text, '') as message_text,
				media_url,
				COALESCE(min_delay_seconds, 5) as min_delay,
				COALESCE(max_delay_seconds, 15) as max_delay
			FROM sequence_steps
			WHERE sequence_id = ?
			AND day_number >= ?
			AND day_number <= ?
			ORDER BY day_number ASC
		`
		
		stepRows, err := s.db.Query(stepsQuery, sequenceID, startFromDay, templateMaxDay)
		if err != nil {
			logrus.Errorf("❌ Failed to get steps for lead %s: %v", lead.Phone, err)
			continue
		}
		
		// Schedule starting from tomorrow at the sequence's schedule_time
		// Malaysia timezone: Add 8 hours
		tomorrow := time.Now().AddDate(0, 0, 1)
		scheduleDate := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, time.UTC).Add(8 * time.Hour)
		
		messagesCreatedForLead := 0
		
		for stepRows.Next() {
			var step struct {
				ID          string
				DayNumber   int
				MessageType string
				Content     string
				MessageText string
				MediaURL    sql.NullString
				MinDelay    int
				MaxDelay    int
			}
			
			err := stepRows.Scan(&step.ID, &step.DayNumber, &step.MessageType, 
				&step.Content, &step.MessageText, &step.MediaURL, &step.MinDelay, &step.MaxDelay)
			if err != nil {
				logrus.Warnf("⚠️ Error scanning step: %v", err)
				continue
			}
			
			// Double-check: Skip if this day already exists for this device+phone
			var existingCount int
			checkQuery := `
				SELECT COUNT(*) 
				FROM broadcast_messages bm
				JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
				WHERE bm.sequence_id = ?
				AND bm.recipient_phone = ?
				AND bm.device_id = ?
				AND ss.day_number = ?
			`
			err = s.db.QueryRow(checkQuery, sequenceID, lead.Phone, lead.DeviceID, step.DayNumber).Scan(&existingCount)
			if err == nil && existingCount > 0 {
				logrus.Debugf("⏭️ Day %d already exists for device %s + phone %s - skipping", 
					step.DayNumber, lead.DeviceID[:8], lead.Phone)
				continue
			}
			
			// Use content from either field (prefer message_text)
			messageContent := step.Content
			if step.MessageText != "" {
				messageContent = step.MessageText
			}
			
			// Create the message
			msg := domainBroadcast.BroadcastMessage{
				ID:             uuid.New().String(),
				UserID:         lead.UserID,
				DeviceID:       lead.DeviceID,
				SequenceID:     &sequenceID,
				SequenceStepID: &step.ID,
				RecipientPhone: lead.Phone,
				RecipientName:  lead.Name,
				Message:        messageContent,
				Content:        messageContent,
				Type:           step.MessageType,
				MinDelay:       step.MinDelay,
				MaxDelay:       step.MaxDelay,
				ScheduledAt:    scheduleDate,
				Status:         "pending",
			}
			
			// Handle media URL
			if step.MediaURL.Valid && step.MediaURL.String != "" {
				msg.MediaURL = step.MediaURL.String
				msg.ImageURL = step.MediaURL.String
			}
			
			// Queue the message
			err = broadcastRepo.QueueMessage(msg)
			if err != nil {
				logrus.Errorf("❌ Failed to queue message for %s day %d: %v", 
					lead.Phone, step.DayNumber, err)
				continue
			}
			
			messagesCreatedForLead++
			totalMessagesCreated++
			
			// Move to next day (add 24 hours)
			scheduleDate = scheduleDate.Add(24 * time.Hour)
			
			logrus.Debugf("✅ Created Day %d for %s (device: %s) - scheduled for %v", 
				step.DayNumber, lead.Phone, lead.DeviceID[:8], scheduleDate)
		}
		stepRows.Close()
		
		if messagesCreatedForLead > 0 {
			totalLeadsUpdated++
			logrus.Infof("✅ Device %s + Phone %s: Created %d new messages (Days %d-%d)", 
				lead.DeviceID[:8], lead.Phone, messagesCreatedForLead, startFromDay, startFromDay+messagesCreatedForLead-1)
		}
	}
	
	logrus.Infof("🎉 Flow Update completed: Updated %d device-lead combinations with %d total messages", 
		totalLeadsUpdated, totalMessagesCreated)
	
	return totalLeadsUpdated, totalMessagesCreated, nil
}
