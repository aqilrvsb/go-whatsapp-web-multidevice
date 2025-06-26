package repository

import (
	"database/sql"
	"time"

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
			db: db,
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, lead.ID, lead.UserID, lead.Name, lead.Phone, 
		lead.Email, lead.Niche, lead.Source, lead.Status, lead.Notes,
		lead.CreatedAt, lead.UpdatedAt)
		
	return err
}

// GetLeadsByNiche gets all leads matching a niche
func (r *leadRepository) GetLeadsByNiche(niche string) ([]models.Lead, error) {
	query := `
		SELECT id, user_id, name, phone, email, niche, source, status, notes, created_at, updated_at
		FROM leads
		WHERE niche = ?
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, niche)
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

// GetNewLeadsForSequence gets new leads matching niche that aren't in sequence
func (r *leadRepository) GetNewLeadsForSequence(niche, sequenceID string) ([]models.Lead, error) {
	query := `
		SELECT l.id, l.user_id, l.name, l.phone, l.email, l.niche, 
		       l.source, l.status, l.notes, l.created_at, l.updated_at
		FROM leads l
		WHERE l.niche = ?
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