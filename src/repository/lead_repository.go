package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
)

type Lead struct {
	ID              string    `json:"id"`
	DeviceID        string    `json:"device_id"`
	UserID          string    `json:"user_id"`
	Name            string    `json:"name"`
	Phone           string    `json:"phone"`
	Niche           string    `json:"niche"`
	Journey         string    `json:"journey"`
	Status          string    `json:"status"`
	LastInteraction *time.Time `json:"last_interaction"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LeadRepository struct {
	db *sql.DB
}

var (
	leadRepo     *LeadRepository
	leadRepoOnce sync.Once
)

// GetLeadRepository returns singleton instance of LeadRepository
func GetLeadRepository() *LeadRepository {
	leadRepoOnce.Do(func() {
		leadRepo = &LeadRepository{
			db: database.GetDB(),
		}
	})
	return leadRepo
}

// GetLeadsByDevice gets all leads for a specific device
func (r *LeadRepository) GetLeadsByDevice(userID string, deviceID string) ([]Lead, error) {
	query := `
		SELECT id, device_id, user_id, name, phone, niche, journey, status, 
		       last_interaction, created_at, updated_at
		FROM leads
		WHERE user_id = $1 AND device_id = $2
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leads []Lead
	for rows.Next() {
		var lead Lead
		err := rows.Scan(
			&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
			&lead.Niche, &lead.Journey, &lead.Status, &lead.LastInteraction,
			&lead.CreatedAt, &lead.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		leads = append(leads, lead)
	}
	
	return leads, nil
}

// CreateLead creates a new lead
func (r *LeadRepository) CreateLead(userID string, deviceID, name, phone, niche, journey, status string) (*Lead, error) {
	if status == "" {
		status = "new"
	}
	
	query := `
		INSERT INTO leads (device_id, user_id, name, phone, niche, journey, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, device_id, user_id, name, phone, niche, journey, status, last_interaction, created_at, updated_at
	`
	
	now := time.Now()
	var lead Lead
	
	err := r.db.QueryRow(query, deviceID, userID, name, phone, niche, journey, status, now, now).Scan(
		&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
		&lead.Niche, &lead.Journey, &lead.Status, &lead.LastInteraction,
		&lead.CreatedAt, &lead.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &lead, nil
}

// UpdateLead updates an existing lead
func (r *LeadRepository) UpdateLead(userID string, leadID, name, phone, niche, journey, status string) (*Lead, error) {
	query := `
		UPDATE leads
		SET name = $3, phone = $4, niche = $5, journey = $6, status = $7, updated_at = $8
		WHERE id = $1 AND user_id = $2
		RETURNING id, device_id, user_id, name, phone, niche, journey, status, last_interaction, created_at, updated_at
	`
	
	var lead Lead
	err := r.db.QueryRow(query, leadID, userID, name, phone, niche, journey, status, time.Now()).Scan(
		&lead.ID, &lead.DeviceID, &lead.UserID, &lead.Name, &lead.Phone,
		&lead.Niche, &lead.Journey, &lead.Status, &lead.LastInteraction,
		&lead.CreatedAt, &lead.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &lead, nil
}

// DeleteLead deletes a lead
func (r *LeadRepository) DeleteLead(userID string, leadID string) error {
	query := `DELETE FROM leads WHERE id = $1 AND user_id = $2`
	
	result, err := r.db.Exec(query, leadID, userID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("lead not found")
	}
	
	return nil
}

// UpdateLastInteraction updates the last interaction time for a lead
func (r *LeadRepository) UpdateLastInteraction(leadID string) error {
	query := `UPDATE leads SET last_interaction = $1 WHERE id = $2`
	
	_, err := r.db.Exec(query, time.Now(), leadID)
	return err
}
