package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
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
	lead.ID = uuid.New().String()
	lead.CreatedAt = time.Now()
	lead.UpdatedAt = time.Now()

	query := `
		INSERT INTO leads (id, user_id, name, phone, email, niche, source, status, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	_, err := r.db.Exec(query, lead.ID, lead.UserID, lead.Name, lead.Phone, 
		lead.Email, lead.Niche, lead.Source, lead.Status, lead.Notes,
		lead.CreatedAt, lead.UpdatedAt)
		
	return err
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
		SELECT id, user_id, name, phone, email, niche, source, status, notes, created_at, updated_at
		FROM leads
		WHERE niche = $1 
		   OR niche LIKE $2 
		   OR niche LIKE $3 
		   OR niche LIKE $4
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
		err := rows.Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Email, &lead.Niche, &lead.Source, &lead.Status, &lead.Notes,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			continue
		}
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// GetLeadsByNicheAndStatus gets all leads matching a niche AND status
func (r *leadRepository) GetLeadsByNicheAndStatus(niche string, status string) ([]models.Lead, error) {
	// First, get all leads matching the niche
	leads, err := r.GetLeadsByNiche(niche)
	if err != nil {
		return nil, err
	}
	
	// If no status specified, return all leads matching the niche
	if status == "" {
		return leads, nil
	}
	
	// Filter by status
	var filteredLeads []models.Lead
	for _, lead := range leads {
		if lead.Status == status {
			filteredLeads = append(filteredLeads, lead)
		}
	}
	
	return filteredLeads, nil
}

// GetNewLeadsForSequence gets new leads matching niche that aren't in sequence
func (r *leadRepository) GetNewLeadsForSequence(niche, sequenceID string) ([]models.Lead, error) {
	query := `
		SELECT l.id, l.user_id, l.name, l.phone, l.email, l.niche, 
		       l.source, l.status, l.notes, l.created_at, l.updated_at
		FROM leads l
		WHERE l.niche = $1
		AND NOT EXISTS (
			SELECT 1 FROM sequence_contacts sc 
			WHERE sc.sequence_id = $2 
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
		err := rows.Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Email, &lead.Niche, &lead.Source, &lead.Status, &lead.Notes,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			continue
		}
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// GetLeadsByDevice gets all leads for a specific user's device
func (r *leadRepository) GetLeadsByDevice(userID, deviceID string) ([]models.Lead, error) {
	query := `
		SELECT id, user_id, name, phone, email, niche, source, status, notes, created_at, updated_at
		FROM leads
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []models.Lead
	for rows.Next() {
		var lead models.Lead
		err := rows.Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Email, &lead.Niche, &lead.Source, &lead.Status, &lead.Notes,
			&lead.CreatedAt, &lead.UpdatedAt)
		if err != nil {
			continue
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
		SET name = $2, phone = $3, email = $4, niche = $5, 
		    source = $6, status = $7, notes = $8, updated_at = $9
		WHERE id = $1
	`
	
	result, err := r.db.Exec(query, id, lead.Name, lead.Phone, lead.Email,
		lead.Niche, lead.Source, lead.Status, lead.Notes, lead.UpdatedAt)
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
	query := `DELETE FROM leads WHERE id = $1`
	
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
