package repository

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
)

var (
	campaignRepo     CampaignRepository
	campaignRepoOnce sync.Once
)

// GetCampaignRepository returns singleton instance of CampaignRepository
func GetCampaignRepository() CampaignRepository {
	campaignRepoOnce.Do(func() {
		campaignRepo = NewCampaignRepository(database.GetDB())
	})
	return campaignRepo
}

type CampaignRepository interface {
	CreateCampaign(campaign *models.Campaign) error
	GetCampaignByDateAndNiche(scheduledDate, niche string) ([]models.Campaign, error)
	GetAllCampaigns(userID string) ([]models.Campaign, error)
	GetCampaignByID(id int) (*models.Campaign, error)
	UpdateCampaignStatus(id int, status string) error
	GetPendingCampaigns() ([]models.Campaign, error)
	// Add new methods for lead status targeting
	GetPendingCampaignsByStatus(userID string, targetStatus string) ([]models.Campaign, error)
	// Additional methods needed by the app
	GetCampaigns(userID string) ([]models.Campaign, error)
	UpdateCampaign(campaign *models.Campaign) error
	DeleteCampaign(id int) error
	GetCampaignsByUser(userID string) ([]models.Campaign, error)
	// New methods for broadcast statistics
	GetCampaignBroadcastStats(campaignID int) (shouldSend, doneSend, failedSend int, err error)
	GetUserCampaignBroadcastStats(userID string) (shouldSend, doneSend, failedSend int, err error)
}

type campaignRepository struct {
	db *sql.DB
}

func NewCampaignRepository(db *sql.DB) CampaignRepository {
	return &campaignRepository{db: db}
}

// CreateCampaign creates a new campaign
func (r *campaignRepository) CreateCampaign(campaign *models.Campaign) error {
	// Set defaults
	if campaign.MinDelaySeconds == 0 {
		campaign.MinDelaySeconds = 10
	}
	if campaign.MaxDelaySeconds == 0 {
		campaign.MaxDelaySeconds = 30
	}
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	
	query := `
		INSERT INTO campaigns 
		(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 time_schedule, min_delay_seconds, max_delay_seconds, status, ai, "limit", created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`
	
	// Default target_status to 'all' if not set
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "all"
	}
	
	err := r.db.QueryRow(query, campaign.UserID, campaign.CampaignDate,
		campaign.Title, campaign.Niche, targetStatus, campaign.Message, campaign.ImageURL,
		campaign.TimeSchedule, campaign.MinDelaySeconds, campaign.MaxDelaySeconds, 
		campaign.Status, campaign.AI, campaign.Limit, campaign.CreatedAt, campaign.UpdatedAt).Scan(&campaign.ID)
		
	return err
}

// GetCampaignByDateAndNiche gets campaigns by date and niche
func (r *campaignRepository) GetCampaignByDateAndNiche(scheduledDate, niche string) ([]models.Campaign, error) {
	query := `
		SELECT id, user_id, title, niche, message, image_url, 
		       campaign_date, COALESCE(time_schedule, '09:00:00') as time_schedule, 
		       min_delay_seconds, max_delay_seconds, 
		       status, created_at, updated_at
		FROM campaigns
		WHERE campaign_date = $1 AND niche = $2
	`
	
	rows, err := r.db.Query(query, scheduledDate, niche)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Niche, 
			&c.Message, &c.ImageURL, &c.CampaignDate, &c.TimeSchedule, 
			&c.MinDelaySeconds, &c.MaxDelaySeconds,
			&c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	
	return campaigns, nil
}

