package usecase

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DirectSequenceEnroller handles the new direct-to-broadcast approach
type DirectSequenceEnroller struct {
	db *sql.DB
}

// NewDirectSequenceEnroller creates a new enroller
func NewDirectSequenceEnroller(db *sql.DB) *DirectSequenceEnroller {
	return &DirectSequenceEnroller{db: db}
}

// EnrollLeadInSequence creates all broadcast messages directly
func (e *DirectSequenceEnroller) EnrollLeadInSequence(lead models.Lead, sequenceID string, initialTrigger string) error {
	logrus.Infof("Direct enrollment: Lead %s (%s) into sequence %s", lead.Phone, lead.Name, sequenceID)
	
	// Track all sequences to process (including linked ones)
	sequencesToProcess := []sequenceToEnroll{
		{ID: sequenceID, Trigger: initialTrigger, StartTime: time.Now().Add(5 * time.Minute)},
	}
	processedSequences := make(map[string]bool)
	allMessages := []broadcastMessage{}
	
	// Process all sequences including linked ones
	for len(sequencesToProcess) > 0 {
		current := sequencesToProcess[0]
		sequencesToProcess = sequencesToProcess[1:]
		
		if processedSequences[current.ID] {
			continue
		}
		processedSequences[current.ID] = true
		
		// Get all steps for current sequence
		steps, err := e.getSequenceSteps(current.ID)
		if err != nil {
			logrus.Errorf("Failed to get steps for sequence %s: %v", current.ID, err)
			continue
		}
		
		// Create messages for each step
		currentTime := current.StartTime
		for i, step := range steps {
			// Calculate scheduled time
			if i > 0 {
				// Add delay hours from previous step
				delayHours := time.Duration(steps[i-1].TriggerDelayHours) * time.Hour
				if delayHours == 0 {
					delayHours = 24 * time.Hour // Default 24 hours
				}
				currentTime = currentTime.Add(delayHours)
			}
			
			// Create broadcast message
			msg := broadcastMessage{
				ID:             uuid.New().String(),
				UserID:         lead.UserID,
				DeviceID:       lead.DeviceID, // Respect device from lead
				SequenceID:     current.ID,
				SequenceStepID: step.ID,
				RecipientPhone: lead.Phone,
				RecipientName:  lead.Name,
				MessageType:    step.MessageType,
				Content:        step.Content,
				MediaURL:       step.MediaURL,
				Status:         "pending",
				ScheduledAt:    currentTime,
				CreatedAt:      time.Now(),
			}
			
			allMessages = append(allMessages, msg)
			
			// Check if this step links to another sequence
			if step.NextTrigger != "" && !strings.Contains(step.NextTrigger, "_day") {
				// This is a sequence trigger, not a day trigger
				linkedSeqID, err := e.findSequenceByTrigger(step.NextTrigger)
				if err == nil && linkedSeqID != "" && !processedSequences[linkedSeqID] {
					// Add linked sequence to process
					nextStartTime := currentTime.Add(time.Duration(step.TriggerDelayHours) * time.Hour)
					sequencesToProcess = append(sequencesToProcess, sequenceToEnroll{
						ID:        linkedSeqID,
						Trigger:   step.NextTrigger,
						StartTime: nextStartTime,
					})
					logrus.Infof("Found linked sequence: %s via trigger %s, will start at %v", 
						linkedSeqID, step.NextTrigger, nextStartTime)
				}
			}
		}
		
		logrus.Infof("Created %d messages for sequence %s", len(steps), current.ID)
	}
	
	// Bulk insert all messages
	if len(allMessages) > 0 {
		err := e.bulkInsertBroadcastMessages(allMessages)
		if err != nil {
			return fmt.Errorf("failed to insert broadcast messages: %w", err)
		}
		logrus.Infof("Successfully enrolled lead %s with %d total messages across all linked sequences", 
			lead.Phone, len(allMessages))
	}
	
	return nil
}

