package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
)

type sequenceRepository struct {
	db *sql.DB
}

var sequenceRepo *sequenceRepository

// GetSequenceRepository returns sequence repository instance
func GetSequenceRepository() *sequenceRepository {
	if sequenceRepo == nil {
		sequenceRepo = &sequenceRepository{
			db: db,
		}
	}
	return sequenceRepo
}

// CreateSequence creates a new sequence
func (r *sequenceRepository) CreateSequence(sequence *models.Sequence) error {
	sequence.ID = uuid.New().String()
	sequence.CreatedAt = time.Now()
	sequence.UpdatedAt = time.Now()

	query := `
		INSERT INTO sequences (id, user_id, device_id, name, description, niche, total_days, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, sequence.ID, sequence.UserID, sequence.DeviceID, 
		sequence.Name, sequence.Description, sequence.Niche, sequence.TotalDays, sequence.IsActive,
		sequence.CreatedAt, sequence.UpdatedAt)
		
	return err
}

// GetSequences gets all sequences for a user
func (r *sequenceRepository) GetSequences(userID string) ([]models.Sequence, error) {
	query := `
		SELECT id, user_id, device_id, name, description, total_days, is_active, created_at, updated_at
		FROM sequences
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var sequences []models.Sequence
	for rows.Next() {
		var seq models.Sequence
		err := rows.Scan(&seq.ID, &seq.UserID, &seq.DeviceID, &seq.Name, 
			&seq.Description, &seq.TotalDays, &seq.IsActive, &seq.CreatedAt, &seq.UpdatedAt)
		if err != nil {
			continue
		}
		sequences = append(sequences, seq)
	}
	
	return sequences, nil
}
// GetSequenceByID gets sequence by ID
func (r *sequenceRepository) GetSequenceByID(sequenceID string) (*models.Sequence, error) {
	query := `
		SELECT id, user_id, device_id, name, description, total_days, is_active, created_at, updated_at
		FROM sequences
		WHERE id = ?
	`
	
	var seq models.Sequence
	err := r.db.QueryRow(query, sequenceID).Scan(&seq.ID, &seq.UserID, &seq.DeviceID, 
		&seq.Name, &seq.Description, &seq.TotalDays, &seq.IsActive, &seq.CreatedAt, &seq.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &seq, nil
}

// UpdateSequence updates a sequence
func (r *sequenceRepository) UpdateSequence(sequence *models.Sequence) error {
	sequence.UpdatedAt = time.Now()
	
	query := `
		UPDATE sequences 
		SET name = ?, description = ?, total_days = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, sequence.Name, sequence.Description, 
		sequence.TotalDays, sequence.IsActive, sequence.UpdatedAt, sequence.ID)
		
	return err
}

// DeleteSequence deletes a sequence
func (r *sequenceRepository) DeleteSequence(sequenceID string) error {
	_, err := r.db.Exec("DELETE FROM sequences WHERE id = ?", sequenceID)
	return err
}

// CreateSequenceStep creates a step in sequence
func (r *sequenceRepository) CreateSequenceStep(step *models.SequenceStep) error {
	step.ID = uuid.New().String()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()

	query := `
		INSERT INTO sequence_steps (id, sequence_id, day, message_type, content, media_url, caption, send_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, step.ID, step.SequenceID, step.Day, step.MessageType,
		step.Content, step.MediaURL, step.Caption, step.SendTime, step.CreatedAt, step.UpdatedAt)
		
	return err
}
// GetSequenceSteps gets all steps for a sequence
func (r *sequenceRepository) GetSequenceSteps(sequenceID string) ([]models.SequenceStep, error) {
	query := `
		SELECT id, sequence_id, day, message_type, content, media_url, caption, send_time, created_at, updated_at
		FROM sequence_steps
		WHERE sequence_id = ?
		ORDER BY day ASC
	`
	
	rows, err := r.db.Query(query, sequenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var steps []models.SequenceStep
	for rows.Next() {
		var step models.SequenceStep
		err := rows.Scan(&step.ID, &step.SequenceID, &step.Day, &step.MessageType,
			&step.Content, &step.MediaURL, &step.Caption, &step.SendTime, 
			&step.CreatedAt, &step.UpdatedAt)
		if err != nil {
			continue
		}
		steps = append(steps, step)
	}
	
	return steps, nil
}

// AddContactToSequence adds a contact to sequence
func (r *sequenceRepository) AddContactToSequence(contact *models.SequenceContact) error {
	contact.ID = uuid.New().String()
	contact.AddedAt = time.Now()
	contact.CurrentDay = 0
	contact.Status = "active"

	query := `
		INSERT INTO sequence_contacts (id, sequence_id, contact_phone, contact_name, current_day, status, added_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (sequence_id, contact_phone) DO NOTHING
	`
	
	_, err := r.db.Exec(query, contact.ID, contact.SequenceID, contact.ContactPhone,
		contact.ContactName, contact.CurrentDay, contact.Status, contact.AddedAt)
		
	return err
}
// GetSequenceContacts gets all contacts in a sequence
func (r *sequenceRepository) GetSequenceContacts(sequenceID string) ([]models.SequenceContact, error) {
	query := `
		SELECT id, sequence_id, contact_phone, contact_name, current_day, status, 
			   added_at, last_message_at, completed_at
		FROM sequence_contacts
		WHERE sequence_id = ?
		ORDER BY added_at DESC
	`
	
	rows, err := r.db.Query(query, sequenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var contacts []models.SequenceContact
	for rows.Next() {
		var contact models.SequenceContact
		err := rows.Scan(&contact.ID, &contact.SequenceID, &contact.ContactPhone,
			&contact.ContactName, &contact.CurrentDay, &contact.Status,
			&contact.AddedAt, &contact.LastMessageAt, &contact.CompletedAt)
		if err != nil {
			continue
		}
		contacts = append(contacts, contact)
	}
	
	return contacts, nil
}

// GetActiveSequenceContacts gets contacts ready for next message
func (r *sequenceRepository) GetActiveSequenceContacts(currentTime time.Time) ([]models.SequenceContact, error) {
	query := `
		SELECT sc.id, sc.sequence_id, sc.contact_phone, sc.contact_name, 
			   sc.current_day, sc.status, sc.added_at, sc.last_message_at, sc.completed_at
		FROM sequence_contacts sc
		JOIN sequences s ON s.id = sc.sequence_id
		WHERE sc.status = 'active' 
		AND s.is_active = true
		AND (sc.last_message_at IS NULL 
			 OR sc.last_message_at < DATE_SUB(?, INTERVAL 24 HOUR))
	`
	
	rows, err := r.db.Query(query, currentTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var contacts []models.SequenceContact
	for rows.Next() {
		var contact models.SequenceContact
		err := rows.Scan(&contact.ID, &contact.SequenceID, &contact.ContactPhone,
			&contact.ContactName, &contact.CurrentDay, &contact.Status,
			&contact.AddedAt, &contact.LastMessageAt, &contact.CompletedAt)
		if err != nil {
			continue
		}
		contacts = append(contacts, contact)
	}
	
	return contacts, nil
}
// UpdateContactProgress updates contact's progress in sequence
func (r *sequenceRepository) UpdateContactProgress(contactID string, currentDay int, status string) error {
	now := time.Now()
	query := `
		UPDATE sequence_contacts 
		SET current_day = ?, status = ?, last_message_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, currentDay, status, now, contactID)
	return err
}

// MarkContactCompleted marks contact as completed
func (r *sequenceRepository) MarkContactCompleted(contactID string) error {
	now := time.Now()
	query := `
		UPDATE sequence_contacts 
		SET status = 'completed', completed_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, now, contactID)
	return err
}

// CreateSequenceLog logs a sent message
func (r *sequenceRepository) CreateSequenceLog(log *models.SequenceLog) error {
	log.ID = uuid.New().String()
	log.SentAt = time.Now()

	query := `
		INSERT INTO sequence_logs (id, sequence_id, contact_id, step_id, day, status, message_id, error_message, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, log.ID, log.SequenceID, log.ContactID, log.StepID,
		log.Day, log.Status, log.MessageID, log.ErrorMessage, log.SentAt)
		
	return err
}

// GetSequenceStats gets statistics for a sequence
func (r *sequenceRepository) GetSequenceStats(sequenceID string) (map[string]int, error) {
	stats := make(map[string]int)
	
	// Get contact counts by status
	query := `
		SELECT status, COUNT(*) as count
		FROM sequence_contacts
		WHERE sequence_id = ?
		GROUP BY status
	`
	
	rows, err := r.db.Query(query, sequenceID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			stats[status] = count
		}
	}
	
	// Get total messages sent
	var messageCount int
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM sequence_logs WHERE sequence_id = ?
	`, sequenceID).Scan(&messageCount)
	
	if err == nil {
		stats["messages_sent"] = messageCount
	}
	
	return stats, nil
}

// GetActiveSequencesWithNiche gets all active sequences that have a niche
func (r *sequenceRepository) GetActiveSequencesWithNiche() ([]models.Sequence, error) {
	query := `
		SELECT id, user_id, device_id, name, description, niche, total_days, is_active, created_at, updated_at
		FROM sequences
		WHERE is_active = true
		AND niche IS NOT NULL
		AND niche != ''
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var sequences []models.Sequence
	for rows.Next() {
		var seq models.Sequence
		err := rows.Scan(&seq.ID, &seq.UserID, &seq.DeviceID, &seq.Name, 
			&seq.Description, &seq.Niche, &seq.TotalDays, &seq.IsActive, 
			&seq.CreatedAt, &seq.UpdatedAt)
		if err != nil {
			continue
		}
		sequences = append(sequences, seq)
	}
	
	return sequences, nil
}