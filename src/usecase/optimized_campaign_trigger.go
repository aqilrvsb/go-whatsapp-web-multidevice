package usecase

import (
	"database/sql"
	"fmt"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// OptimizedCampaignTrigger uses TIMESTAMPTZ for proper timezone handling
type OptimizedCampaignTrigger struct {
	broadcastManager broadcast.BroadcastManagerInterface
	db               *sql.DB
}

// NewOptimizedCampaignTrigger creates an optimized trigger service
func NewOptimizedCampaignTrigger(db *sql.DB) *OptimizedCampaignTrigger {
	return &OptimizedCampaignTrigger{
		broadcastManager: broadcast.GetBroadcastManager(),
		db:               db,
	}
}

// ProcessCampaigns uses TIMESTAMPTZ for timezone-aware campaign processing
func (oct *OptimizedCampaignTrigger) ProcessCampaigns() error {
	logrus.Info("Processing campaigns with optimized timezone handling...")
	
	// Query campaigns that are ready to send using TIMESTAMPTZ
	query := `
		SELECT 
			c.id, c.user_id, c.title, c.message, c.niche, 
			COALESCE(c.target_status, 'all') as target_status, 
			c.image_url, c.min_delay_seconds, c.max_delay_seconds,
			c.campaign_date, c.time_schedule
		FROM campaigns c
		WHERE c.status = 'pending'
		AND (
			-- If scheduled_at exists, use it
			(c.scheduled_at IS NOT NULL AND c.scheduled_at <= CURRENT_TIMESTAMP)
			OR
			-- Fallback to old columns
			(c.scheduled_at IS NULL AND 
			 (c.campaign_date || ' ' || COALESCE(c.time_schedule, '00:00:00'))::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur' <= CURRENT_TIMESTAMP)
		)
		ORDER BY COALESCE(c.scheduled_at, (c.campaign_date || ' ' || COALESCE(c.time_schedule, '00:00:00'))::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur')
	`
	
	rows, err := oct.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query campaigns: %v", err)
	}
	defer rows.Close()
	
	campaignCount := 0
	for rows.Next() {
		var campaign models.Campaign
		err := rows.Scan(
			&campaign.ID, &campaign.UserID, &campaign.Title, &campaign.Message,
			&campaign.Niche, &campaign.TargetStatus, &campaign.ImageURL,
			&campaign.MinDelaySeconds, &campaign.MaxDelaySeconds,
			&campaign.CampaignDate, &campaign.TimeSchedule,
		)
		if err != nil {
			logrus.Errorf("Failed to scan campaign: %v", err)
			continue
		}
		
		campaignCount++
		logrus.Infof("Processing campaign: %s (ID: %d)", campaign.Title, campaign.ID)
		
		// Execute campaign in goroutine
		go oct.executeCampaign(&campaign)
	}
	
	logrus.Infof("Found and triggered %d campaigns", campaignCount)
	return nil
}

// executeCampaign remains the same as original
func (oct *OptimizedCampaignTrigger) executeCampaign(campaign *models.Campaign) {
	logrus.Infof("Executing campaign: %s", campaign.Title)
	
	// Get leads matching the campaign niche AND status
	leadRepo := repository.GetLeadRepository()
	
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "prospect"
	}
	
	leads, err := leadRepo.GetLeadsByNicheAndStatus(campaign.Niche, targetStatus)
	if err != nil {
		logrus.Errorf("Failed to get leads for campaign %d: %v", campaign.ID, err)
		return
	}
	
	logrus.Infof("Found %d leads matching niche: %s and status: %s", len(leads), campaign.Niche, targetStatus)
	
	// Debug: Log first few leads if any
	if len(leads) > 0 {
		for i, lead := range leads {
			if i < 3 { // Log first 3 leads
				logrus.Debugf("Lead %d: Name=%s, Phone=%s, Niche=%s, Status=%s", 
					i+1, lead.Name, lead.Phone, lead.Niche, lead.TargetStatus)
			}
		}
	} else {
		logrus.Warnf("No leads found for niche '%s' with status '%s'", campaign.Niche, targetStatus)
	}
	
	// Get ALL connected devices for the user
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetUserDevices(campaign.UserID)
	if err != nil {
		logrus.Errorf("Failed to get devices for user %s: %v", campaign.UserID, err)
		return
	}
	
	// Filter only connected devices
	connectedDevices := make([]*models.UserDevice, 0)
	for _, device := range devices {
		// Check for connected, Connected, online, or Online status
		if device.Status == "connected" || device.Status == "Connected" || 
		   device.Status == "online" || device.Status == "Online" {
			connectedDevices = append(connectedDevices, device)
		}
	}
	
	if len(connectedDevices) == 0 {
		logrus.Errorf("No connected devices found for user %s", campaign.UserID)
		return
	}
	
	logrus.Infof("Using %d connected devices for campaign distribution", len(connectedDevices))
	
	// Queue messages for each lead
	broadcastRepo := repository.GetBroadcastRepository()
	successful := 0
	failed := 0
	deviceIndex := 0
	
	for _, lead := range leads {
		// Round-robin device selection
		device := connectedDevices[deviceIndex%len(connectedDevices)]
		deviceIndex++
		
		msg := domainBroadcast.BroadcastMessage{
			UserID:         campaign.UserID,
			DeviceID:       device.ID,
			CampaignID:     &campaign.ID,
			RecipientPhone: lead.Phone,
			Type:           "text",
			Content:        campaign.Message,
			MediaURL:       campaign.ImageURL,
			ScheduledAt:    time.Now(),
			MinDelay:       campaign.MinDelaySeconds,
			MaxDelay:       campaign.MaxDelaySeconds,
		}
		
		err := broadcastRepo.QueueMessage(msg)
		if err != nil {
			logrus.Errorf("Failed to queue message for %s: %v", lead.Phone, err)
			failed++
		} else {
			successful++
		}
	}
	
	// Update campaign status
	_, err = oct.db.Exec("UPDATE campaigns SET status = 'sent', updated_at = CURRENT_TIMESTAMP WHERE id = $1", campaign.ID)
	if err != nil {
		logrus.Errorf("Failed to update campaign status: %v", err)
	}
	
	logrus.Infof("Campaign %s completed: %d messages queued, %d failed", 
		campaign.Title, successful, failed)
}