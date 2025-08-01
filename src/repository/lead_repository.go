package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/sirupsen/logrus"
)

type leadRepository struct {
	db *sql.DB
}

var leadRepo *leadRepository

// GetLeadRepository returns lead repository instance
func GetLeadRepository() *leadRepository {
	if leadRepo == nil {
		leadRepo = &leadRepository{
			db: database.GetDB(),
		}
	}
	return leadRepo
}

// CreateLead creates a new lead
func (r *leadRepository) CreateLead(lead *models.Lead) error {
	// Let database generate the ID (SERIAL)
	lead.CreatedAt = time.Now()
	lead.UpdatedAt = time.Now()

	query := `
		INSERT INTO leads(device_id, user_id, name, phone, niche, journey, status, target_status, ` + "`trigger`" + `, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// Map Notes to journey column
	journey := lead.Notes
	
	// Default target_status if empty
	if lead.TargetStatus == "" {
		lead.TargetStatus = "prospect"
	}
	
	// Status can be empty or use a default
	status := lead.Status
	if status == "" {
		status = "new"  // Default status
	}
	
	// Debug logging
	log.Printf("CreateLead - DeviceID: %s, UserID: %s, Name: %s, Phone: %s", lead.DeviceID, lead.UserID, lead.Name, lead.Phone)
	log.Printf("CreateLead - Niche: %s, Status: %s, TargetStatus: %s, Platform: %s", lead.Niche, status, lead.TargetStatus, lead.Platform)
	log.Printf("CreateLead - Journey: %s, Trigger: %s", journey, lead.Trigger)
	
	result, err := r.db.Exec(query, lead.DeviceID, lead.UserID, lead.Name, lead.Phone, 
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.Platform, lead.CreatedAt, lead.UpdatedAt)
	
	if err != nil {
		log.Printf("CreateLead - Error executing query: %v", err)
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	lead.ID = fmt.Sprintf("%d", id)
	return nil
}

// GetLeadsByNiche gets all leads matching a niche (supports comma-separated niches)
func (r *leadRepository) GetLeadsByNiche(niche string) ([]models.Lead, error) {
	// Use LIKE pattern to match leads that contain this niche
	// This will match:
	// - Exact match: niche = 'ITADRESS'
	// - As first item: niche = 'ITADRESS,OTHER'
	// - As middle item: niche = 'OTHER,ITADRESS,MORE'
	// - As last item: niche = 'OTHER,ITADRESS'
	query := `
		SELECT id, device_id, user_id, name, phone, niche, journey, status, 
		       COALESCE(target_status, 'prospect') AS target_status, ` + "`trigger`" + `, created_at, updated_at
		FROM leads
		WHERE niche = ? 
		   OR niche LIKE ? 
		   OR niche LIKE ? 
		   OR niche LIKE ?
		ORDER BY created_at DESC
	`
	
	// Pattern matching for comma-separated values
	exactMatch := niche
	startsWithPattern := niche + ",%"
	endsWithPattern := "%," + niche
	containsPattern := "%," + niche + ",%"
	
	rows, err := r.db.Query(query, exactMatch, startsWithPattern, endsWithPattern, containsPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	for rows.Next() {
		var lead models.Lead
		var journey sql.NullString
		var trigger sql.NullString
		err := rows.Scan(&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Niche, &journey, &lead.Status, &lead.TargetStatus, &trigger,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning lead in GetLeadsByNiche: %v", err)
			continue
		}
		// Map journey to Notes field
		if journey.Valid {
			lead.Notes = journey.String
		}
		// Map trigger
		if trigger.Valid {
			lead.Trigger = trigger.String
		}
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// GetLeadsByNicheAndStatus gets all leads matching a niche AND status for a specific user
func (r *leadRepository) GetLeadsByNicheAndStatus(niche string, status string) ([]models.Lead, error) {
	// This needs to be updated to accept deviceID parameter
	// For now, return empty to prevent cross-device data leakage
	logrus.Warnf("GetLeadsByNicheAndStatus called without device filtering - this is a security issue!")
	return []models.Lead{}, nil
}

// GetLeadsByDeviceNicheAndStatus gets leads for a specific device matching niche and status
func (r *leadRepository) GetLeadsByDeviceNicheAndStatus(deviceID, niche, status string) ([]models.Lead, error) {
	// Trim whitespace from niche to avoid matching issues
	niche = strings.TrimSpace(niche)
	
	query := `
		SELECT id, device_id, user_id, name, phone, niche, journey, status, target_status, ` + "`trigger`" + `, created_at, updated_at
		FROM leads
		WHERE device_id = ?
		AND (? = '' OR niche LIKE CONCAT('%', ?, '%'))
		AND (? = '' OR target_status = ?)
		ORDER BY created_at DESC
	`
	
	// Debug logging
	logrus.Debugf("GetLeadsByDeviceNicheAndStatus - DeviceID: '%s', Niche: '%s' (len=%d), Status: '%s'", 
		deviceID, niche, len(niche), status)
	
	rows, err := r.db.Query(query, deviceID, niche, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	var foundCount int
	for rows.Next() {
		var lead models.Lead
		var journey sql.NullString
		var targetStatus sql.NullString
		var trigger sql.NullString
		
		err := rows.Scan(&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Niche, &journey, &lead.Status, &targetStatus, &trigger,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			continue
		}
		
		// Journey field is stored in DB but not in model, skip it
		
		if targetStatus.Valid {
			lead.TargetStatus = targetStatus.String
		}
		
		if trigger.Valid {
			lead.Trigger = trigger.String
		}
		
		// Debug: Show what niche was found
		logrus.Debugf("Found lead with niche: '%s' (searching for '%s')", lead.Niche, niche)
		
		leads = append(leads, lead)
		foundCount++
	}
	
	logrus.Debugf("GetLeadsByDeviceNicheAndStatus - Found %d leads with niche containing '%s'", foundCount, niche)
	
	return leads, nil
}

// GetNewLeadsForSequence gets new leads matching niche that aren't in sequence
func (r *leadRepository) GetNewLeadsForSequence(niche, sequenceID string) ([]models.Lead, error) {
	query := `

		SELECT l.id, l.user_id, l.name, l.phone, l.niche, 
		       l.journey, l.status, l.created_at, l.updated_at
		FROM leads l
		WHERE l.niche LIKE CONCAT('%', ?, '%')
		AND NOT EXISTS (
			SELECT 1 FROM sequence_contacts sc 
			WHERE sc.sequence_id = ? 
			AND sc.contact_phone = l.phone
		)
		ORDER BY l.created_at DESC
	`
	
	rows, err := r.db.Query(query, niche, sequenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	for rows.Next() {
		var lead models.Lead
		var journey sql.NullString
		err := rows.Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Niche, &journey, &lead.Status,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning lead in GetNewLeadsForSequence: %v", err)
			continue
		}
		// Map journey to Notes field
		if journey.Valid {
			lead.Notes = journey.String
		}
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// GetLeadsByDevice gets all leads for a specific user's device
func (r *leadRepository) GetLeadsByDevice(userID, deviceID string) ([]models.Lead, error) {
	query := `
		SELECT id, device_id, user_id, name, phone, niche, journey, status, target_status, ` + "`trigger`" + `, created_at, updated_at
		FROM leads
		WHERE user_id = ? AND device_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	for rows.Next() {
		var lead models.Lead
		var journey sql.NullString
		var targetStatus sql.NullString
		var trigger sql.NullString
		
		err := rows.Scan(&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Niche, &journey, &lead.Status, &targetStatus, &trigger,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			continue
		}
		
		// Map journey to Notes field
		if journey.Valid {
			lead.Notes = journey.String
		}
		
		// Map target_status
		if targetStatus.Valid {
			lead.TargetStatus = targetStatus.String
		}
		
		// Map trigger
		if trigger.Valid {
			lead.Trigger = trigger.String
		}
		
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// UpdateLead updates an existing lead
func (r *leadRepository) UpdateLead(id string, lead *models.Lead) error {
	lead.UpdatedAt = time.Now()
	
	query := `
		UPDATE leads 
		SET device_id = ?, name = ?, phone = ?, niche = ?, 
		    journey = ?, status = ?, target_status = ?, ` + "`trigger`" + ` = ?, updated_at = ?
		WHERE id = ?
	`
	
	// Map Notes to journey column
	journey := lead.Notes
	
	// Default status if empty
	status := lead.Status
	if status == "" {
		status = "new"
	}
	
	// Default target_status if empty
	if lead.TargetStatus == "" {
		lead.TargetStatus = "prospect"
	}
	
	result, err := r.db.Exec(query, lead.DeviceID, lead.Name, lead.Phone,
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.UpdatedAt, id)
	if err != nil {
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found")
	}
	
	return nil
}

// DeleteLead deletes a lead
func (r *leadRepository) DeleteLead(id string) error {
	query := `DELETE FROM leads WHERE id = ?`
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found")
	}
	
	return nil
}


// GetLeadByDeviceUserPhone gets a lead by device_id, user_id and phone combination
func (r *leadRepository) GetLeadByDeviceUserPhone(deviceID, userID, phone string) (*models.Lead, error) {
	lead := &models.Lead{}
	query := `
		SELECT id, device_id, user_id, name, phone, email, niche, source, status, 
		       target_status, ` + "`trigger`" + `, notes, created_at, updated_at,
		       COALESCE(platform, '') AS platform
		FROM leads
		WHERE device_id = ? AND user_id = ? AND phone = ?
		LIMIT 1
	`
	
	err := r.db.QueryRow(query, deviceID, userID, phone).Scan(
		&lead.ID,
		&lead.DeviceID,
		&lead.UserID,
		&lead.Name,
		&lead.Phone,
		&lead.Email,
		&lead.Niche,
		&lead.Source,
		&lead.Status,
		&lead.TargetStatus,
		&lead.Trigger,
		&lead.Notes,
		&lead.CreatedAt,
		&lead.UpdatedAt,
		&lead.Platform,
	)
	
	if err != nil {
		return nil, err
	}
	
	return lead, nil
}

// GetLeadByDeviceUserPhoneNiche gets a lead by device_id, user_id, phone AND niche combination
func (r *leadRepository) GetLeadByDeviceUserPhoneNiche(deviceID, userID, phone, niche string) (*models.Lead, error) {
	lead := &models.Lead{}
	query := `
		SELECT id, device_id, user_id, name, phone, email, niche, source, status, 
		       target_status, ` + "`trigger`" + `, notes, created_at, updated_at,
		       COALESCE(platform, '') AS platform
		FROM leads
		WHERE device_id = ? AND user_id = ? AND phone = ? AND niche = ?
		LIMIT 1
	`
	
	err := r.db.QueryRow(query, deviceID, userID, phone, niche).Scan(
		&lead.ID,
		&lead.DeviceID,
		&lead.UserID,
		&lead.Name,
		&lead.Phone,
		&lead.Email,
		&lead.Niche,
		&lead.Source,
		&lead.Status,
		&lead.TargetStatus,
		&lead.Trigger,
		&lead.Notes,
		&lead.CreatedAt,
		&lead.UpdatedAt,
		&lead.Platform,
	)
	
	if err != nil {
		return nil, err
	}
	
	return lead, nil
}

// GetLeadsByPhone gets leads by phone number
func (r *leadRepository) GetLeadsByPhone(phone string) ([]models.Lead, error) {
	query := `
		SELECT id, device_id, user_id, name, phone, niche, journey, status, 
		       COALESCE(target_status, 'prospect') AS target_status, ` + "`trigger`" + `, created_at, updated_at
		FROM leads
		WHERE phone = ?
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	for rows.Next() {
		var lead models.Lead
		var notes sql.NullString
		
		err := rows.Scan(
			&lead.ID,
			&lead.DeviceID,
			&lead.UserID,
			&lead.Name,
			&lead.Phone,
			&lead.Niche,
			&notes,
			&lead.Status,
			&lead.TargetStatus,
			&lead.Trigger,
			&lead.CreatedAt,
			&lead.UpdatedAt,
		)
		
		if err != nil {
			logrus.Errorf("Error scanning lead: %v", err)
			continue
		}
		
		// Map journey column to Notes field
		if notes.Valid {
			lead.Notes = notes.String
		}
		
		leads = append(leads, lead)
	}
	
	return leads, nil
}
