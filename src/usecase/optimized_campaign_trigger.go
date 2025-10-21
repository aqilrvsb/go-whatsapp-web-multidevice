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
	// Process campaigns with timezone handling
	
	// Query campaigns that are ready to send - Fixed like sequences (no CONVERT_TZ)
	query := `
		SELECT c.id, c.user_id, c.title, c.message, c.niche, 
			COALESCE(c.target_status, 'all') AS target_status, 
			COALESCE(c.image_url, '') AS image_url, c.min_delay_seconds, c.max_delay_seconds,
			c.campaign_date, c.time_schedule
		FROM campaigns c
		WHERE c.status = 'pending'
		AND (
			-- If scheduled_at exists, use it
			(c.scheduled_at IS NOT NULL AND c.scheduled_at <= NOW())
			OR
			-- Fallback to old columns - Simple comparison without CONVERT_TZ
			(c.scheduled_at IS NULL AND 
			 STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
		)
		ORDER BY COALESCE(c.scheduled_at, STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s'))
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
	
	if campaignCount > 0 {
		logrus.Infof("Found and triggered %d campaigns", campaignCount)
	}
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
		// Platform devices are always treated as online
		if device.Platform != "" {
			connectedDevices = append(connectedDevices, device)
			logrus.Debugf("Including platform device %s as online", device.DeviceName)
			continue
		}
		
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
	
	// Get leads from ALL connected devices
	allLeads := []models.Lead{}
	for _, device := range connectedDevices {
		deviceLeads, err := leadRepo.GetLeadsByDeviceNicheAndStatus(device.ID, campaign.Niche, targetStatus)
		if err != nil {
			logrus.Errorf("Failed to get leads for device %s: %v", device.ID, err)
			continue
		}
		if len(deviceLeads) > 0 {
			logrus.Infof("Found %d leads for device %s", len(deviceLeads), device.ID)
			allLeads = append(allLeads, deviceLeads...)
		}
	}
	
	leads := allLeads
	
	logrus.Infof("Total: Found %d leads matching niche: %s and status: %s across all devices", 
		len(leads), campaign.Niche, targetStatus)
	
	// Queue messages for each lead
	broadcastRepo := repository.GetBroadcastRepository()
	successful := 0
	failed := 0
	
	for _, lead := range leads {
		// Check if message already exists for this campaign and phone
		var existingCount int
		checkQuery := `
			SELECT COUNT(*) FROM broadcast_messages 
			WHERE campaign_id = ? 
			AND recipient_phone = ? 
			AND status IN ('pending', 'processing', 'queued', 'sent')
		`
		err := oct.db.QueryRow(checkQuery, campaign.ID, lead.Phone).Scan(&existingCount)
		
		if err == nil && existingCount > 0 {
			logrus.Debugf("Message already exists for campaign %d and phone %s, skipping", campaign.ID, lead.Phone)
			continue // Skip this lead
		}
		
		// Use the device that owns this lead
		msg := domainBroadcast.BroadcastMessage{
			UserID:         campaign.UserID,
			DeviceID:       lead.DeviceName, // Use device_name for message sending
			DeviceName:     lead.DeviceName,
			CampaignID:     &campaign.ID,
			RecipientPhone: lead.Phone,
			RecipientName:  lead.Name,
			Type:           "text",
			Content:        campaign.Message,
			MediaURL:       campaign.ImageURL,
			ScheduledAt:    time.Now(),
			// MinDelay and MaxDelay removed - will be fetched from campaigns table during processing
		}
		
		err = broadcastRepo.QueueMessage(msg)
		if err != nil {
			logrus.Errorf("Failed to queue message for %s: %v", lead.Phone, err)
			failed++
		} else {
			successful++
		}
	}
	
	// Update campaign status to triggered after queueing
	if successful > 0 {
		// Only mark as triggered if we actually queued some messages
		_, err = oct.db.Exec("UPDATE campaigns SET status = 'triggered', updated_at = CURRENT_TIMESTAMP WHERE id = ?", campaign.ID)
		if err != nil {
			logrus.Errorf("Failed to update campaign status to triggered: %v", err)
		}
		logrus.Infof("Campaign %s triggered: %d messages queued, %d failed", 
			campaign.Title, successful, failed)
	} else {
		// No messages queued, mark as finished
		_, err = oct.db.Exec("UPDATE campaigns SET status = 'finished', updated_at = CURRENT_TIMESTAMP WHERE id = ?", campaign.ID)
		if err != nil {
			logrus.Errorf("Failed to update campaign status to finished: %v", err)
		}
		logrus.Infof("Campaign %s finished: No matching leads found", campaign.Title)
	}
}