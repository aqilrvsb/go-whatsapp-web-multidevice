package usecase

import (
	"fmt"
	"strings"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// CampaignTriggerService handles campaign and sequence triggers
type CampaignTriggerService struct {
	broadcastManager broadcast.BroadcastManagerInterface
}

// NewCampaignTriggerService creates new trigger service
func NewCampaignTriggerService() *CampaignTriggerService {
	return &CampaignTriggerService{
		broadcastManager: broadcast.GetBroadcastManager(),
	}
}

// ProcessCampaignTriggers processes campaigns scheduled for today
func (cts *CampaignTriggerService) ProcessCampaignTriggers() error {
	logrus.Info("Processing campaign triggers...")
	
	campaignRepo := repository.GetCampaignRepository()
	
	// Use fixed UTC+8 offset for Malaysia if timezone loading fails
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		logrus.Warnf("Failed to load Malaysia timezone, using fixed UTC+8: %v", err)
		loc = time.FixedZone("UTC+8", 8*60*60) // 8 hours ahead of UTC
	}
	
	// Get current time in Malaysia
	nowMalaysia := time.Now().In(loc)
	todayMalaysia := nowMalaysia.Format("2006-01-02")
	
	// Also get UTC time for comparison
	nowUTC := time.Now().UTC()
	todayUTC := nowUTC.Format("2006-01-02")
	
	logrus.Infof("Server UTC time: %s, Malaysia time: %s", nowUTC.Format("2006-01-02 15:04:05"), nowMalaysia.Format("2006-01-02 15:04:05"))
	
	// Get campaigns for both UTC and Malaysia dates to handle timezone differences
	var campaigns []models.Campaign
	campaignsMap := make(map[int]bool) // To avoid duplicates
	
	// Check UTC date and surrounding dates
	dates := []string{
		nowUTC.Add(-24 * time.Hour).Format("2006-01-02"),
		todayUTC,
		nowUTC.Add(24 * time.Hour).Format("2006-01-02"),
		todayMalaysia, // Also check Malaysia date
	}
	
	for _, date := range dates {
		dateCampaigns, err := campaignRepo.GetCampaignsByDate(date)
		if err == nil {
			for _, c := range dateCampaigns {
				if _, exists := campaignsMap[c.ID]; !exists {
					campaigns = append(campaigns, c)
					campaignsMap[c.ID] = true
				}
			}
		}
	}
	
	logrus.Infof("Found %d unique campaigns (checking dates: %v)", len(campaigns), dates)
	
	for _, campaign := range campaigns {
		logrus.Infof("Checking campaign: %s (ID: %d, Status: %s, ScheduledTime: '%s')", 
			campaign.Title, campaign.ID, campaign.Status, campaign.ScheduledTime)
		
		// Check if already processed
		if campaign.Status == "sent" {
			logrus.Infof("Campaign %d already sent, skipping", campaign.ID)
			continue
		}
		
		// Check if it's time to send
		scheduledTimeStr := strings.TrimSpace(campaign.ScheduledTime)
		if scheduledTimeStr == "" || scheduledTimeStr == "00:00:00" || strings.Contains(scheduledTimeStr, "0001-01-01") {
			// If no scheduled time, midnight, or default date, send immediately
			logrus.Infof("Campaign %d has no/default scheduled time ('%s'), sending now", campaign.ID, campaign.ScheduledTime)
			go cts.executeCampaign(&campaign)
		} else {
			// Parse the scheduled time
			// Extract just the date part from CampaignDate (in case it has timestamp)
			campaignDateOnly := campaign.CampaignDate
			if len(campaignDateOnly) > 10 {
				campaignDateOnly = campaignDateOnly[:10] // Take only YYYY-MM-DD
			}
			
			// Extract just the time part from ScheduledTime
			scheduledTimeOnly := campaign.ScheduledTime
			if len(scheduledTimeOnly) > 8 {
				scheduledTimeOnly = scheduledTimeOnly[:8] // Take only HH:MM:SS
			}
			
			scheduledTimeStr := fmt.Sprintf("%s %s", campaignDateOnly, scheduledTimeOnly)
			scheduledTime, err := time.Parse("2006-01-02 15:04:05", scheduledTimeStr)
			if err != nil {
				logrus.Errorf("Failed to parse scheduled time for campaign %d: %v (tried to parse: %s)", campaign.ID, err, scheduledTimeStr)
				continue
			}
			
			// Use Malaysia timezone for comparison
			loc := time.FixedZone("UTC+8", 8*60*60)
			nowMalaysia := time.Now().In(loc)
			
			// The scheduled time is assumed to be in Malaysia time
			scheduledMalaysia := time.Date(
				scheduledTime.Year(), scheduledTime.Month(), scheduledTime.Day(),
				scheduledTime.Hour(), scheduledTime.Minute(), scheduledTime.Second(), 0, loc)
			
			logrus.Infof("Campaign %d: Now Malaysia: %s, Scheduled: %s", 
				campaign.ID, 
				nowMalaysia.Format("2006-01-02 15:04:05"), 
				scheduledMalaysia.Format("2006-01-02 15:04:05"))
			
			if nowMalaysia.After(scheduledMalaysia) || nowMalaysia.Equal(scheduledMalaysia) {
				// Time to send this campaign
				logrus.Infof("Campaign %d scheduled time reached, sending now", campaign.ID)
				go cts.executeCampaign(&campaign)
			} else {
				timeDiff := scheduledMalaysia.Sub(nowMalaysia)
				logrus.Infof("Campaign %d not yet time, will trigger in %v", campaign.ID, timeDiff)
			}
		}
	}
	
	return nil
}