// GetAllCampaigns gets all campaigns for a user
func (r *campaignRepository) GetAllCampaigns(userID string) ([]models.Campaign, error) {
	query := `
		SELECT 
			id, user_id, title, niche, 
			COALESCE(target_status, 'all') as target_status,
			message, image_url, campaign_date, 
			COALESCE(time_schedule, '') as time_schedule,
			COALESCE(min_delay_seconds, 10) as min_delay_seconds,
			COALESCE(max_delay_seconds, 30) as max_delay_seconds,
			status, ai, COALESCE("limit", 0) as limit, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1
		ORDER BY campaign_date DESC, time_schedule DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Niche, 
			&c.TargetStatus, &c.Message, &c.ImageURL, &c.CampaignDate, 
			&c.TimeSchedule, &c.MinDelaySeconds, &c.MaxDelaySeconds,
			&c.Status, &c.AI, &c.Limit, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	
	return campaigns, nil
}

// GetCampaignByID gets a campaign by ID
func (r *campaignRepository) GetCampaignByID(id int) (*models.Campaign, error) {
	query := `
		SELECT 
			id, user_id, title, niche, 
			COALESCE(target_status, 'all') as target_status,
			message, image_url, campaign_date, 
			COALESCE(time_schedule, '') as time_schedule,
			COALESCE(min_delay_seconds, 10) as min_delay_seconds,
			COALESCE(max_delay_seconds, 30) as max_delay_seconds,
			status, ai, COALESCE("limit", 0) as limit, created_at, updated_at
		FROM campaigns
		WHERE id = $1
	`
	
	var c models.Campaign
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.UserID, &c.Title, &c.Niche, 
		&c.TargetStatus, &c.Message, &c.ImageURL, &c.CampaignDate, 
		&c.TimeSchedule, &c.MinDelaySeconds, &c.MaxDelaySeconds,
		&c.Status, &c.AI, &c.Limit, &c.CreatedAt, &c.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &c, nil
}

// UpdateCampaignStatus updates campaign status
func (r *campaignRepository) UpdateCampaignStatus(id int, status string) error {
	query := `UPDATE campaigns SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(query, status, time.Now(), id)
	return err
}

