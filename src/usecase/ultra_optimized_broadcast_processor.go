package usecase

import (
	"fmt"
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/sirupsen/logrus"
)

// UltraOptimizedBroadcastProcessor uses broadcast-specific worker pools
type UltraOptimizedBroadcastProcessor struct {
	manager broadcast.UltraScaleBroadcastManager
	ticker  *time.Ticker
}

// StartUltraOptimizedBroadcastProcessor starts the ultra-optimized processor
func StartUltraOptimizedBroadcastProcessor() {
	processor := &UltraOptimizedBroadcastProcessor{
		manager: *broadcast.GetUltraScaleBroadcastManager(),
		ticker:  time.NewTicker(2 * time.Second), // Check every 2 seconds for faster response
	}
	
	logrus.Info("Starting Ultra-Optimized Broadcast Processor for 3000+ devices")
	
	// Process immediately on start
	processor.processMessages()
	
	// Then process periodically
	for range processor.ticker.C {
		processor.processMessages()
	}
}

func (p *UltraOptimizedBroadcastProcessor) processMessages() {
	db := database.GetDB()
	
	// Get pending messages grouped by broadcast
	rows, err := db.Query(`
		SELECT 
			bm.id, bm.user_id, bm.device_id, bm.campaign_id, bm.sequence_id,
			bm.recipient_phone, bm.content as message, bm.media_url as image_url,
			COALESCE(c.min_delay_seconds, 5) as min_delay,
			COALESCE(c.max_delay_seconds, 15) as max_delay,
			d.status as device_status
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN user_devices d ON bm.device_id = d.id
		WHERE bm.status = 'pending'
		AND bm.scheduled_at <= NOW()
		ORDER BY bm.campaign_id NULLS LAST, bm.sequence_id NULLS LAST, bm.created_at
		LIMIT 1000
	`)
	
	if err != nil {
		logrus.Errorf("Failed to get pending messages: %v", err)
		return
	}
	defer rows.Close()
	
	messageCount := 0
	campaignPools := make(map[int]bool)
	sequencePools := make(map[string]bool)
	
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var campaignID *int
		var sequenceID *string
		var minDelay, maxDelay int
		var deviceStatus string
		
		err := rows.Scan(
			&msg.ID, &msg.UserID, &msg.DeviceID, &campaignID, &sequenceID,
			&msg.RecipientPhone, &msg.Message, &msg.ImageURL,
			&minDelay, &maxDelay, &deviceStatus,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan message: %v", err)
			continue
		}
		
		// Check device status
		if deviceStatus != "connected" && deviceStatus != "online" {
			// Skip this device - mark messages as skipped
			db.Exec(`UPDATE broadcast_messages SET status = 'skipped', error_message = 'Device offline' 
					 WHERE device_id = $1 AND status = 'pending'`, msg.DeviceID)
			continue
		}
		
		// Set broadcast references
		msg.CampaignID = campaignID
		msg.SequenceID = sequenceID
		msg.MinDelay = minDelay
		msg.MaxDelay = maxDelay
		msg.Type = "text"
		if msg.ImageURL != "" {
			msg.Type = "image"
		}
		
		// Create pool if needed
		if campaignID != nil && !campaignPools[*campaignID] {
			_, err := p.manager.StartBroadcastPool("campaign", fmt.Sprintf("%d", *campaignID), msg.UserID)
			if err != nil {
				logrus.Errorf("Failed to start campaign pool: %v", err)
				continue
			}
			campaignPools[*campaignID] = true
			
			// Update campaign status to processing
			db.Exec(`UPDATE campaigns SET status = 'processing', 
					 first_processed_at = COALESCE(first_processed_at, NOW()) 
					 WHERE id = $1`, *campaignID)
		}
		
		if sequenceID != nil && !sequencePools[*sequenceID] {
			_, err := p.manager.StartBroadcastPool("sequence", *sequenceID, msg.UserID)
			if err != nil {
				logrus.Errorf("Failed to start sequence pool: %v", err)
				continue
			}
			sequencePools[*sequenceID] = true
		}
		
		// Queue message to appropriate pool
		err = p.manager.QueueMessageToBroadcast(&msg)
		if err != nil {
			logrus.Errorf("Failed to queue message: %v", err)
			// Update to failed
			db.Exec(`UPDATE broadcast_messages SET status = 'failed', error_message = $1 WHERE id = $2`, 
				err.Error(), msg.ID)
		} else {
			messageCount++
		}
	}
	
	if messageCount > 0 {
		logrus.Infof("Queued %d messages to broadcast pools", messageCount)
	}
}
