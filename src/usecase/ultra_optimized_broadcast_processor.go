package usecase

import (
	"fmt"
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// UltraOptimizedBroadcastProcessor uses broadcast-specific worker pools
type UltraOptimizedBroadcastProcessor struct {
	manager *broadcast.UltraScaleBroadcastManager
	ticker  *time.Ticker
}

// StartUltraOptimizedBroadcastProcessor starts the ultra-optimized processor
func StartUltraOptimizedBroadcastProcessor() {
	processor := &UltraOptimizedBroadcastProcessor{
		manager: broadcast.GetUltraScaleBroadcastManager(),
		ticker:  time.NewTicker(5 * time.Second), // Check every 5 seconds
	}
	
	logrus.Info("🚀 Ultra-optimized broadcast processor starting...")
	logrus.Info("✅ UltraOptimizedBroadcastProcessor initialized successfully")
	logrus.Info("⏰ Will check for messages every 5 seconds")
	
	// Process immediately on start
	logrus.Info("🔄 Running initial message check...")
	processor.processMessages()
	
	// Then process periodically
	logrus.Info("♻️ Starting periodic processing loop...")
	for range processor.ticker.C {
		logrus.Debug("⏰ Ticker fired - checking for messages...")
		processor.processMessages()
	}
}

func (p *UltraOptimizedBroadcastProcessor) processMessages() {
	startTime := time.Now()
	logrus.Debug("📥 UltraOptimizedBroadcastProcessor.processMessages() started")
	
	// Get repository instance
	broadcastRepo := repository.GetBroadcastRepository()
	userRepo := repository.GetUserRepository()
	
	// First, clean up old messages (older than 1 day)
	db := database.GetDB()
	
	// TEMPORARY: Delete ALL pending messages to clear the bootloop
	result, err := db.Exec(`DELETE FROM broadcast_messages WHERE status = 'pending'`)
	if err != nil {
		logrus.Errorf("❌ Failed to delete pending messages: %v", err)
	} else {
		rowsDeleted, _ := result.RowsAffected()
		if rowsDeleted > 0 {
			logrus.Infof("🗑️ Deleted %d pending messages to clear bootloop", rowsDeleted)
		}
	}
	
	// Check how many old messages exist
	var oldCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM broadcast_messages 
		WHERE status = 'pending'
		AND scheduled_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 1 DAY)
	`).Scan(&oldCount)
	if err == nil && oldCount > 0 {
		logrus.Infof("🔍 Found %d old messages to clean up", oldCount)
	}
	
	// Delete old messages
	result, err = db.Exec(`
		DELETE FROM broadcast_messages 
		WHERE status = 'pending'
		AND scheduled_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 1 DAY)
		LIMIT 1000
	`)
	if err != nil {
		// Log error but continue processing
		logrus.Warnf("⚠️ Failed to clean old messages: %v", err)
	} else {
		rowsDeleted, _ := result.RowsAffected()
		if rowsDeleted > 0 {
			logrus.Infof("🗑️ Cleaned up %d old messages (older than 1 day)", rowsDeleted)
		}
	}
	
	// Get all devices with pending messages
	logrus.Debug("🔍 Fetching devices with pending messages...")
	devices, err := broadcastRepo.GetDevicesWithPendingMessages()
	if err != nil {
		logrus.Errorf("❌ Failed to get devices with pending messages: %v", err)
		return
	}
	
	if len(devices) == 0 {
		logrus.Debug("💤 No devices with pending messages found")
		return
	}
	
	logrus.Infof("📱 Found %d devices with pending messages", len(devices))
	
	// Check message ages for debugging
	var debugInfo struct {
		Total      int
		TooOld     int
		InWindow   int
		MinAge     string
		MaxAge     string
	}
	
	err = db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN scheduled_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 1 DAY) THEN 1 ELSE 0 END) as too_old,
			SUM(CASE WHEN scheduled_at >= DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 1 DAY) THEN 1 ELSE 0 END) as in_window,
			TIMESTAMPDIFF(HOUR, MAX(scheduled_at), DATE_ADD(NOW(), INTERVAL 8 HOUR)) as min_age_hours,
			TIMESTAMPDIFF(HOUR, MIN(scheduled_at), DATE_ADD(NOW(), INTERVAL 8 HOUR)) as max_age_hours
		FROM broadcast_messages
		WHERE status = 'pending'
	`).Scan(&debugInfo.Total, &debugInfo.TooOld, &debugInfo.InWindow, &debugInfo.MinAge, &debugInfo.MaxAge)
	
	if err == nil {
		logrus.Infof("📊 Pending messages: Total=%d, TooOld=%d, InWindow=%d, Age=%s-%s hours", 
			debugInfo.Total, debugInfo.TooOld, debugInfo.InWindow, debugInfo.MinAge, debugInfo.MaxAge)
	}
	
	messageCount := 0
	campaignPools := make(map[int]bool)
	sequencePools := make(map[string]bool)
	
	// Process each device using GetPendingMessagesAndLock for atomic locking
	for i, deviceID := range devices {
		logrus.Debugf("🔐 Processing device %d/%d: %s", i+1, len(devices), deviceID)
		
		// Use GetPendingMessagesAndLock to atomically claim messages
		messages, err := broadcastRepo.GetPendingMessagesAndLock(deviceID, 100)
		if err != nil {
			logrus.Errorf("❌ Failed to get pending messages for device %s: %v", deviceID, err)
			continue
		}
		
		if len(messages) == 0 {
			// Let's check why no messages were claimed
			var pendingCount int
			db.QueryRow(`
				SELECT COUNT(*) FROM broadcast_messages 
				WHERE device_id = ? AND status = 'pending'
			`, deviceID).Scan(&pendingCount)
			
			if pendingCount > 0 {
				logrus.Warnf("⚠️ Device %s has %d pending messages but none were claimed (possible time window issue)", deviceID, pendingCount)
			} else {
				logrus.Debugf("💤 No messages to process for device %s", deviceID)
			}
			continue
		}
		
		logrus.Infof("📨 Found %d messages for device %s", len(messages), deviceID)
		
		// Get device details
		device, err := userRepo.GetDeviceByID(deviceID)
		if err != nil {
			logrus.Errorf("❌ Failed to get device %s: %v", deviceID, err)
			continue
		}
		
		logrus.Debugf("📱 Device %s - Status: %s, Platform: %s", deviceID, device.Status, device.Platform)
		
		// Check if device is online (platform devices always considered online)
		if device.Platform == "" && device.Status != "connected" && device.Status != "online" {
			// Skip this WhatsApp Web device - mark messages as skipped
			logrus.Warnf("⚠️ Device %s is offline (status: %s), skipping messages", deviceID, device.Status)
			db := database.GetDB()
			db.Exec(`UPDATE broadcast_messages SET status = 'skipped', error_message = 'Device offline' 
					 WHERE device_id = ? AND status = 'processing'`, deviceID)
			continue
		}
		
		logrus.Infof("✅ Device %s is online, processing messages...", deviceID)
		
		// Process each message
		for _, msg := range messages {
			// Create pool if needed
			if msg.CampaignID != nil && !campaignPools[*msg.CampaignID] {
				logrus.Infof("🎯 Creating campaign pool for campaign ID: %d", *msg.CampaignID)
				_, err := p.manager.StartBroadcastPool("campaign", fmt.Sprintf("%d", *msg.CampaignID))
				if err != nil {
					logrus.Errorf("❌ Failed to start campaign pool: %v", err)
					continue
				}
				campaignPools[*msg.CampaignID] = true
				
				// Update campaign status to processing
				db := database.GetDB()
				db.Exec(`UPDATE campaigns SET status = 'processing', 
						 updated_at = NOW() 
						 WHERE id = ?`, *msg.CampaignID)
				logrus.Infof("📊 Updated campaign %d status to 'processing'", *msg.CampaignID)
			}
			
			if msg.SequenceID != nil && !sequencePools[*msg.SequenceID] {
				logrus.Infof("🔄 Creating sequence pool for sequence ID: %s", *msg.SequenceID)
				_, err := p.manager.StartBroadcastPool("sequence", *msg.SequenceID)
				if err != nil {
					logrus.Errorf("❌ Failed to start sequence pool: %v", err)
					continue
				}
				sequencePools[*msg.SequenceID] = true
			}
			
			// Queue message to appropriate pool
			var broadcastType, broadcastID string
			if msg.CampaignID != nil {
				broadcastType = "campaign"
				broadcastID = fmt.Sprintf("%d", *msg.CampaignID)
				logrus.Debugf("📮 Processing campaign message %s for %s", msg.ID, msg.RecipientPhone)
			} else if msg.SequenceID != nil {
				broadcastType = "sequence"
				broadcastID = *msg.SequenceID
				logrus.Debugf("📮 Processing sequence message %s for %s", msg.ID, msg.RecipientPhone)
			}
			
			if broadcastType != "" {
				logrus.Debugf("📤 Queueing message %s to %s pool %s", msg.ID, broadcastType, broadcastID)
				err = p.manager.QueueMessageToBroadcast(broadcastType, broadcastID, &msg)
				if err != nil {
					logrus.Errorf("❌ Failed to queue message %s: %v", msg.ID, err)
					// Update to failed
					db := database.GetDB()
					db.Exec(`UPDATE broadcast_messages SET status = 'failed', error_message = ? WHERE id = ?`, 
						err.Error(), msg.ID)
				} else {
					messageCount++
					logrus.Debugf("✅ Successfully queued message %s", msg.ID)
				}
			} else {
				logrus.Warnf("⚠️ Message %s has no campaign or sequence ID, skipping", msg.ID)
			}
		}
	}
	
	if messageCount > 0 {
		logrus.Infof("✨ Queued %d messages to broadcast pools in %v", messageCount, time.Since(startTime))
	} else {
		logrus.Debugf("💤 No messages queued in this cycle (took %v)", time.Since(startTime))
	}
}
