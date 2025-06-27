package usecase

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/services"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OptimizedCampaignTrigger handles campaign execution with optimized broadcasting
type OptimizedCampaignTrigger struct {
	broadcastManager *broadcast.OptimizedBroadcastManager
	whatsappService  services.IWhatsappService
	ticker           *time.Ticker
	stopChan         chan bool
	isRunning        bool
	mutex            sync.Mutex
}

// NewOptimizedCampaignTrigger creates new optimized campaign trigger
func NewOptimizedCampaignTrigger(whatsappService services.IWhatsappService) *OptimizedCampaignTrigger {
	return &OptimizedCampaignTrigger{
		broadcastManager: broadcast.GetBroadcastManager(),
		whatsappService:  whatsappService,
		stopChan:         make(chan bool),
	}
}

// Start begins the campaign trigger service
func (ct *OptimizedCampaignTrigger) Start() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	
	if ct.isRunning {
		logrus.Warn("Campaign trigger already running")
		return
	}
	
	ct.ticker = time.NewTicker(60 * time.Second) // Check every minute
	ct.isRunning = true
	
	go ct.run()
	logrus.Info("Campaign trigger started")
}

// Stop stops the campaign trigger service
func (ct *OptimizedCampaignTrigger) Stop() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	
	if !ct.isRunning {
		return
	}
	
	ct.ticker.Stop()
	ct.stopChan <- true
	ct.isRunning = false
	
	logrus.Info("Campaign trigger stopped")
}

// run is the main loop
func (ct *OptimizedCampaignTrigger) run() {
	// Process immediately on start
	ct.processCampaigns()
	
	for {
		select {
		case <-ct.ticker.C:
			ct.processCampaigns()
		case <-ct.stopChan:
			return
		}
	}
}

// processCampaigns checks and executes pending campaigns
func (ct *OptimizedCampaignTrigger) processCampaigns() {
	startTime := time.Now()
	logrus.Debug("Starting campaign processing...")
	
	// Get pending campaigns
	campaignRepo := repository.GetCampaignRepository()
	campaigns, err := campaignRepo.GetPendingCampaigns()
	if err != nil {
		logrus.Errorf("Failed to get pending campaigns: %v", err)
		return
	}
	
	if len(campaigns) == 0 {
		return
	}
	
	logrus.Infof("Found %d pending campaigns to process", len(campaigns))
	
	// Process each campaign in parallel
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Process max 10 campaigns simultaneously
	
	for _, campaign := range campaigns {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(c models.Campaign) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			ct.executeCampaign(&c)
		}(campaign)
	}
	
	wg.Wait()
	
	logrus.Infof("Campaign processing completed in %v", time.Since(startTime))
}

// executeCampaign executes a single campaign
func (ct *OptimizedCampaignTrigger) executeCampaign(campaign *models.Campaign) {
	logrus.Infof("Executing campaign: %s (ID: %d)", campaign.Title, campaign.ID)
	
	// Update campaign status to processing
	campaignRepo := repository.GetCampaignRepository()
	_ = campaignRepo.UpdateCampaignStatus(campaign.ID, "processing")
	
	// Get leads matching the campaign
	leads, err := ct.getLeadsForCampaign(campaign)
	if err != nil {
		logrus.Errorf("Failed to get leads for campaign %d: %v", campaign.ID, err)
		_ = campaignRepo.UpdateCampaignStatus(campaign.ID, "failed")
		return
	}
	
	if len(leads) == 0 {
		logrus.Warnf("No leads found for campaign %d (niche: %s)", campaign.ID, campaign.Niche)
		_ = campaignRepo.UpdateCampaignStatus(campaign.ID, "sent")
		return
	}
	
	logrus.Infof("Found %d leads for campaign %d", len(leads), campaign.ID)
	
	// Get user's connected devices
	devices, err := ct.getUserDevices(campaign.UserID)
	if err != nil || len(devices) == 0 {
		logrus.Errorf("No connected devices for user %s", campaign.UserID)
		_ = campaignRepo.UpdateCampaignStatus(campaign.ID, "failed")
		return
	}
	
	logrus.Infof("Using %d devices for campaign %d", len(devices), campaign.ID)
	
	// Ensure workers exist for all devices
	ct.ensureWorkersForDevices(devices)
	
	// Create broadcast messages
	broadcastMessages := ct.createBroadcastMessages(campaign, leads, devices)
	
	// Queue messages to workers
	ct.queueMessagesToWorkers(broadcastMessages, devices)
	
	// Update campaign status
	_ = campaignRepo.UpdateCampaignStatus(campaign.ID, "sent")
	
	logrus.Infof("Campaign %d queued %d messages across %d devices", 
		campaign.ID, len(broadcastMessages), len(devices))
}

// getLeadsForCampaign gets leads based on campaign settings
func (ct *OptimizedCampaignTrigger) getLeadsForCampaign(campaign *models.Campaign) ([]models.Lead, error) {
	leadRepo := repository.GetLeadRepository()
	
	// Get leads by niche if specified
	if campaign.Niche != "" && campaign.Niche != "all" {
		return leadRepo.GetLeadsByNiche(campaign.Niche)
	}
	
	// Get all leads for the user
	return leadRepo.GetLeadsByUserID(campaign.UserID)
}

