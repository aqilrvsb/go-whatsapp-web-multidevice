package usecase

import (
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// StartBroadcastWorkerProcessor starts the background worker that processes queued messages
// FIXED VERSION: Now uses Worker Pool System to prevent duplicate messages
func StartBroadcastWorkerProcessor() {
	logrus.Info("Starting broadcast worker processor with Worker Pool System...")
	
	// Get repositories
	broadcastRepo := repository.GetBroadcastRepository()

	// Get the Ultra Scale Broadcast Manager (Worker Pool System)
	broadcastManager := broadcast.GetBroadcastManager()
	
	// Process messages every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Get all devices with pending messages
			devices, err := broadcastRepo.GetDevicesWithPendingMessages()
			if err != nil {
				logrus.Errorf("Failed to get devices with pending messages: %v", err)
				continue
			}
			
			if len(devices) == 0 {
				continue // No pending messages
			}
			
			// logrus.Infof("Found %d devices with pending messages", len(devices))
			
			// Process each device
			for _, deviceID := range devices {
				// Get pending messages for this device
				messages, err := broadcastRepo.GetPendingMessagesAndLock(deviceID, 100)
				if err != nil {
					logrus.Errorf("Failed to get pending messages for device %s: %v", deviceID, err)
					continue
				}
				
				if len(messages) == 0 {
					continue
				}

				// SIMPLIFIED: For Whacenter/platform devices, no need to check device status
				// Just process all messages directly - device info not required
				logrus.Infof("Processing %d messages for device %s", len(messages), deviceID)

				// FIXED: Queue messages to Worker Pool instead of direct processing
				// This prevents duplicate messages by using channel-based queuing
				for _, msg := range messages {
					// Worker Pool will handle:
					// 1. Status updates (pending -> queued -> sent)
					// 2. Anti-spam delays
					// 3. Sequential sending with mutex lock
					// 4. Automatic retries on failure
					
					err := broadcastManager.QueueMessage(&msg)
					if err != nil {
						logrus.Errorf("Failed to queue message %s to worker pool: %v", msg.ID, err)
						// Update status to failed if can't queue
						broadcastRepo.UpdateMessageStatus(msg.ID, "failed", err.Error())
					} else {
						logrus.Debugf("Message %s queued to worker pool for device %s", msg.ID, deviceID)
					}
				}
				
				logrus.Infof("Queued %d messages to worker pool for device %s", len(messages), deviceID)
			}
		}
	}
}
