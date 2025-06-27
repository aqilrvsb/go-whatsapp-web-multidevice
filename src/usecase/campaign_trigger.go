package usecase

import (
	"fmt"
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
	
	// Get both today and tomorrow to handle timezone differences
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	
	// Get campaigns for both dates
	campaignsToday, err := campaignRepo.GetCampaignsByDate(today)
	if err != nil {
		return err
	}
	
	campaignsTomorrow, err := campaignRepo.GetCampaignsByDate(tomorrow)
	if err != nil {
		return err
	}
	
	// Combine campaigns
	campaigns := append(campaignsToday, campaignsTomorrow...)
	
	logrus.Infof("Found %d campaigns scheduled for today/tomorrow", len(campaigns))
	
	for _, campaign := range campaigns {
		// Check if already processed
		if campaign.Status == "sent" {
			continue
		}
		
		// Check if it's time to send
		if campaign.ScheduledTime == "" {
			// If no scheduled time, send immediately
			logrus.Infof("Campaign %d has no scheduled time, sending now", campaign.ID)
			go cts.executeCampaign(&campaign)
		} else {
			// Parse the scheduled time
			now := time.Now()
			scheduledTimeStr := fmt.Sprintf("%s %s:00", campaign.CampaignDate, campaign.ScheduledTime)
			scheduledTime, err := time.Parse("2006-01-02 15:04:05", scheduledTimeStr)
			if err != nil {
				logrus.Errorf("Failed to parse scheduled time for campaign %d: %v", campaign.ID, err)
				continue
			}
			
			if now.After(scheduledTime) {
				// Time to send this campaign
				logrus.Infof("Campaign %d scheduled time reached, sending now", campaign.ID)
				go cts.executeCampaign(&campaign)
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