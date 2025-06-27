package usecase

import (
	"fmt"
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

// OptimizedSequenceTrigger handles sequence execution with optimized broadcasting
type OptimizedSequenceTrigger struct {
	broadcastManager *broadcast.OptimizedBroadcastManager
	whatsappService  services.IWhatsappService
	ticker           *time.Ticker
	stopChan         chan bool
	isRunning        bool
	mutex            sync.Mutex
}

// NewOptimizedSequenceTrigger creates new optimized sequence trigger
func NewOptimizedSequenceTrigger(whatsappService services.IWhatsappService) *OptimizedSequenceTrigger {
	return &OptimizedSequenceTrigger{
		broadcastManager: broadcast.GetBroadcastManager(), // Same broadcast manager as campaigns
		whatsappService:  whatsappService,
		stopChan:         make(chan bool),
	}
}

// Start begins the sequence trigger service
func (st *OptimizedSequenceTrigger) Start() {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	
	if st.isRunning {
		logrus.Warn("Sequence trigger already running")
		return
	}
	
	st.ticker = time.NewTicker(5 * time.Minute) // Check every 5 minutes
	st.isRunning = true
	
	go st.run()
	logrus.Info("Sequence trigger started")
}

// Stop stops the sequence trigger service
func (st *OptimizedSequenceTrigger) Stop() {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	
	if !st.isRunning {
		return
	}
	
	st.ticker.Stop()
	st.stopChan <- true
	st.isRunning = false
	
	logrus.Info("Sequence trigger stopped")
}

// run is the main loop
func (st *OptimizedSequenceTrigger) run() {
	// Process immediately on start
	st.processSequences()
	
	for {
		select {
		case <-st.ticker.C:
			st.processSequences()
		case <-st.stopChan:
			return
		}
	}
}

// processSequences checks and processes active sequences
func (st *OptimizedSequenceTrigger) processSequences() {
	startTime := time.Now()
	logrus.Debug("Starting sequence processing...")
	
	// Get active sequences
	sequenceRepo := repository.GetSequenceRepository()
	sequences, err := sequenceRepo.GetActiveSequences()
	if err != nil {
		logrus.Errorf("Failed to get active sequences: %v", err)
		return
	}
	
	if len(sequences) == 0 {
		return
	}
	
	logrus.Infof("Found %d active sequences to process", len(sequences))
	
	// Process each sequence
	for _, sequence := range sequences {
		st.processSequence(&sequence)
	}
	
	logrus.Infof("Sequence processing completed in %v", time.Since(startTime))
}

// processSequence processes a single sequence
func (st *OptimizedSequenceTrigger) processSequence(sequence *models.Sequence) {
	logrus.Infof("Processing sequence: %s (ID: %s)", sequence.Name, sequence.ID)
	
	// Get sequence contacts that need processing
	sequenceRepo := repository.GetSequenceRepository()
	contacts, err := sequenceRepo.GetSequenceContactsForProcessing(sequence.ID)
	if err != nil {
		logrus.Errorf("Failed to get contacts for sequence %s: %v", sequence.ID, err)
		return
	}
	
	if len(contacts) == 0 {
		logrus.Debugf("No contacts need processing for sequence %s", sequence.ID)
		return
	}
	
	logrus.Infof("Found %d contacts to process for sequence %s", len(contacts), sequence.ID)
	
	// Get user's connected devices
	devices, err := st.getUserDevices(sequence.UserID)
	if err != nil || len(devices) == 0 {
		logrus.Errorf("No connected devices for user %s", sequence.UserID)
		return
	}
	
	// Ensure workers exist for all devices
	st.ensureWorkersForDevices(devices)
	
	// Process each contact
	deviceIndex := 0
	broadcastRepo := repository.GetBroadcastRepository()
	
	for _, contact := range contacts {
		// Get the next step for this contact
		nextStep, err := sequenceRepo.GetSequenceStep(sequence.ID, contact.CurrentStep+1)
		if err != nil {
			logrus.Errorf("Failed to get step %d for sequence %s: %v", 
				contact.CurrentStep+1, sequence.ID, err)
			continue
		}
		
		// Check if it's time to send based on schedule_time
		if !st.isTimeToSend(nextStep.ScheduleTime) {
			continue
		}
		
		// Select device (round-robin)
		device := devices[deviceIndex%len(devices)]
		deviceIndex++
		
		// Format phone number
		phoneJID := contact.ContactPhone
		if !strings.Contains(phoneJID, "@") {
			phoneJID = strings.TrimPrefix(phoneJID, "+")
			phoneJID = phoneJID + "@s.whatsapp.net"
		}
		
		// Create broadcast message record
		broadcastMsg := &models.BroadcastMessage{
			ID:             uuid.New().String(),
			UserID:         sequence.UserID,
			DeviceID:       device.ID,
			SequenceID:     &sequence.ID,
			RecipientPhone: contact.ContactPhone,
			MessageType:    "text",
			Content:        nextStep.Content,
			MediaURL:       nextStep.ImageURL,
			Status:         "pending",
			ScheduledAt:    time.Now(),
			CreatedAt:      time.Now(),
			GroupID:        sequence.ID, // Use sequence ID as group
			GroupOrder:     contact.CurrentStep + 1,
		}
		
		// Save to database
		err = broadcastRepo.CreateBroadcastMessage(broadcastMsg)
		if err != nil {
			logrus.Errorf("Failed to create broadcast message: %v", err)
			continue
		}
		
		// Create domain message with delay settings
		domainMsg := &broadcast.BroadcastMessage{
			ID:           broadcastMsg.ID,
			DeviceID:     device.ID,
			RecipientJID: phoneJID,
			Message:      nextStep.Content,
			ImageURL:     nextStep.ImageURL,
			SequenceID:   &sequence.ID,
			GroupID:      sequence.ID,
			GroupOrder:   contact.CurrentStep + 1,
			RetryCount:   0,
			CreatedAt:    time.Now(),
			MinDelay:     nextStep.MinDelaySeconds,
			MaxDelay:     nextStep.MaxDelaySeconds,
		}
		
		// Queue message to worker
		err = st.broadcastManager.QueueMessage(device.ID, domainMsg)
		if err != nil {
			logrus.Errorf("Failed to queue sequence message: %v", err)
			_ = broadcastRepo.UpdateBroadcastStatus(broadcastMsg.ID, "failed", err.Error())
			continue
		}
		
		// Update contact progress
		err = sequenceRepo.UpdateContactProgress(contact.ID, contact.CurrentStep+1)
		if err != nil {
			logrus.Errorf("Failed to update contact progress: %v", err)
		}
		
		// Log sequence message
		_ = sequenceRepo.LogSequenceMessage(models.SequenceLog{
			ID:         uuid.New().String(),
			SequenceID: sequence.ID,
			ContactID:  contact.ID,
			StepID:     nextStep.ID,
			Day:        contact.CurrentStep + 1,
			Status:     "queued",
			MessageID:  broadcastMsg.ID,
			SentAt:     time.Now(),
			CreatedAt:  time.Now(),
		})
	}
	
	logrus.Infof("Queued %d messages for sequence %s", len(contacts), sequence.ID)
}

// isTimeToSend checks if current time matches schedule time
func (st *OptimizedSequenceTrigger) isTimeToSend(scheduleTime string) bool {
	if scheduleTime == "" {
		return true // No schedule, send anytime
	}
	
	now := time.Now()
	currentTime := now.Format("15:04")
	
	// Parse schedule time
	scheduleParts := strings.Split(scheduleTime, ":")
	if len(scheduleParts) != 2 {
		return true // Invalid format, send anyway
	}
	
	// Check if current time is past schedule time
	return currentTime >= scheduleTime
}

// getUserDevices gets connected devices for a user
func (st *OptimizedSequenceTrigger) getUserDevices(userID string) ([]*models.UserDevice, error) {
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
func (st *OptimizedSequenceTrigger) ensureWorkersForDevices(devices []*models.UserDevice) {
	for _, device := range devices {
		// Get WhatsApp client for device
		client := st.whatsappService.GetDeviceByID(device.ID)
		if client == nil {
			logrus.Warnf("No WhatsApp client for device %s", device.ID)
			continue
		}
		
		// Create or get worker (same workers handle both campaigns and sequences)
		_, err := st.broadcastManager.CreateOrGetWorker(device.ID, client)
		if err != nil {
			logrus.Errorf("Failed to create worker for device %s: %v", device.ID, err)
		}
	}
}

// GetStatus returns current status of the sequence trigger
func (st *OptimizedSequenceTrigger) GetStatus() map[string]interface{} {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	
	return map[string]interface{}{
		"running":    st.isRunning,
		"next_check": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}
}
