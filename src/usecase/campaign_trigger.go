package usecase

import (
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

var (
	timezoneWarningLogged = false
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
	// Only log when there are campaigns to process
	campaignRepo := repository.GetCampaignRepository()
	
	// Get pending campaigns that are ready to send
	campaigns, err := campaignRepo.GetPendingCampaigns()
	if err != nil {
		logrus.Errorf("Failed to get pending campaigns: %v", err)
		return err
	}
	
	if len(campaigns) > 0 {
		logrus.Infof("Found %d campaigns ready to process", len(campaigns))
	}
	
	for _, campaign := range campaigns {
		logrus.Infof("Checking campaign: %s (ID: %d, Status: %s, Date: %s, Time: %s)", 
			campaign.Title, campaign.ID, campaign.Status, campaign.CampaignDate, campaign.TimeSchedule)
		
		// Execute campaign
		go cts.executeCampaign(&campaign)
	}
	
	return nil
}

// executeCampaign executes a campaign
func (cts *CampaignTriggerService) executeCampaign(campaign *models.Campaign) {
	logrus.Infof("Executing campaign: %s", campaign.Title)
	
	// Get ALL connected devices for the user FIRST
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
		// Update campaign status to failed immediately
		campaignRepo := repository.GetCampaignRepository()
		if err := campaignRepo.UpdateCampaignStatus(campaign.ID, "failed"); err != nil {
			logrus.Errorf("Failed to update campaign status to failed: %v", err)
		}
		return
	}
	
	logrus.Infof("Using %d connected devices for campaign distribution", len(connectedDevices))
	
	// Check if campaign has target_status field
	targetStatus := campaign.TargetStatus
	if targetStatus == "" {
		targetStatus = "prospect" // Default to prospect if not set
	}
	
	// Get broadcast repository
	broadcastRepo := repository.GetBroadcastRepository()
	leadRepo := repository.GetLeadRepository()
	
	successful := 0
	failed := 0
	
	// Process leads for EACH device separately
	for _, device := range connectedDevices {
		// Get leads for THIS SPECIFIC DEVICE matching the campaign criteria
		leads, err := leadRepo.GetLeadsByDeviceNicheAndStatus(device.ID, campaign.Niche, targetStatus)
		if err != nil {
			logrus.Errorf("Failed to get leads for device %s: %v", device.ID, err)
			continue
		}
		
		if len(leads) == 0 {
			logrus.Debugf("No leads found for device %s matching niche: %s and status: %s", 
				device.ID, campaign.Niche, targetStatus)
			continue
		}
		
		logrus.Infof("Found %d leads for device %s (status: %s) matching criteria", 
			len(leads), device.ID, device.Status)
		
		// Queue messages for this device's leads
		for _, lead := range leads {
			// Create broadcast message
			msg := domainBroadcast.BroadcastMessage{
				UserID:         campaign.UserID,
				DeviceID:       device.ID, // Use the specific device that owns this lead
				CampaignID:     &campaign.ID,
				RecipientPhone: lead.Phone,
				Type:           "text", // Default to text, or determine from ImageURL
				Content:        campaign.Message,
				MediaURL:       campaign.ImageURL,
				ScheduledAt:    time.Now(),
				Status:         "pending",
				MinDelay:       campaign.MinDelaySeconds,
				MaxDelay:       campaign.MaxDelaySeconds,
			}
			
			// If image URL is provided, set type to image
			if campaign.ImageURL != "" {
				msg.Type = "image"
			}
			
			// Queue the message
			err := broadcastRepo.QueueMessage(msg)
			if err != nil {
				logrus.Errorf("Failed to queue message for %s: %v", lead.Phone, err)
				failed++
			} else {
				successful++
			}
		}
	}
	
	// Update campaign status based on results
	campaignRepo := repository.GetCampaignRepository()
	if successful == 0 && failed > 0 {
		// All messages failed - mark campaign as failed
		if err := campaignRepo.UpdateCampaignStatus(campaign.ID, "failed"); err != nil {
			logrus.Errorf("Failed to update campaign status to failed: %v", err)
		}
		logrus.Infof("Campaign %s marked as failed: 0 messages queued, %d failed", 
			campaign.Title, failed)
	} else if successful > 0 {
		// At least some messages queued - mark as triggered
		if err := campaignRepo.UpdateCampaignStatus(campaign.ID, "triggered"); err != nil {
			logrus.Errorf("Failed to update campaign status to triggered: %v", err)
		}
		logrus.Infof("Campaign %s triggered: %d messages queued across %d devices, %d failed", 
			campaign.Title, successful, len(connectedDevices), failed)
	} else {
		// No leads found - mark as completed
		if err := campaignRepo.UpdateCampaignStatus(campaign.ID, "completed"); err != nil {
			logrus.Errorf("Failed to update campaign status to completed: %v", err)
		}
		logrus.Infof("Campaign %s completed: No leads found matching criteria", campaign.Title)
	}
}

