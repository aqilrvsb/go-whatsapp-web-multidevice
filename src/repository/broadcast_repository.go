package repository

import (
	"database/sql"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/google/uuid"
)

type broadcastRepository struct {
	db *sql.DB
}

var broadcastRepo *broadcastRepository

// GetBroadcastRepository returns broadcast repository instance
func GetBroadcastRepository() *broadcastRepository {
	if broadcastRepo == nil {
		broadcastRepo = &broadcastRepository{
			db: database.GetDB(),
		}
	}
	return broadcastRepo
}

// QueueMessage adds a message to the queue
func (r *broadcastRepository) QueueMessage(msg domainBroadcast.BroadcastMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	
	query := `
		INSERT INTO message_queue 
		(id, device_id, message_type, reference_id, contact_phone, content, 
		 media_url, caption, priority, status, scheduled_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, msg.ID, msg.DeviceID, msg.Type, msg.ReferenceID,
		msg.Phone, msg.Content, msg.MediaURL, msg.Caption, msg.Priority,
		"pending", msg.ScheduledAt, time.Now())
		
	return err
}

// GetPendingMessages gets pending messages for processing
func (r *broadcastRepository) GetPendingMessages(limit int) ([]domainBroadcast.BroadcastMessage, error) {
	query := `
		SELECT id, device_id, message_type, reference_id, contact_phone, 
		       content, media_url, caption, priority, scheduled_at, retry_count
		FROM message_queue
		WHERE status = 'pending'
		AND (scheduled_at IS NULL OR scheduled_at <= ?)
		ORDER BY priority DESC, created_at ASC
		LIMIT ?
	`
	
	rows, err := r.db.Query(query, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var messages []domainBroadcast.BroadcastMessage
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var deviceID string
		
		err := rows.Scan(&msg.ID, &deviceID, &msg.Type, &msg.ReferenceID,
			&msg.Phone, &msg.Content, &msg.MediaURL, &msg.Caption,
			&msg.Priority, &msg.ScheduledAt, &msg.RetryCount)
		if err != nil {
			continue
		}
		
		// Set device ID
		msg.DeviceID = deviceID
		messages = append(messages, msg)
	}
	
	return messages, nil
}

// UpdateMessageStatus updates message status
func (r *broadcastRepository) UpdateMessageStatus(messageID, status string) error {
	query := `
		UPDATE message_queue 
		SET status = ?, processed_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, status, time.Now(), messageID)
	return err
}

// SetMessageError sets error message
func (r *broadcastRepository) SetMessageError(messageID, errorMsg string) error {
	query := `
		UPDATE message_queue 
		SET error_message = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, errorMsg, messageID)
	return err
}

// CreateBroadcastJob creates a new broadcast job
func (r *broadcastRepository) CreateBroadcastJob(jobType, referenceID, deviceID string, totalContacts int) (string, error) {
	jobID := uuid.New().String()
	
	query := `
		INSERT INTO broadcast_jobs 
		(id, job_type, reference_id, device_id, total_contacts, status, started_at)
		VALUES (?, ?, ?, ?, ?, 'running', ?)
	`
	
	_, err := r.db.Exec(query, jobID, jobType, referenceID, deviceID, totalContacts, time.Now())
	return jobID, err
}

// UpdateBroadcastJob updates broadcast job progress
func (r *broadcastRepository) UpdateBroadcastJob(jobID string, processed, successful, failed int) error {
	query := `
		UPDATE broadcast_jobs 
		SET processed_contacts = ?, successful_contacts = ?, failed_contacts = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, processed, successful, failed, jobID)
	return err
}

// CompleteBroadcastJob marks job as completed
func (r *broadcastRepository) CompleteBroadcastJob(jobID string) error {
	query := `
		UPDATE broadcast_jobs 
		SET status = 'completed', completed_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, time.Now(), jobID)
	return err
}