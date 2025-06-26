package broadcast

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
)

// Run starts the device worker
func (dw *DeviceWorker) Run() {
	logrus.Infof("Starting worker for device %s", dw.deviceID)
	
	for {
		select {
		case <-dw.ctx.Done():
			dw.isRunning = false
			return
			
		case msg := <-dw.queue:
			// Apply rate limiting
			dw.applyRateLimit()
			
			// Send message
			err := dw.sendMessage(msg)
			
			// Update status
			repo := repository.GetBroadcastRepository()
			if err != nil {
				logrus.Errorf("Failed to send message %s: %v", msg.ID, err)
				
				// Retry logic
				if msg.RetryCount < 3 {
					msg.RetryCount++
					// Re-queue with delay
					go func() {
						time.Sleep(time.Duration(30*msg.RetryCount) * time.Second)
						dw.queue <- msg
					}()
				} else {
					// Mark as failed
					repo.UpdateMessageStatus(msg.ID, "failed")
					repo.SetMessageError(msg.ID, err.Error())
				}
			} else {
				// Mark as sent
				repo.UpdateMessageStatus(msg.ID, "sent")
				dw.messagesSent++
				dw.lastSentTime = time.Now()
			}
		}
	}
}

// applyRateLimit applies delay between messages
func (dw *DeviceWorker) applyRateLimit() {
	// Calculate time since last message
	timeSinceLastMsg := time.Since(dw.lastSentTime)
	
	// Generate random delay between min and max
	delay := time.Duration(dw.minDelay) * time.Second
	if dw.maxDelay > dw.minDelay {
		randomDelay := rand.Intn(dw.maxDelay - dw.minDelay)
		delay = time.Duration(dw.minDelay+randomDelay) * time.Second
	}
	
	// Wait if needed
	if timeSinceLastMsg < delay {
		waitTime := delay - timeSinceLastMsg
		logrus.Debugf("Device %s: Waiting %v before next message", dw.deviceID, waitTime)
		time.Sleep(waitTime)
	}
}

// sendMessage sends a message through WhatsApp
func (dw *DeviceWorker) sendMessage(msg BroadcastMessage) error {
	// Ensure client is connected
	if !dw.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}
	
	// Sanitize phone number
	whatsapp.SanitizePhone(&msg.Phone)
	
	// Validate recipient
	recipient, err := whatsapp.ValidateJidWithLogin(dw.client, msg.Phone)
	if err != nil {
		return fmt.Errorf("invalid recipient: %v", err)
	}
	
	// Send based on content type
	if msg.MediaURL != "" {
		// Image message
		// TODO: Implement image sending with URL
		return dw.sendImageMessage(recipient, msg)
	} else {
		// Text message
		return dw.sendTextMessage(recipient, msg)
	}
}

// sendTextMessage sends text message
func (dw *DeviceWorker) sendTextMessage(recipient whatsmeow.JID, msg BroadcastMessage) error {
	message := &whatsmeow.Message{
		ExtendedTextMessage: &whatsmeow.ExtendedTextMessage{
			Text: &msg.Content,
		},
	}
	
	_, err := dw.client.SendMessage(context.Background(), recipient, message)
	return err
}

// sendImageMessage sends image message
func (dw *DeviceWorker) sendImageMessage(recipient whatsmeow.JID, msg BroadcastMessage) error {
	// TODO: Download image from URL and send
	// For now, return not implemented
	return fmt.Errorf("image sending not yet implemented")
}

// Stop stops the worker
func (dw *DeviceWorker) Stop() {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	
	if dw.cancel != nil {
		dw.cancel()
	}
	dw.isRunning = false
}

// IsHealthy checks if worker is healthy
func (dw *DeviceWorker) IsHealthy() bool {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	
	// Check if running
	if !dw.isRunning {
		return false
	}
	
	// Check if client is connected
	if dw.client != nil && !dw.client.IsConnected() {
		return false
	}
	
	// Check if stuck (no messages sent in last 10 minutes with queue)
	if len(dw.queue) > 0 && time.Since(dw.lastSentTime) > 10*time.Minute {
		return false
	}
	
	return true
}