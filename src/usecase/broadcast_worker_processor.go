package usecase

import (
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// StartBroadcastWorkerProcessor starts the background worker that processes queued messages
func StartBroadcastWorkerProcessor() {
	logrus.Info("Starting broadcast worker processor...")
	
	// Get the broadcast manager instance
	manager := broadcast.GetBroadcastManager()
	broadcastRepo := repository.GetBroadcastRepository()
	userRepo := repository.GetUserRepository()
	
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
			
			logrus.Infof("Found %d devices with pending messages", len(devices))
			
			// Process each device
			for _, deviceID := range devices {
				// Get pending messages for this device
				messages, err := broadcastRepo.GetPendingMessages(deviceID, 10) // Get up to 10 messages
				if err != nil {
					logrus.Errorf("Failed to get pending messages for device %s: %v", deviceID, err)
					continue
				}
				
				if len(messages) == 0 {
					continue
				}
				
				// Get device details
				device, err := userRepo.GetDeviceByID(deviceID)
				if err != nil {
					logrus.Errorf("Failed to get device %s: %v", deviceID, err)
					continue
				}
				
				// Check if device is online
				if device.Status != "online" && device.Status != "Online" && 
				   device.Status != "connected" && device.Status != "Connected" {
					logrus.Debugf("Device %s is not online (status: %s), skipping", deviceID, device.Status)
					continue
				}
				
				logrus.Infof("Processing %d messages for device %s", len(messages), deviceID)
				
				// Process messages directly
				for _, msg := range messages {
					// Update status to processing
					err := broadcastRepo.UpdateMessageStatus(msg.ID, "processing", "")
					if err != nil {
						logrus.Errorf("Failed to update message status: %v", err)
						continue
					}
					
					// Send message using the manager
					err = manager.SendMessage(msg)
					if err != nil {
						logrus.Errorf("Failed to send message %s: %v", msg.ID, err)
						// Update status to failed
						broadcastRepo.UpdateMessageStatus(msg.ID, "failed", err.Error())
					} else {
						logrus.Infof("Successfully sent message %s to %s", msg.ID, msg.RecipientPhone)
						// Update status to sent
						broadcastRepo.UpdateMessageStatus(msg.ID, "sent", "")
					}
					
					// Add delay between messages
					if msg.MinDelay > 0 && msg.MaxDelay > 0 {
						delay := time.Duration(msg.MinDelay) * time.Second
						if msg.MaxDelay > msg.MinDelay {
							// Random delay between min and max
							delay = time.Duration(msg.MinDelay + (msg.MaxDelay-msg.MinDelay)/2) * time.Second
						}
						time.Sleep(delay)
					} else {
						// Default delay
						time.Sleep(5 * time.Second)
					}
				}
			}
		}
	}
}
