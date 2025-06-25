package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
)

type Campaign struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	CampaignDate  string    `json:"campaign_date"`
	Title         string    `json:"title"`
	Niche         string    `json:"niche"`
	Message       string    `json:"message"`
	ImageURL      string    `json:"image_url"`
	ScheduledTime string    `json:"scheduled_time"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CampaignRepository struct {
	db *sql.DB
}

var (
	campaignRepo     *CampaignRepository
	campaignRepoOnce sync.Once
)

// GetCampaignRepository returns singleton instance of CampaignRepository
func GetCampaignRepository() *CampaignRepository {
	campaignRepoOnce.Do(func() {
		campaignRepo = &CampaignRepository{
			db: database.GetDB(),
		}
	})
	return campaignRepo
}

// GetCampaigns gets all campaigns for a user
func (r *CampaignRepository) GetCampaigns(userID string) ([]Campaign, error) {
	query := `
		SELECT id, user_id, campaign_date, title, COALESCE(niche, ''), message, COALESCE(image_url, ''), 
		       COALESCE(scheduled_time::text, ''), status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1
		ORDER BY campaign_date DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []Campaign
	for rows.Next() {
		var campaign Campaign
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.CampaignDate, &campaign.Title,
			&campaign.Niche, &campaign.Message, &campaign.ImageURL, &campaign.ScheduledTime,
			&campaign.Status, &campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}

// CreateCampaign creates a new campaign
func (r *CampaignRepository) CreateCampaign(userID string, campaignDate, title, niche, message, imageURL, scheduledTime string) (*Campaign, error) {
	query := `
		INSERT INTO campaigns (user_id, campaign_date, title, niche, message, image_url, scheduled_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, campaign_date) 
		DO UPDATE SET title = $3, niche = $4, message = $5, image_url = $6, scheduled_time = $7, updated_at = $10
		RETURNING id, user_id, campaign_date, title, COALESCE(niche, ''), message, COALESCE(image_url, ''), COALESCE(scheduled_time::text, ''), status, created_at, updated_at
	`
	
	now := time.Now()
	var campaign Campaign
	
	err := r.db.QueryRow(query, userID, campaignDate, title, niche, message, imageURL, scheduledTime, "scheduled", now, now).Scan(
		&campaign.ID, &campaign.UserID, &campaign.CampaignDate, &campaign.Title,
		&campaign.Niche, &campaign.Message, &campaign.ImageURL, &campaign.ScheduledTime,
		&campaign.Status, &campaign.CreatedAt, &campaign.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &campaign, nil
}

// UpdateCampaign updates an existing campaign
func (r *CampaignRepository) UpdateCampaign(userID string, campaignID, title, niche, message, imageURL, scheduledTime, status string) (*Campaign, error) {
	query := `
		UPDATE campaigns
		SET title = $3, niche = $4, message = $5, image_url = $6, scheduled_time = $7, status = $8, updated_at = $9
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, campaign_date, title, COALESCE(niche, ''), message, COALESCE(image_url, ''), COALESCE(scheduled_time::text, ''), status, created_at, updated_at
	`
	
	var campaign Campaign
	err := r.db.QueryRow(query, campaignID, userID, title, niche, message, imageURL, scheduledTime, status, time.Now()).Scan(
		&campaign.ID, &campaign.UserID, &campaign.CampaignDate, &campaign.Title,
		&campaign.Niche, &campaign.Message, &campaign.ImageURL, &campaign.ScheduledTime,
		&campaign.Status, &campaign.CreatedAt, &campaign.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &campaign, nil
}

// DeleteCampaign deletes a campaign
func (r *CampaignRepository) DeleteCampaign(userID string, campaignID string) error {
	query := `DELETE FROM campaigns WHERE id = $1 AND user_id = $2`
	
	result, err := r.db.Exec(query, campaignID, userID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("campaign not found")
	}
	
	return nil
}

// GetCampaignByDate gets a campaign for a specific date
func (r *CampaignRepository) GetCampaignByDate(userID string, date string) (*Campaign, error) {
	query := `
		SELECT id, user_id, campaign_date, title, COALESCE(niche, ''), message, COALESCE(image_url, ''), 
		       COALESCE(scheduled_time::text, ''), status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1 AND campaign_date = $2
	`
	
	var campaign Campaign
	err := r.db.QueryRow(query, userID, date).Scan(
		&campaign.ID, &campaign.UserID, &campaign.CampaignDate, &campaign.Title,
		&campaign.Niche, &campaign.Message, &campaign.ImageURL, &campaign.ScheduledTime,
		&campaign.Status, &campaign.CreatedAt, &campaign.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, err
	}
	
	return &campaign, nil
}