// executeCampaign executes a campaign
func (cts *CampaignTriggerService) executeCampaign(campaign *models.Campaign) {
	logrus.Infof("Executing campaign: %s", campaign.Title)
	
	// Get leads matching the campaign niche AND status
	leadRepo := repository.GetLeadRepository()
	var leads []models.Lead
	var err error
	
	// Check if campaign has target_status field
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "prospect" // Default to prospect if not set
	}
	
	leads, err = leadRepo.GetLeadsByNicheAndStatus(campaign.Niche, targetStatus)
	if err != nil {
		logrus.Errorf("Failed to get leads for campaign %s: %v", campaign.ID, err)
		return
	}
	
	logrus.Infof("Found %d leads matching niche: %s and status: %s", len(leads), campaign.Niche, targetStatus)
	
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
		if device.Status == "connected" {
			connectedDevices = append(connectedDevices, device)
		}
	}
	
	if len(connectedDevices) == 0 {
		logrus.Errorf("No connected devices found for user %s", campaign.UserID)
		return
	}
	
	logrus.Infof("Using %d connected devices for campaign distribution", len(connectedDevices))
	
	// Get broadcast repository
	broadcastRepo := repository.GetBroadcastRepository()
	
	// Queue messages for each lead, distributing across devices
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
	campaignRepo := repository.GetCampaignRepository()
	campaign.Status = "sent"
	campaignRepo.UpdateCampaign(campaign)
	
	logrus.Infof("Campaign %s completed: %d messages queued across %d devices, %d failed", 
		campaign.Title, successful, len(connectedDevices), failed)
}
// ProcessSequenceTriggers processes new leads for sequence enrollment
func (cts *CampaignTriggerService) ProcessSequenceTriggers() error {
	logrus.Info("Processing sequence triggers for new leads...")
	
	// Load Malaysia timezone for consistent processing
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		logrus.Warnf("Failed to load Malaysia timezone for sequences, using UTC: %v", err)
		loc = time.UTC
	}
	
	nowMalaysia := time.Now().In(loc)
	logrus.Infof("Processing sequences at Malaysia time: %s", nowMalaysia.Format("2006-01-02 15:04:05"))
	
	sequenceRepo := repository.GetSequenceRepository()
	leadRepo := repository.GetLeadRepository()
	
	// Get all active sequences with niche
	sequences, err := sequenceRepo.GetActiveSequencesWithNiche()
	if err != nil {
		return err
	}
	
	for _, sequence := range sequences {
		if sequence.Niche == "" {
			continue
		}
		
		// Get new leads matching this niche that aren't in sequence yet
		newLeads, err := leadRepo.GetNewLeadsForSequence(sequence.Niche, sequence.ID)
		if err != nil {
			logrus.Errorf("Failed to get new leads for sequence %s: %v", sequence.ID, err)
			continue
		}
		
		if len(newLeads) > 0 {
			logrus.Infof("Found %d new leads for sequence %s (niche: %s)", 
				len(newLeads), sequence.Name, sequence.Niche)
			
			// Add leads to sequence
			for _, lead := range newLeads {
				contact := &models.SequenceContact{
					SequenceID:   sequence.ID,
					ContactPhone: lead.Phone,
					ContactName:  lead.Name,
				}
				
				err := sequenceRepo.AddContactToSequence(contact)
				if err != nil {
					logrus.Errorf("Failed to add lead %s to sequence: %v", lead.Phone, err)
				} else {
					logrus.Infof("Added lead %s to sequence %s", lead.Phone, sequence.Name)
				}
			}
		}
	}
	
	return nil
}

// StartTriggerProcessor starts the background processor for triggers
func StartTriggerProcessor() {
	cts := NewCampaignTriggerService()
	
	// Process every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Process campaign triggers
			if err := cts.ProcessCampaignTriggers(); err != nil {
				logrus.Errorf("Error processing campaign triggers: %v", err)
			}
			
			// Process sequence triggers for new leads
			if err := cts.ProcessSequenceTriggers(); err != nil {
				logrus.Errorf("Error processing sequence triggers: %v", err)
			}
		}
	}
}