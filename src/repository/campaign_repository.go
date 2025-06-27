package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
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
	
	// Set default delay values if not provided
	if campaign.MinDelaySeconds == 0 {
		campaign.MinDelaySeconds = 10
	}
	if campaign.MaxDelaySeconds == 0 {
		campaign.MaxDelaySeconds = 30
	}
	
	query := `
		INSERT INTO campaigns 
		(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 scheduled_time, min_delay_seconds, max_delay_seconds, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	
	// Default target_status to 'all' if not set
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "all"
	}
	
	err := r.db.QueryRow(query, campaign.UserID, campaign.CampaignDate,
		campaign.Title, campaign.Niche, targetStatus, campaign.Message, campaign.ImageURL,
		campaign.ScheduledTime, campaign.MinDelaySeconds, campaign.MaxDelaySeconds, 
		campaign.Status, campaign.CreatedAt, campaign.UpdatedAt).Scan(&campaign.ID)
		
	return err
}

// GetCampaignByDateAndNiche gets campaigns by date and niche
func (r *campaignRepository) GetCampaignByDateAndNiche(scheduledDate, niche string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, niche, message, image_url, 
		       campaign_date, COALESCE(scheduled_time::text, '09:00:00') as scheduled_time, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
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
		
		err := rows.Scan(&campaign.ID, &campaign.UserID,
			&campaign.Title, &campaign.Niche, &campaign.Message, &campaign.ImageURL,
			&campaign.CampaignDate, &campaign.ScheduledTime, &campaign.MinDelaySeconds, 
			&campaign.MaxDelaySeconds, &campaign.Status,
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
		SET title = $1, niche = $2, message = $3, image_url = $4,
		    campaign_date = $5, scheduled_time = $6, min_delay_seconds = $7, 
		    max_delay_seconds = $8, status = $9, updated_at = $10
		WHERE id = $11
	`
	
	_, err := r.db.Exec(query, campaign.Title, campaign.Niche, campaign.Message,
		campaign.ImageURL, campaign.CampaignDate, campaign.ScheduledTime,
		campaign.MinDelaySeconds, campaign.MaxDelaySeconds,
		campaign.Status, campaign.UpdatedAt, campaign.ID)
		
	return err
}

// GetCampaigns gets all campaigns for a user
func (r *campaignRepository) GetCampaigns(userID string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, message, niche, image_url, 
		       campaign_date, COALESCE(scheduled_time::text, '09:00:00') as scheduled_time, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1
		ORDER BY campaign_date DESC, scheduled_time DESC
	`
	
	log.Printf("Getting campaigns for user: %s", userID)
	rows, err := r.db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying campaigns: %v", err)
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.ImageURL,
			&campaign.CampaignDate, &campaign.ScheduledTime, &campaign.MinDelaySeconds,
			&campaign.MaxDelaySeconds, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning campaign: %v", err)
			continue
		}
		
		log.Printf("Found campaign: ID=%d, Date=%s, Time=%s, Title=%s", 
			campaign.ID, campaign.CampaignDate, campaign.ScheduledTime, campaign.Title)
		campaigns = append(campaigns, campaign)
	}
	
	log.Printf("Total campaigns found: %d", len(campaigns))
	return campaigns, nil
}

// DeleteCampaign deletes a campaign
func (r *campaignRepository) DeleteCampaign(campaignID int) error {
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
		       campaign_date, COALESCE(scheduled_time::text, '09:00:00') as scheduled_time, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
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
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.ImageURL,
			&campaign.CampaignDate, &campaign.ScheduledTime, &campaign.MinDelaySeconds,
			&campaign.MaxDelaySeconds, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}

// GetCampaignsByUser gets all campaigns for a user
func (r *campaignRepository) GetCampaignsByUser(userID string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, message, niche, image_url, 
		       campaign_date, COALESCE(scheduled_time::text, '09:00:00') as scheduled_time, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.ImageURL,
			&campaign.CampaignDate, &campaign.ScheduledTime, &campaign.MinDelaySeconds,
			&campaign.MaxDelaySeconds, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	return campaigns, nil
}

// GetPendingCampaigns gets campaigns ready to be sent
func (r *campaignRepository) GetPendingCampaigns() ([]models.Campaign, error) {
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	currentTime := now.Format("15:04")
	
	query := `
		SELECT id, user_id, title, message, niche, image_url, 
		       campaign_date, COALESCE(scheduled_time::text, '09:00:00') as scheduled_time, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
		FROM campaigns
		WHERE status = 'scheduled' 
		  AND campaign_date <= $1
		  AND (scheduled_time IS NULL OR scheduled_time <= $2)
		ORDER BY campaign_date, scheduled_time
	`
	
	rows, err := r.db.Query(query, todayStr, currentTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var campaign models.Campaign
		
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.ImageURL,
			&campaign.CampaignDate, &campaign.ScheduledTime, &campaign.MinDelaySeconds,
			&campaign.MaxDelaySeconds, &campaign.Status,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning pending campaign: %v", err)
			continue
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	log.Printf("Found %d pending campaigns ready to send", len(campaigns))
	return campaigns, nil
}

// UpdateCampaignStatus updates campaign status
func (r *campaignRepository) UpdateCampaignStatus(campaignID int, status string) error {
	query := `UPDATE campaigns SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(query, status, time.Now(), campaignID)
	return err
}
