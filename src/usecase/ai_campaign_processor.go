package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/go-redis/redis/v8"
)

type DeviceTracker struct {
	DeviceID string
	Sent     int
	Failed   int
	Limit    int
	Status   string // active, limit_reached, failed
}

type AICampaignProcessor struct {
	broadcastRepo *repository.BroadcastRepository
	leadAIRepo    repository.LeadAIRepository
	userRepo      *repository.UserRepository
	campaignRepo  repository.CampaignRepository
	redisClient   *redis.Client
}
func NewAICampaignProcessor(
	broadcastRepo *repository.BroadcastRepository,
	leadAIRepo repository.LeadAIRepository,
	userRepo *repository.UserRepository,
	campaignRepo repository.CampaignRepository,
	redisClient *redis.Client,
) *AICampaignProcessor {
	return &AICampaignProcessor{
		broadcastRepo: broadcastRepo,
		leadAIRepo:    leadAIRepo,
		userRepo:      userRepo,
		campaignRepo:  campaignRepo,
		redisClient:   redisClient,
	}
}

func (p *AICampaignProcessor) ProcessAICampaign(ctx context.Context, campaignID int) error {
	// 1. Get campaign details
	campaign, err := p.campaignRepo.GetCampaignByID(campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}
	
	// Verify this is an AI campaign
	if campaign.AI == nil || *campaign.AI != "ai" {
		return fmt.Errorf("campaign %d is not an AI campaign", campaignID)
	}
	
	// 2. Get all pending AI leads based on campaign criteria
	var leads []models.LeadAI
	if campaign.TargetStatus == "all" {
		leads, err = p.leadAIRepo.GetLeadAIByNiche(campaign.UserID, campaign.Niche)
	} else {
		leads, err = p.leadAIRepo.GetLeadAIByNicheAndStatus(campaign.UserID, campaign.Niche, campaign.TargetStatus)
	}
	
	if err != nil {
		return fmt.Errorf("failed to get AI leads: %w", err)
	}	
	// Filter only pending leads
	var pendingLeads []models.LeadAI
	for _, lead := range leads {
		if lead.Status == "pending" {
			pendingLeads = append(pendingLeads, lead)
		}
	}
	
	if len(pendingLeads) == 0 {
		return nil
	}
	
	// 3. Get all connected devices for the user
	devices, err := p.userRepo.GetUserDevices(campaign.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user devices: %w", err)
	}
	
	// Filter only connected devices
	var connectedDevices []*models.UserDevice
	for _, device := range devices {
		if device.Status == "online" {
			connectedDevices = append(connectedDevices, device)
		}
	}
	
	if len(connectedDevices) == 0 {
		// Mark campaign as failed
		p.campaignRepo.UpdateCampaignStatus(campaignID, "failed")
		return fmt.Errorf("no connected devices available")
	}
	
	// 4. Initialize device tracking
	deviceStatus := make(map[string]*DeviceTracker)
	for _, device := range connectedDevices {
		deviceStatus[device.ID] = &DeviceTracker{
			DeviceID: device.ID,
			Sent:     0,
			Failed:   0,
			Limit:    campaign.Limit,
			Status:   "active",
		}
		
		// Initialize campaign progress in database
		progress := &models.AICampaignProgress{
			CampaignID: campaignID,
			DeviceID:   device.ID,
			LeadsSent:  0,
			LeadsFailed: 0,
			Status:     "active",
		}
		p.leadAIRepo.UpdateCampaignProgress(progress)
	}	
	// 5. Start round-robin assignment
	deviceIndex := 0
	successCount := 0
	failCount := 0
	
	// Create a wait group for concurrent processing
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent sends
	
	for _, lead := range pendingLeads {
		assigned := false
		attempts := 0
		
		// Try to assign to a device
		for attempts < len(connectedDevices) && !assigned {
			device := connectedDevices[deviceIndex%len(connectedDevices)]
			tracker := deviceStatus[device.ID]
			
			// Check if device can accept more leads
			if tracker.Status == "active" && tracker.Sent < tracker.Limit {
				// Assign lead to device
				wg.Add(1)
				semaphore <- struct{}{} // Acquire semaphore
				
				go func(l models.LeadAI, d *models.UserDevice, t *DeviceTracker) {
					defer wg.Done()
					defer func() { <-semaphore }() // Release semaphore
					
					err := p.assignAndSendLead(ctx, l, d, campaign)
					if err != nil {
						t.Failed++
						failCount++
						
						// Mark device as failed after 3 consecutive failures
						if t.Failed > 3 {
							t.Status = "failed"
							p.updateDeviceProgress(campaignID, d.ID, t)
						}
						
						// Update lead status to failed
						p.leadAIRepo.UpdateStatus(l.ID, "failed")
					} else {
						t.Sent++
						successCount++
						assigned = true
						
						// Update lead with device assignment
						p.leadAIRepo.AssignDevice(l.ID, d.ID)
						p.leadAIRepo.UpdateStatus(l.ID, "sent")
						
						// Check if device reached limit
						if t.Sent >= t.Limit {
							t.Status = "limit_reached"
						}
						
						// Update progress in database
						p.updateDeviceProgress(campaignID, d.ID, t)
					}
				}(lead, device, tracker)
				
				assigned = true // Mark as assigned to prevent multiple attempts
			}
			
			deviceIndex++
			attempts++
		}		
		// If not assigned after all attempts, mark as failed
		if !assigned {
			p.leadAIRepo.UpdateStatus(lead.ID, "failed")
			failCount++
		}
		
		// Add a small delay between lead assignments
		time.Sleep(time.Duration(rand.Intn(500)+500) * time.Millisecond)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// 6. Update campaign status
	var campaignStatus string
	if successCount > 0 {
		if failCount > 0 {
			campaignStatus = "completed_with_errors"
		} else {
			campaignStatus = "completed"
		}
	} else {
		campaignStatus = "failed"
	}
	
	p.campaignRepo.UpdateCampaignStatus(campaignID, campaignStatus)
	
	return nil
}

func (p *AICampaignProcessor) assignAndSendLead(ctx context.Context, lead models.LeadAI, device *models.UserDevice, campaign *models.Campaign) error {
	// Create broadcast message
	message := domainBroadcast.BroadcastMessage{
		ID:             fmt.Sprintf("ai_%d_%d_%s", campaign.ID, lead.ID, time.Now().Format("20060102150405")),
		UserID:         campaign.UserID,
		DeviceID:       device.ID,
		CampaignID:     &campaign.ID,
		RecipientPhone: lead.Phone,
		RecipientJID:   lead.Phone + "@s.whatsapp.net",
		Type:           "text",
		Content:        campaign.Message,
		Message:        campaign.Message,
		MediaURL:       campaign.ImageURL,
		ImageURL:       campaign.ImageURL,
		Status:         "queued",
		CreatedAt:      time.Now(),
		MinDelay:       campaign.MinDelaySeconds,
		MaxDelay:       campaign.MaxDelaySeconds,
	}
	
	// Queue message to database
	err := p.broadcastRepo.QueueMessage(message)
	if err != nil {
		return fmt.Errorf("failed to queue broadcast message: %w", err)
	}
	
	// Add delay between messages (human-like behavior)
	minDelay := campaign.MinDelaySeconds
	maxDelay := campaign.MaxDelaySeconds
	if minDelay < 5 {
		minDelay = 5
	}
	if maxDelay < minDelay {
		maxDelay = minDelay + 10
	}
	
	delay := rand.Intn(maxDelay-minDelay) + minDelay
	time.Sleep(time.Duration(delay) * time.Second)
	
	return nil
}

func (p *AICampaignProcessor) updateDeviceProgress(campaignID int, deviceID string, tracker *DeviceTracker) {
	progress := &models.AICampaignProgress{
		CampaignID:  campaignID,
		DeviceID:    deviceID,
		LeadsSent:   tracker.Sent,
		LeadsFailed: tracker.Failed,
		Status:      tracker.Status,
	}
	
	p.leadAIRepo.UpdateCampaignProgress(progress)
}