// GetPendingCampaigns gets all campaigns with pending status
func (r *campaignRepository) GetPendingCampaigns() ([]models.Campaign, error) {
	// OPTIMIZED: Let PostgreSQL handle timezone conversions
	query := `
		SELECT 
			id, user_id, title, niche, 
			COALESCE(target_status, 'all') as target_status,
			message, image_url, campaign_date, 
			COALESCE(time_schedule, '') as time_schedule,
			COALESCE(min_delay_seconds, 10) as min_delay_seconds,
			COALESCE(max_delay_seconds, 30) as max_delay_seconds,
			status, created_at, updated_at
		FROM campaigns
		WHERE status = 'pending'
		AND id NOT IN (
			-- Exclude campaigns that already have broadcast messages
			SELECT DISTINCT campaign_id 
			FROM broadcast_messages 
			WHERE campaign_id IS NOT NULL
		)
		AND (
			-- Immediate execution (no time set)
			time_schedule IS NULL 
			OR time_schedule = ''
			-- Or scheduled time has passed (PostgreSQL handles timezone)
			OR (campaign_date || ' ' || time_schedule)::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur' <= CURRENT_TIMESTAMP
		)
		ORDER BY campaign_date, time_schedule
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("❌ [Campaign Repository] Error querying pending campaigns: %v", err)
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Niche, 
			&c.TargetStatus, &c.Message, &c.ImageURL, &c.CampaignDate, 
			&c.TimeSchedule, &c.MinDelaySeconds, &c.MaxDelaySeconds,
			&c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			log.Printf("❌ [Campaign Repository] Error scanning campaign: %v", err)
			continue
		}
		campaigns = append(campaigns, c)
	}
	
	return campaigns, nil
}

// GetPendingCampaignsByStatus gets pending campaigns filtered by target status
func (r *campaignRepository) GetPendingCampaignsByStatus(userID string, targetStatus string) ([]models.Campaign, error) {
	query := `
		SELECT 
			id, user_id, title, niche, 
			COALESCE(target_status, 'all') as target_status,
			message, image_url, campaign_date, 
			COALESCE(time_schedule, '') as time_schedule,
			COALESCE(min_delay_seconds, 10) as min_delay_seconds,
			COALESCE(max_delay_seconds, 30) as max_delay_seconds,
			status, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1 
		AND status = 'pending'
		AND (target_status = $2 OR target_status = 'all')
		AND (
			time_schedule IS NULL 
			OR time_schedule = ''
			OR (campaign_date || ' ' || time_schedule)::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur' <= CURRENT_TIMESTAMP
		)
		ORDER BY campaign_date, time_schedule
	`
	
	rows, err := r.db.Query(query, userID, targetStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Niche, 
			&c.TargetStatus, &c.Message, &c.ImageURL, &c.CampaignDate, 
			&c.TimeSchedule, &c.MinDelaySeconds, &c.MaxDelaySeconds,
			&c.Status, &c.AI, &c.Limit, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	
	return campaigns, nil
}

// GetCampaigns is an alias for GetAllCampaigns
func (r *campaignRepository) GetCampaigns(userID string) ([]models.Campaign, error) {
	return r.GetAllCampaigns(userID)
}

// GetCampaignsByUser is an alias for GetAllCampaigns
func (r *campaignRepository) GetCampaignsByUser(userID string) ([]models.Campaign, error) {
	return r.GetAllCampaigns(userID)
}

// UpdateCampaign updates an existing campaign
func (r *campaignRepository) UpdateCampaign(campaign *models.Campaign) error {
	query := `
		UPDATE campaigns 
		SET title = $1, niche = $2, target_status = $3, message = $4, 
		    image_url = $5, campaign_date = $6, time_schedule = $7,
		    min_delay_seconds = $8, max_delay_seconds = $9, 
		    status = $10, ai = $11, "limit" = $12, updated_at = $13
		WHERE id = $14 AND user_id = $15
	`
	
	campaign.UpdatedAt = time.Now()
	
	result, err := r.db.Exec(query, 
		campaign.Title, campaign.Niche, campaign.TargetStatus, campaign.Message,
		campaign.ImageURL, campaign.CampaignDate, campaign.TimeSchedule,
		campaign.MinDelaySeconds, campaign.MaxDelaySeconds,
		campaign.Status, campaign.AI, campaign.Limit, campaign.UpdatedAt, campaign.ID, campaign.UserID)
	
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// DeleteCampaign deletes a campaign by ID
func (r *campaignRepository) DeleteCampaign(id int) error {
	query := `DELETE FROM campaigns WHERE id = $1`
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// GetCampaignBroadcastStats gets broadcast statistics for a specific campaign
func (r *campaignRepository) GetCampaignBroadcastStats(campaignID int) (shouldSend, doneSend, failedSend int, err error) {
	// Get campaign details first
	campaign, err := r.GetCampaignByID(campaignID)
	if err != nil {
		return 0, 0, 0, err
	}
	
	// Get total leads that should receive the campaign based on target_status and niche
	shouldSendQuery := `
		SELECT COUNT(DISTINCT l.phone) 
		FROM leads l
		WHERE l.user_id = $1 
		AND l.niche = $2
		AND ($3 = 'all' OR l.target_status = $3)
	`
	
	err = r.db.QueryRow(shouldSendQuery, campaign.UserID, campaign.Niche, campaign.TargetStatus).Scan(&shouldSend)
	if err != nil {
		return 0, 0, 0, err
	}
	
	// Get done and failed counts from broadcast_messages
	statsQuery := `
		SELECT 
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as done_send,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_send
		FROM broadcast_messages
		WHERE campaign_id = $1
	`
	
	err = r.db.QueryRow(statsQuery, campaignID).Scan(&doneSend, &failedSend)
	if err != nil {
		return shouldSend, 0, 0, err
	}
	
	return shouldSend, doneSend, failedSend, nil
}

// GetUserCampaignBroadcastStats gets broadcast statistics for all campaigns of a user
func (r *campaignRepository) GetUserCampaignBroadcastStats(userID string) (shouldSend, doneSend, failedSend int, err error) {
	// Get all campaigns for the user
	campaigns, err := r.GetAllCampaigns(userID)
	if err != nil {
		return 0, 0, 0, err
	}
	
	totalShouldSend := 0
	totalDoneSend := 0
	totalFailedSend := 0
	
	// Calculate stats for each campaign
	for _, campaign := range campaigns {
		should, done, failed, err := r.GetCampaignBroadcastStats(campaign.ID)
		if err != nil {
			// Log error but continue with other campaigns
			log.Printf("Error getting stats for campaign %d: %v", campaign.ID, err)
			continue
		}
		
		totalShouldSend += should
		totalDoneSend += done
		totalFailedSend += failed
	}
	
	return totalShouldSend, totalDoneSend, totalFailedSend, nil
}
