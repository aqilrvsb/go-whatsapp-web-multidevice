package repository

import (
	"database/sql"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type sequenceRepository struct {
	db *sql.DB
}

var sequenceRepo *sequenceRepository

// GetSequenceRepository returns sequence repository instance
func GetSequenceRepository() *sequenceRepository {
	if sequenceRepo == nil {
		sequenceRepo = &sequenceRepository{
			db: database.GetDB(),
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
		INSERT INTO sequences (id, user_id, device_id, name, description, niche, status, 
		                      trigger, start_trigger, end_trigger, total_days, is_active, time_schedule, 
		                      min_delay_seconds, max_delay_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	
	_, err := r.db.Exec(query, sequence.ID, sequence.UserID, nil, // device_id is NULL - sequences use all user devices
		sequence.Name, sequence.Description, sequence.Niche, sequence.Status, 
		sequence.Trigger, sequence.StartTrigger, sequence.EndTrigger, sequence.TotalDays, 
		sequence.IsActive, sequence.TimeSchedule, sequence.MinDelaySeconds, sequence.MaxDelaySeconds,
		sequence.CreatedAt, sequence.UpdatedAt)
		
	return err
}

// GetSequences gets all sequences for a user
func (r *sequenceRepository) GetSequences(userID string) ([]models.Sequence, error) {
	query := `
		SELECT id, user_id, device_id, name, description, niche, status, 
		       COALESCE(start_trigger, '') as start_trigger,
		       COALESCE(end_trigger, '') as end_trigger,
		       total_days, is_active, 
		       COALESCE(schedule_time, '09:00') as schedule_time, 
		       COALESCE(min_delay_seconds, 10) as min_delay_seconds,
		       COALESCE(max_delay_seconds, 30) as max_delay_seconds,
		       created_at, updated_at
		FROM sequences
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		logrus.Errorf("Failed to query sequences: %v", err)
		return nil, err
	}
	defer rows.Close()
	
	var sequences []models.Sequence
	for rows.Next() {
		var seq models.Sequence
		err := rows.Scan(&seq.ID, &seq.UserID, &seq.DeviceID, &seq.Name, 
			&seq.Description, &seq.Niche, &seq.Status, &seq.StartTrigger, &seq.EndTrigger,
			&seq.TotalDays, &seq.IsActive, 
			&seq.TimeSchedule, &seq.MinDelaySeconds, &seq.MaxDelaySeconds,
			&seq.CreatedAt, &seq.UpdatedAt)
		if err != nil {
			logrus.Errorf("Failed to scan sequence row: %v", err)
			continue
		}
		sequences = append(sequences, seq)
	}
	
	logrus.Infof("Repository: Found %d sequences for user %s", len(sequences), userID)
	return sequences, nil
}
// GetSequenceByID gets sequence by ID
func (r *sequenceRepository) GetSequenceByID(sequenceID string) (*models.Sequence, error) {
	query := `
		SELECT id, user_id, device_id, name, description, niche, status, 
		       COALESCE(start_trigger, '') as start_trigger,
		       COALESCE(end_trigger, '') as end_trigger,
		       total_days, is_active, 
		       COALESCE(schedule_time, '09:00') as schedule_time,
		       COALESCE(min_delay_seconds, 10) as min_delay_seconds,
		       COALESCE(max_delay_seconds, 30) as max_delay_seconds,
		       created_at, updated_at
		FROM sequences
		WHERE id = $1
	`
	
	var seq models.Sequence
	err := r.db.QueryRow(query, sequenceID).Scan(&seq.ID, &seq.UserID, &seq.DeviceID, 
		&seq.Name, &seq.Description, &seq.Niche, &seq.Status, &seq.StartTrigger, &seq.EndTrigger,
		&seq.TotalDays, &seq.IsActive, 
		&seq.TimeSchedule, &seq.MinDelaySeconds, &seq.MaxDelaySeconds,
		&seq.CreatedAt, &seq.UpdatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.Errorf("Sequence not found: %s", sequenceID)
		} else {
			logrus.Errorf("Failed to get sequence %s: %v", sequenceID, err)
		}
		return nil, err
	}
	
	return &seq, nil
}

// UpdateSequence updates a sequence
func (r *sequenceRepository) UpdateSequence(sequence *models.Sequence) error {
	sequence.UpdatedAt = time.Now()
	
	query := `
		UPDATE sequences 
		SET name = $1, description = $2, niche = $3, status = $4, 
		    start_trigger = $5, end_trigger = $6, total_days = $7, 
		    is_active = $8, schedule_time = $9, min_delay_seconds = $10, 
		    max_delay_seconds = $11, updated_at = $12
		WHERE id = $13
	`
	
	_, err := r.db.Exec(query, sequence.Name, sequence.Description, sequence.Niche, 
		sequence.Status, sequence.StartTrigger, sequence.EndTrigger, sequence.TotalDays, 
		sequence.IsActive, sequence.TimeSchedule, sequence.MinDelaySeconds, 
		sequence.MaxDelaySeconds, sequence.UpdatedAt, sequence.ID)
		
	return err
}

// DeleteSequence deletes a sequence
func (r *sequenceRepository) DeleteSequence(sequenceID string) error {
	_, err := r.db.Exec("DELETE FROM sequences WHERE id = $1", sequenceID)
	return err
}

// DeleteSequenceSteps deletes all steps for a sequence
func (r *sequenceRepository) DeleteSequenceSteps(sequenceID string) error {
	_, err := r.db.Exec("DELETE FROM sequence_steps WHERE sequence_id = $1", sequenceID)
	return err
}

// CreateSequenceStep creates a step in sequence
func (r *sequenceRepository) CreateSequenceStep(step *models.SequenceStep) error {
	step.ID = uuid.New().String()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()

	query := `
		INSERT INTO sequence_steps (id, sequence_id, day, trigger, message_type, content, media_url, caption, send_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	_, err := r.db.Exec(query, step.ID, step.SequenceID, step.Day, step.Trigger, step.MessageType,
		step.Content, step.MediaURL, step.Caption, step.SendTime, step.CreatedAt, step.UpdatedAt)
		
	return err
}
// GetSequenceSteps gets all steps for a sequence
func (r *sequenceRepository) GetSequenceSteps(sequenceID string) ([]models.SequenceStep, error) {
	query := `
		SELECT id, sequence_id, day, COALESCE(trigger, '') as trigger, message_type, content, media_url, caption, send_time, created_at, updated_at
		FROM sequence_steps
		WHERE sequence_id = $1
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
		err := rows.Scan(&step.ID, &step.SequenceID, &step.Day, &step.Trigger, &step.MessageType,
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
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		WHERE sequence_id = $1
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
			 OR sc.last_message_at < DATE_SUB($1, INTERVAL 24 HOUR))
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
		SET current_day = $1, status = $2, last_message_at = $3
		WHERE id = $4
	`
	
	_, err := r.db.Exec(query, currentDay, status, now, contactID)
	return err
}

// MarkContactCompleted marks contact as completed
func (r *sequenceRepository) MarkContactCompleted(contactID string) error {
	now := time.Now()
	query := `
		UPDATE sequence_contacts 
		SET status = 'completed', completed_at = $1
		WHERE id = $2
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		WHERE sequence_id = $1
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
		SELECT COUNT(*) FROM sequence_logs WHERE sequence_id = $1
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