// getUserDevices gets connected devices for a user
func (ct *OptimizedCampaignTrigger) getUserDevices(userID string) ([]*models.UserDevice, error) {
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetUserDevices(userID)
	if err != nil {
		return nil, err
	}
	
	// Filter only connected devices
	var connectedDevices []*models.UserDevice
	for _, device := range devices {
		if device.Status == "online" || device.Status == "connected" {
			connectedDevices = append(connectedDevices, device)
		}
	}
	
	return connectedDevices, nil
}

// ensureWorkersForDevices creates workers for all devices if not exists
func (ct *OptimizedCampaignTrigger) ensureWorkersForDevices(devices []*models.UserDevice) {
	for _, device := range devices {
		// Get WhatsApp client for device
		client := ct.whatsappService.GetDeviceByID(device.ID)
		if client == nil {
			logrus.Warnf("No WhatsApp client for device %s", device.ID)
			continue
		}
		
		// Create or get worker (no need to pass delays here, they're per-message)
		_, err := ct.broadcastManager.CreateOrGetWorker(device.ID, client)
		if err != nil {
			logrus.Errorf("Failed to create worker for device %s: %v", device.ID, err)
		}
	}
}

// createBroadcastMessages creates broadcast messages for the campaign
func (ct *OptimizedCampaignTrigger) createBroadcastMessages(campaign *models.Campaign, leads []models.Lead, devices []*models.UserDevice) []*domainBroadcast.BroadcastMessage {
	broadcastRepo := repository.GetBroadcastRepository()
	var messages []*domainBroadcast.BroadcastMessage
	
	// Group ID for this campaign
	groupID := uuid.New().String()
	
	// Distribute leads across devices
	deviceIndex := 0
	for i, lead := range leads {
		// Round-robin device selection
		device := devices[deviceIndex%len(devices)]
		deviceIndex++
		
		// Format phone number
		phoneJID := lead.Phone
		if !strings.Contains(phoneJID, "@") {
			phoneJID = strings.TrimPrefix(phoneJID, "+")
			phoneJID = phoneJID + "@s.whatsapp.net"
		}
		
		// Create broadcast message record
		broadcastMsg := &models.BroadcastMessage{
			ID:             uuid.New().String(),
			UserID:         campaign.UserID,
			DeviceID:       device.ID,
			CampaignID:     &campaign.ID,
			RecipientPhone: lead.Phone,
			MessageType:    "text",
			Content:        campaign.Message,
			MediaURL:       campaign.ImageURL,
			Status:         "pending",
			ScheduledAt:    time.Now(),
			CreatedAt:      time.Now(),
			GroupID:        groupID,
			GroupOrder:     i,
		}
		
		// Save to database
		err := broadcastRepo.CreateBroadcastMessage(broadcastMsg)
		if err != nil {
			logrus.Errorf("Failed to create broadcast message: %v", err)
			continue
		}
		
		// Create domain message with delay settings
		domainMsg := &domainBroadcast.BroadcastMessage{
			ID:           broadcastMsg.ID,
			DeviceID:     device.ID,
			RecipientJID: phoneJID,
			Message:      campaign.Message,
			ImageURL:     campaign.ImageURL,
			CampaignID:   &campaign.ID,
			GroupID:      groupID,
			GroupOrder:   i,
			RetryCount:   0,
			CreatedAt:    time.Now(),
			MinDelay:     campaign.MinDelaySeconds,
			MaxDelay:     campaign.MaxDelaySeconds,
		}
		
		messages = append(messages, domainMsg)
	}
	
	return messages
}

// queueMessagesToWorkers distributes messages to device workers
func (ct *OptimizedCampaignTrigger) queueMessagesToWorkers(messages []*domainBroadcast.BroadcastMessage, devices []*models.UserDevice) {
	// Group messages by device
	messagesByDevice := make(map[string][]*domainBroadcast.BroadcastMessage)
	for _, msg := range messages {
		messagesByDevice[msg.DeviceID] = append(messagesByDevice[msg.DeviceID], msg)
	}
	
	// Queue messages to each device's worker
	for deviceID, deviceMessages := range messagesByDevice {
		logrus.Infof("Queueing %d messages to device %s", len(deviceMessages), deviceID)
		
		// Shuffle messages to avoid patterns
		rand.Shuffle(len(deviceMessages), func(i, j int) {
			deviceMessages[i], deviceMessages[j] = deviceMessages[j], deviceMessages[i]
		})
		
		// Queue each message
		for _, msg := range deviceMessages {
			err := ct.broadcastManager.QueueMessage(deviceID, (*broadcast.BroadcastMessage)(msg))
			if err != nil {
				logrus.Errorf("Failed to queue message to device %s: %v", deviceID, err)
				// Update message status to failed
				broadcastRepo := repository.GetBroadcastRepository()
				_ = broadcastRepo.UpdateBroadcastStatus(msg.ID, "failed", err.Error())
			}
		}
	}
}

// GetStatus returns current status of the campaign trigger
func (ct *OptimizedCampaignTrigger) GetStatus() map[string]interface{} {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	
	return map[string]interface{}{
		"running":         ct.isRunning,
		"worker_status":   ct.broadcastManager.GetWorkerStatus(),
		"next_check":      time.Now().Add(60 * time.Second).Format(time.RFC3339),
	}
}
