package repository

import (
	"database/sql"
	"time"

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
			db: db,
		}
	}
	return campaignRepo
}

// CreateCampaign creates a new campaign
func (r *campaignRepository) CreateCampaign(campaign *models.Campaign) error {
	campaign.ID = uuid.New().String()
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	campaign.Status = "pending"

	query := `
		INSERT INTO campaigns 
		(id, user_id, device_id, title, niche, message, image_url, 
		 scheduled_date, scheduled_time, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, campaign.ID, campaign.UserID, campaign.DeviceID,
		campaign.Title, campaign.Niche, campaign.Message, campaign.ImageURL,
		campaign.ScheduledDate, campaign.ScheduledTime, campaign.Status,
		campaign.CreatedAt, campaign.UpdatedAt)
		
	return err
}

// GetCampaignsByDate gets campaigns scheduled for a specific date
func (r *campaignRepository) GetCampaignsByDate(date string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, device_id, title, niche, message, image_url,
		       scheduled_date, scheduled_time, status, created_at, updated_at
		FROM campaigns
		WHERE scheduled_date = ?
		AND status = 'pending'
		ORDER BY scheduled_time ASC
	`
	
	rows, err := r.db.Query(query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		err := rows.Scan(&campaign.ID, &campaign.UserID, &campaign.DeviceID,
			&campaign.Title, &campaign.Niche, &campaign.Message, &campaign.ImageURL,
			&campaign.ScheduledDate, &campaign.ScheduledTime, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt)
		if err != nil {
			continue
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
		SET title = ?, niche = ?, message = ?, image_url = ?,
		    scheduled_date = ?, scheduled_time = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, campaign.Title, campaign.Niche, campaign.Message,
		campaign.ImageURL, campaign.ScheduledDate, campaign.ScheduledTime,
		campaign.Status, campaign.UpdatedAt, campaign.ID)
		
	return err
}