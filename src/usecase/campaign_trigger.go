package usecase

import (
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// CampaignTriggerService handles campaign and sequence triggers
type CampaignTriggerService struct {
	broadcastManager *broadcast.BroadcastManager
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
	today := time.Now().Format("2006-01-02")
	
	// Get campaigns scheduled for today
	campaigns, err := campaignRepo.GetCampaignsByDate(today)
	if err != nil {
		return err
	}
	
	logrus.Infof("Found %d campaigns scheduled for today", len(campaigns))
	
	for _, campaign := range campaigns {
		// Check if already processed
		if campaign.Status == "sent" {
			continue
		}
		
		// Check if it's time to send
		scheduledTime, err := time.Parse("15:04", campaign.ScheduledTime)
		if err != nil {
			logrus.Errorf("Invalid scheduled time for campaign %s: %v", campaign.ID, err)
			continue
		}
		
		now := time.Now()
		campaignTime := time.Date(now.Year(), now.Month(), now.Day(), 
			scheduledTime.Hour(), scheduledTime.Minute(), 0, 0, now.Location())
		
		if now.After(campaignTime) {
			// Time to send this campaign
			go cts.executeCampaign(&campaign)
		}
	}
	
	return nil
}

// executeCampaign executes a campaign
func (cts *CampaignTriggerService) executeCampaign(campaign *models.Campaign) {
	logrus.Infof("Executing campaign: %s", campaign.Title)
	
	// Get leads matching the campaign niche
	leadRepo := repository.GetLeadRepository()
	leads, err := leadRepo.GetLeadsByNiche(campaign.Niche)
	if err != nil {
		logrus.Errorf("Failed to get leads for campaign %s: %v", campaign.ID, err)
		return
	}
	
	logrus.Infof("Found %d leads matching niche: %s", len(leads), campaign.Niche)
	
	// Get ALL connected devices for the user
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetUserDevices(campaign.UserID)
	if err != nil {
		logrus.Errorf("Failed to get devices for user %s: %v", campaign.UserID, err)
		return
	}
	
	// Filter only connected devices
	connectedDevices := make([]models.UserDevice, 0)
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