// ProcessSequenceTriggers processes new leads for sequence enrollment
func (cts *CampaignTriggerService) ProcessSequenceTriggers() error {
	// Only log when there's actual work to do
	
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

// ProcessDailySequenceMessages processes sequence messages for contacts
func (cts *CampaignTriggerService) ProcessDailySequenceMessages() error {
	// Only log when processing actual messages
	
	sequenceRepo := repository.GetSequenceRepository()
	broadcastRepo := repository.GetBroadcastRepository()
	userRepo := repository.GetUserRepository()
	
	// Get active sequences
	sequences, err := sequenceRepo.GetActiveSequencesWithNiche()
	if err != nil {
		return err
	}
	
	for _, sequence := range sequences {
		// Get all active contacts in this sequence
		contacts, err := sequenceRepo.GetSequenceContacts(sequence.ID)
		if err != nil {
			logrus.Errorf("Failed to get contacts for sequence %s: %v", sequence.ID, err)
			continue
		}
		
		for _, contact := range contacts {
			// Check if 24 hours have passed since last message
			if contact.LastMessageAt != nil {
				timeSince := time.Since(*contact.LastMessageAt)
				if timeSince < 24*time.Hour {
					continue // Not time yet
				}
			}
			
			// Get the next step for this contact
			nextDay := contact.CurrentDay + 1
			if nextDay > sequence.TotalDays {
				// Contact has completed the sequence
				sequenceRepo.MarkContactCompleted(contact.ID)
				continue
			}
			
			// Get step for the next day
			steps, err := sequenceRepo.GetSequenceSteps(sequence.ID)
			if err != nil {
				continue
			}
			
			var nextStep *models.SequenceStep
			for _, step := range steps {
				if step.Day == nextDay {
					nextStep = &step
					break
				}
			}
			
			if nextStep == nil {
				continue // No step found for this day
			}
			
			// Get the lead's assigned device from database
			db := database.GetDB()
			var leadDeviceID string
			err = db.QueryRow(`
				SELECT device_id FROM leads 
				WHERE phone = $1 AND user_id = $2
				LIMIT 1
			`, contact.ContactPhone, sequence.UserID).Scan(&leadDeviceID)
			
			if err != nil {
				logrus.Errorf("Failed to get device for contact %s: %v", contact.ContactPhone, err)
				continue
			}
			
			// Check if the lead's device is connected
			devices, err := userRepo.GetUserDevices(sequence.UserID)
			if err != nil {
				continue
			}
			
			// Find the lead's specific device
			var device *models.UserDevice
			for _, d := range devices {
				if d.ID == leadDeviceID && (d.Status == "connected" || d.Status == "online") {
					device = d
					break
				}
			}
			
			if device == nil {
				logrus.Warnf("Lead's device %s not connected for sequence %s, skipping contact %s", 
					leadDeviceID, sequence.Name, contact.ContactPhone)
				// Don't advance the sequence if the lead's device is not available
				continue
			}
			
			// Check if we already created a message for this contact/day
			var existingCount int
			err = db.QueryRow(`
				SELECT COUNT(*) FROM broadcast_messages 
				WHERE sequence_id = $1 
				AND recipient_phone = $2 
				AND group_order = $3
			`, sequence.ID, contact.ContactPhone, nextDay).Scan(&existingCount)
			
			if err == nil && existingCount > 0 {
				logrus.Infof("Message already exists for contact %s day %d, skipping", contact.ContactPhone, nextDay)
				continue
			}
			
			// Queue the message
			msg := domainBroadcast.BroadcastMessage{
				UserID:         sequence.UserID,
				DeviceID:       device.ID,
				SequenceID:     &sequence.ID,
				RecipientPhone: contact.ContactPhone,
				Type:           nextStep.MessageType,
				Content:        nextStep.Content,
				MediaURL:       nextStep.MediaURL,
				ScheduledAt:    time.Now(),
				MinDelay:       5,  // Default delays for sequences
				MaxDelay:       15,
			}
			
			err = broadcastRepo.QueueMessage(msg)
			if err != nil {
				logrus.Errorf("Failed to queue sequence message: %v", err)
				// Don't update progress if we couldn't queue the message
			} else {
				// Update contact progress only after successfully queuing
				err = sequenceRepo.UpdateContactProgress(contact.ID, nextDay, "active")
				if err != nil {
					logrus.Errorf("Failed to update contact progress: %v", err)
				}
				
				// Log the message
				log := &models.SequenceLog{
					SequenceID: sequence.ID,
					ContactID:  contact.ID,
					StepID:     nextStep.ID,
					Day:        nextDay,
					Status:     "queued", // Not "sent" yet
					SentAt:     time.Now(),
				}
				sequenceRepo.CreateSequenceLog(log)
				
				logrus.Infof("Queued sequence message for %s (day %d of %s)", 
					contact.ContactPhone, nextDay, sequence.Name)
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
			
			// Process daily sequence messages
			if err := cts.ProcessDailySequenceMessages(); err != nil {
				logrus.Errorf("Error processing daily sequence messages: %v", err)
			}
		}
	}
}