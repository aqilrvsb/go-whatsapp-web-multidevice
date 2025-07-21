// Unified processor for campaigns and sequences using Redis
package usecase

import (
	"fmt"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/sirupsen/logrus"
)

// UnifiedBroadcastProcessor handles both campaigns and sequences with Redis
type UnifiedBroadcastProcessor struct {
	manager  broadcast.BroadcastManagerInterface
	ticker   *time.Ticker
	stopChan chan bool
}

// StartUnifiedBroadcastProcessor starts the unified processor
func StartUnifiedBroadcastProcessor() {
	processor := &UnifiedBroadcastProcessor{
		manager:  broadcast.GetBroadcastManager(), // This will enforce Redis
		ticker:   time.NewTicker(2 * time.Second),
		stopChan: make(chan bool),
	}
	
	logrus.Info("🚀 Starting Unified Broadcast Processor (Redis-based for 3000+ devices)")
	
	// Process immediately on start
	processor.processMessages()
	
	// Then process periodically
	go processor.run()
}

func (p *UnifiedBroadcastProcessor) run() {
	for {
		select {
		case <-p.ticker.C:
			p.processMessages()
		case <-p.stopChan:
			return
		}
	}
}

func (p *UnifiedBroadcastProcessor) processMessages() {
	db := database.GetDB()
	
	// Get pending messages for BOTH campaigns and sequences
	query := `
		SELECT 
			bm.id, bm.user_id, bm.device_id, 
			bm.campaign_id, bm.sequence_id, bm.sequence_stepid,
			bm.recipient_phone, bm.recipient_name,
			bm.content as message, bm.media_url as image_url,
			bm.type as message_type,
			COALESCE(c.min_delay_seconds, s.min_delay_seconds, 5) as min_delay,
			COALESCE(c.max_delay_seconds, s.max_delay_seconds, 15) as max_delay,
			d.status as device_status,
			d.platform as device_platform,
			CASE 
				WHEN bm.campaign_id IS NOT NULL THEN 'campaign'
				WHEN bm.sequence_id IS NOT NULL THEN 'sequence'
			END as broadcast_type,
			COALESCE(bm.campaign_id::text, bm.sequence_id) as broadcast_id
		FROM broadcast_messages bm
		LEFT JOIN campaigns c ON bm.campaign_id = c.id
		LEFT JOIN sequences s ON bm.sequence_id = s.id
		LEFT JOIN user_devices d ON bm.device_id = d.id
		WHERE bm.status = 'pending'
		AND bm.scheduled_at <= NOW()
		ORDER BY 
			bm.created_at ASC,
			bm.campaign_id NULLS LAST,
			bm.sequence_id NULLS LAST
		LIMIT 5000
	`
	
	rows, err := db.Query(query)
	if err != nil {
		logrus.Errorf("Failed to get pending messages: %v", err)
		return
	}
	defer rows.Close()
	
	// Track unique broadcasts
	activePools := make(map[string]bool)
	messageCount := 0
	skippedCount := 0
	
	for rows.Next() {
		var msg domainBroadcast.BroadcastMessage
		var broadcastType, broadcastID string
		var deviceStatus, devicePlatform *string
		
		err := rows.Scan(
			&msg.ID, &msg.UserID, &msg.DeviceID,
			&msg.CampaignID, &msg.SequenceID, &msg.SequenceStepID,
			&msg.RecipientPhone, &msg.RecipientName,
			&msg.Message, &msg.ImageURL, &msg.Type,
			&msg.MinDelay, &msg.MaxDelay,
			&deviceStatus, &devicePlatform,
			&broadcastType, &broadcastID,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan message: %v", err)
			continue
		}
		
		// Check device availability
		if !p.isDeviceAvailable(deviceStatus, devicePlatform) {
			// Mark as skipped
			db.Exec(`UPDATE broadcast_messages SET 
					status = 'skipped', 
					error_message = 'Device offline or unavailable' 
				WHERE id = $1`, msg.ID)
			skippedCount++
			continue
		}
		
		// Ensure pool exists
		poolKey := fmt.Sprintf("%s:%s", broadcastType, broadcastID)
		if !activePools[poolKey] {
			redisManager, ok := p.manager.(*broadcast.UltraScaleRedisManager)
			if !ok {
				logrus.Error("Manager is not UltraScaleRedisManager - Redis is required!")
				continue
			}
			
			_, err := redisManager.CreateOptimizedPool(broadcastType, broadcastID, msg.UserID)
			if err != nil {
				logrus.Errorf("Failed to create pool for %s: %v", poolKey, err)
				continue
			}
			
			activePools[poolKey] = true
			logrus.Infof("Created pool for %s %s", broadcastType, broadcastID)
			
			// Update status to processing
			if broadcastType == "campaign" {
				db.Exec(`UPDATE campaigns SET status = 'processing' WHERE id = $1`, broadcastID)
			}
		}
		
		// Queue message to pool
		redisManager := p.manager.(*broadcast.UltraScaleRedisManager)
		pool, exists := redisManager.GetPool(poolKey)
		if !exists {
			logrus.Errorf("Pool %s not found after creation!", poolKey)
			continue
		}
		
		// Queue the message
		if err := pool.QueueMessage(&msg); err != nil {
			logrus.Errorf("Failed to queue message: %v", err)
			db.Exec(`UPDATE broadcast_messages SET status = 'failed', error_message = $1 WHERE id = $2`, 
				err.Error(), msg.ID)
		} else {
			messageCount++
		}
	}
	
	if messageCount > 0 || skippedCount > 0 {
		logrus.Infof("Processed batch: %d queued, %d skipped, %d active pools", 
			messageCount, skippedCount, len(activePools))
	}
}

// isDeviceAvailable checks if device can send messages
func (p *UnifiedBroadcastProcessor) isDeviceAvailable(status, platform *string) bool {
	// Platform devices (Wablas/Whacenter) are always available
	if platform != nil && *platform != "" {
		return true
	}
	
	// Regular WhatsApp devices must be online/connected
	if status == nil {
		return false
	}
	
	return *status == "online" || *status == "connected"
}

// Stop stops the processor
func (p *UnifiedBroadcastProcessor) Stop() {
	p.ticker.Stop()
	p.stopChan <- true
	logrus.Info("Unified Broadcast Processor stopped")
}