// getSequenceSteps retrieves all steps for a sequence
func (e *DirectSequenceEnroller) getSequenceSteps(sequenceID string) ([]sequenceStep, error) {
	query := `
		SELECT 
			id, sequence_id, day_number, message_type, 
			content, media_url, trigger, next_trigger, 
			trigger_delay_hours, min_delay_seconds, max_delay_seconds
		FROM sequence_steps
		WHERE sequence_id = $1
		ORDER BY day_number ASC
	`
	
	rows, err := e.db.Query(query, sequenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var steps []sequenceStep
	for rows.Next() {
		var step sequenceStep
		var mediaURL, nextTrigger sql.NullString
		
		err := rows.Scan(
			&step.ID, &step.SequenceID, &step.DayNumber, &step.MessageType,
			&step.Content, &mediaURL, &step.Trigger, &nextTrigger,
			&step.TriggerDelayHours, &step.MinDelaySeconds, &step.MaxDelaySeconds,
		)
		if err != nil {
			logrus.Warnf("Error scanning step: %v", err)
			continue
		}
		
		if mediaURL.Valid {
			step.MediaURL = mediaURL.String
		}
		if nextTrigger.Valid {
			step.NextTrigger = nextTrigger.String
		}
		
		steps = append(steps, step)
	}
	
	return steps, nil
}

// findSequenceByTrigger finds a sequence ID by its entry trigger
func (e *DirectSequenceEnroller) findSequenceByTrigger(trigger string) (string, error) {
	var sequenceID string
	query := `
		SELECT DISTINCT s.id
		FROM sequences s
		JOIN sequence_steps ss ON ss.sequence_id = s.id
		WHERE ss.trigger = $1
		AND ss.is_entry_point = true
		AND s.is_active = true
		LIMIT 1
	`
	
	err := e.db.QueryRow(query, trigger).Scan(&sequenceID)
	if err != nil {
		return "", err
	}
	
	return sequenceID, nil
}

// bulkInsertBroadcastMessages inserts multiple messages efficiently
func (e *DirectSequenceEnroller) bulkInsertBroadcastMessages(messages []broadcastMessage) error {
	if len(messages) == 0 {
		return nil
	}
	
	// Build bulk insert query
	valueStrings := make([]string, 0, len(messages))
	valueArgs := make([]interface{}, 0, len(messages)*11)
	
	for i, msg := range messages {
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*11+1, i*11+2, i*11+3, i*11+4, i*11+5, i*11+6, i*11+7, i*11+8, i*11+9, i*11+10, i*11+11,
		))
		
		valueArgs = append(valueArgs,
			msg.ID, msg.UserID, msg.DeviceID, msg.SequenceID, msg.SequenceStepID,
			msg.RecipientPhone, msg.RecipientName, msg.MessageType, msg.Content,
			msg.Status, msg.ScheduledAt,
		)
	}
	
	query := fmt.Sprintf(`
		INSERT INTO broadcast_messages (
			id, user_id, device_id, sequence_id, sequence_stepid,
			recipient_phone, recipient_name, message_type, content,
			status, scheduled_at
		) VALUES %s
		ON CONFLICT (id) DO NOTHING
	`, strings.Join(valueStrings, ","))
	
	_, err := e.db.Exec(query, valueArgs...)
	return err
}

// Helper structs
type sequenceToEnroll struct {
	ID        string
	Trigger   string
	StartTime time.Time
}

type sequenceStep struct {
	ID                string
	SequenceID        string
	DayNumber         int
	MessageType       string
	Content           string
	MediaURL          string
	Trigger           string
	NextTrigger       string
	TriggerDelayHours int
	MinDelaySeconds   int
	MaxDelaySeconds   int
}

type broadcastMessage struct {
	ID             string
	UserID         string
	DeviceID       string
	SequenceID     string
	SequenceStepID string
	RecipientPhone string
	RecipientName  string
	MessageType    string
	Content        string
	MediaURL       string
	Status         string
	ScheduledAt    time.Time
	CreatedAt      time.Time
}
