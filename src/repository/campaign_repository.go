package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
)

type campaignRepository struct {
	db *sql.DB
}

var campaignRepo *campaignRepository

// GetCampaignRepository returns campaign repository instance
func GetCampaignRepository() *campaignRepository {
	if campaignRepo == nil {
		campaignRepo = &campaignRepository{
			db: database.GetDB(),
		}
	}
	return campaignRepo
}

// CreateCampaign creates a new campaign
func (r *campaignRepository) CreateCampaign(campaign *models.Campaign) error {
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	
	query := `
		INSERT INTO campaigns 
		(user_id, campaign_date, title, niche, message, image_url, 
		 scheduled_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	
	err := r.db.QueryRow(query, campaign.UserID, campaign.CampaignDate,
		campaign.Title, campaign.Niche, campaign.Message, campaign.ImageURL,
		campaign.ScheduledTime, campaign.Status,
		campaign.CreatedAt, campaign.UpdatedAt).Scan(&campaign.ID)
		
	return err
}

// GetCampaignByDateAndNiche gets campaigns by date and niche
func (r *campaignRepository) GetCampaignByDateAndNiche(scheduledDate, niche string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, niche, message, image_url, 
		       campaign_date, scheduled_time, status, created_at, updated_at
		FROM campaigns
		WHERE campaign_date = $1 AND niche = $2 AND status != 'sent'
	`
	
	rows, err := r.db.Query(query, scheduledDate, niche)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		var scheduledTime sql.NullString
		
		err := rows.Scan(&campaign.ID, &campaign.UserID,
			&campaign.Title, &campaign.Niche, &campaign.Message, &campaign.ImageURL,
			&campaign.CampaignDate, &scheduledTime, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt)
		if err != nil {
			continue
		}
		
		if scheduledTime.Valid {
			campaign.ScheduledTime = scheduledTime.String
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}

// UpdateCampaign updates a campaign
func (r *campaignRepository) UpdateCampaign(campaign *models.Campaign) error {
	campaign.UpdatedAt = time.Now()
	
	query := `
		UPDATE campaigns 
		SET title = $1, niche = $2, message = $3, image_url = $4,
		    campaign_date = $5, scheduled_time = $6, status = $7, updated_at = $8
		WHERE id = $9
	`
	
	_, err := r.db.Exec(query, campaign.Title, campaign.Niche, campaign.Message,
		campaign.ImageURL, campaign.CampaignDate, campaign.ScheduledTime,
		campaign.Status, campaign.UpdatedAt, campaign.ID)
		
	return err
}

// GetCampaigns gets all campaigns for a user
func (r *campaignRepository) GetCampaigns(userID string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, message, device_id, niche, image_url, 
		       scheduled_date, scheduled_time, status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1
		ORDER BY scheduled_date DESC, scheduled_time DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		var scheduledTime sql.NullString
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.DeviceID, &campaign.Niche, &campaign.ImageURL,
			&campaign.ScheduledDate, &scheduledTime, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		if scheduledTime.Valid {
			campaign.ScheduledTime = scheduledTime.String
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}


// DeleteCampaign deletes a campaign
func (r *campaignRepository) DeleteCampaign(campaignID string) error {
	query := `DELETE FROM campaigns WHERE id = $1`
	
	result, err := r.db.Exec(query, campaignID)
	if err != nil {
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("campaign not found")
	}
	
	return nil
}


// GetCampaignsByDate gets all campaigns scheduled for a specific date
func (r *campaignRepository) GetCampaignsByDate(scheduledDate string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, message, niche, image_url, 
		       campaign_date, scheduled_time, status, created_at, updated_at
		FROM campaigns
		WHERE campaign_date = $1 AND status != 'sent'
		ORDER BY scheduled_time ASC
	`
	
	rows, err := r.db.Query(query, scheduledDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		var scheduledTime sql.NullString
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.ImageURL,
			&campaign.CampaignDate, &scheduledTime, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		if scheduledTime.Valid {
			campaign.ScheduledTime = scheduledTime.String
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}
