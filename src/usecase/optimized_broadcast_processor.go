package usecase

import (
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"context"
	"encoding/json"
)

const (
	MAX_CONCURRENT_WORKERS = 500  // Limit concurrent workers
	WORKER_CHECK_INTERVAL  = 10 * time.Second
	MESSAGE_BATCH_SIZE     = 100
)

type OptimizedBroadcastProcessor struct {
	redisClient   *redis.Client
	broadcastRepo *repository.BroadcastRepository
	userRepo      *repository.UserRepository
	manager       broadcast.BroadcastManagerInterface
	activeWorkers sync.Map // deviceID -> bool
	workerPool    chan struct{} // Semaphore for limiting concurrent workers
}

// StartOptimizedBroadcastProcessor starts the Redis-based broadcast processor for 3000+ devices
func StartOptimizedBroadcastProcessor() {
	logrus.Info("Starting OPTIMIZED broadcast processor for 3000+ devices...")
	
	// Initialize Redis
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		logrus.Errorf("Failed to parse Redis URL: %v", err)
		// Fall back to basic processor
		StartBroadcastWorkerProcessor()
		return
	}
	
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logrus.Errorf("Failed to connect to Redis: %v", err)
		// Fall back to basic processor
		StartBroadcastWorkerProcessor()
		return
	}
	
	logrus.Info("Connected to Redis for optimized broadcast processing")
	
	processor := &OptimizedBroadcastProcessor{
		redisClient:   redisClient,
		broadcastRepo: repository.GetBroadcastRepository(),
		userRepo:      repository.GetUserRepository(),
		manager:       broadcast.GetBroadcastManager(),
		workerPool:    make(chan struct{}, MAX_CONCURRENT_WORKERS),
	}
	
	// Initialize worker pool
	for i := 0; i < MAX_CONCURRENT_WORKERS; i++ {
		processor.workerPool <- struct{}{}
	}
	
	// Start the main processor
	go processor.run()
	
	// Start the queue monitor
	go processor.monitorQueues()
	
	// Start the worker health checker
	go processor.checkWorkerHealth()
}

func (p *OptimizedBroadcastProcessor) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.processNewMessages()
		}
	}
}
func (p *OptimizedBroadcastProcessor) processNewMessages() {
	// Get devices with pending messages from database
	devices, err := p.broadcastRepo.GetDevicesWithPendingMessages()
	if err != nil {
		logrus.Errorf("Failed to get devices with pending messages: %v", err)
		return
	}
	
	if len(devices) == 0 {
		return
	}
	
	logrus.Debugf("Found %d devices with pending messages", len(devices))
	
	// Process each device in parallel
	var wg sync.WaitGroup
	for _, deviceID := range devices {
		// Check if worker already exists for this device
		if _, exists := p.activeWorkers.Load(deviceID); exists {
			continue // Worker already processing this device
		}
		
		// Get a slot from the worker pool
		select {
		case <-p.workerPool:
			wg.Add(1)
			go func(devID string) {
				defer wg.Done()
				defer func() {
					p.workerPool <- struct{}{} // Return slot to pool
					p.activeWorkers.Delete(devID)
				}()
				
				// Mark as active
				p.activeWorkers.Store(devID, true)
				
				// Process messages for this device
				p.processDeviceMessages(devID)
			}(deviceID)
		default:
			// Worker pool is full, skip this device for now
			logrus.Debugf("Worker pool full, skipping device %s", deviceID)
		}
	}
	
	// Wait for all workers to complete
	wg.Wait()
}

func (p *OptimizedBroadcastProcessor) processDeviceMessages(deviceID string) {
	ctx := context.Background()
	
	// Check device status
	device, err := p.userRepo.GetDeviceByID(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get device %s: %v", deviceID, err)
		return
	}
	
	// Check if device is online
	if device.Status != "online" && device.Status != "Online" && 
	   device.Status != "connected" && device.Status != "Connected" {
		logrus.Debugf("Device %s is not online (status: %s)", deviceID, device.Status)
		return
	}
	
	// Check if WhatsApp client exists for this device
	clientManager := whatsapp.GetClientManager()
	_, err = clientManager.GetClient(deviceID)
	if err != nil {
		logrus.Warnf("WhatsApp client not found for device %s, trying to find alternative device", deviceID)
		
		// Try to find an alternative connected device for the same user
		devices, _ := p.userRepo.GetUserDevices(device.UserID)
		var alternativeDeviceID string
		
		for _, altDevice := range devices {
			if altDevice.ID != deviceID && 
			   (altDevice.Status == "online" || altDevice.Status == "Online" || 
			    altDevice.Status == "connected" || altDevice.Status == "Connected") {
				// Check if this device has a WhatsApp client
				if _, err := clientManager.GetClient(altDevice.ID); err == nil {
					alternativeDeviceID = altDevice.ID
					logrus.Infof("Found alternative device %s for user", alternativeDeviceID)
					break
				}
			}
		}
		
		if alternativeDeviceID == "" {
			logrus.Errorf("No alternative devices found for user, skipping messages for device %s", deviceID)
			// Mark messages as failed
			messages, _ := p.broadcastRepo.GetPendingMessages(deviceID, 1000)
			for _, msg := range messages {
				p.broadcastRepo.UpdateMessageStatus(msg.ID, "failed", "Device not connected")
			}
			return
		}
		
		// Update all pending messages to use the alternative device
		query := `
			UPDATE broadcast_messages 
			SET device_id = $1 
			WHERE device_id = $2 AND status = 'pending'
		`
		db := database.GetDB()
		result, err := db.Exec(query, alternativeDeviceID, deviceID)
		if err != nil {
			logrus.Errorf("Failed to update messages to alternative device: %v", err)
			return
		}
		
		rowsAffected, _ := result.RowsAffected()
		logrus.Infof("Updated %d messages from device %s to %s", rowsAffected, deviceID, alternativeDeviceID)
		
		// Process the alternative device instead
		deviceID = alternativeDeviceID
		device.ID = alternativeDeviceID
	}
	
	// Create Redis queue key for this device
	queueKey := fmt.Sprintf("broadcast:queue:%s", deviceID)
	processingKey := fmt.Sprintf("broadcast:processing:%s", deviceID)
	
	// Move messages from database to Redis queue (if not already there)
	messages, err := p.broadcastRepo.GetPendingMessages(deviceID, MESSAGE_BATCH_SIZE)
	if err != nil {
		logrus.Errorf("Failed to get pending messages for device %s: %v", deviceID, err)
		return
	}
	
	// Add messages to Redis queue
	for _, msg := range messages {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			logrus.Errorf("Failed to marshal message: %v", err)
			continue
		}
		
		// Add to Redis queue with score as timestamp for ordering
		score := float64(msg.ScheduledAt.Unix())
		p.redisClient.ZAdd(ctx, queueKey, &redis.Z{
			Score:  score,
			Member: string(msgJSON),
		})
		
		// Update status in database
		p.broadcastRepo.UpdateMessageStatus(msg.ID, "queued", "")
	}
	
	// Process messages from Redis queue
	for {
		// Get next message from queue
		results, err := p.redisClient.ZPopMin(ctx, queueKey, 1).Result()
		if err != nil || len(results) == 0 {
			break // No more messages
		}
		
		// Parse message
		var msg domainBroadcast.BroadcastMessage
		if err := json.Unmarshal([]byte(results[0].Member.(string)), &msg); err != nil {
			logrus.Errorf("Failed to unmarshal message: %v", err)
			continue
		}
		
		// Move to processing set
		p.redisClient.SAdd(ctx, processingKey, results[0].Member)
		
		// Send message
		startTime := time.Now()
		err = p.manager.SendMessage(msg)
		sendDuration := time.Since(startTime)
		
		if err != nil {
			logrus.Errorf("Failed to send message %s: %v (took %v)", msg.ID, err, sendDuration)
			p.broadcastRepo.UpdateMessageStatus(msg.ID, "failed", err.Error())
			
			// Remove from processing
			p.redisClient.SRem(ctx, processingKey, results[0].Member)
			
			// Retry logic - add back to queue with delay
			retryScore := float64(time.Now().Add(1 * time.Minute).Unix())
			p.redisClient.ZAdd(ctx, queueKey, &redis.Z{
				Score:  retryScore,
				Member: results[0].Member,
			})
		} else {
			logrus.Infof("Sent message %s to %s in %v", msg.ID, msg.RecipientPhone, sendDuration)
			p.broadcastRepo.UpdateMessageStatus(msg.ID, "sent", "")
			
			// Remove from processing
			p.redisClient.SRem(ctx, processingKey, results[0].Member)
			
			// Update metrics
			p.updateMetrics(deviceID, true, sendDuration)
		}
		
		// Apply delay between messages
		delay := p.calculateDelay(msg.MinDelay, msg.MaxDelay)
		time.Sleep(delay)
	}
}

func (p *OptimizedBroadcastProcessor) calculateDelay(minDelay, maxDelay int) time.Duration {
	if minDelay <= 0 {
		minDelay = 5
	}
	if maxDelay <= 0 || maxDelay < minDelay {
		maxDelay = minDelay + 10
	}
	
	// Use average for now (could add randomization)
	avgDelay := (minDelay + maxDelay) / 2
	return time.Duration(avgDelay) * time.Second
}

func (p *OptimizedBroadcastProcessor) updateMetrics(deviceID string, success bool, duration time.Duration) {
	ctx := context.Background()
	metricsKey := fmt.Sprintf("broadcast:metrics:%s", deviceID)
	
	if success {
		p.redisClient.HIncrBy(ctx, metricsKey, "success_count", 1)
		p.redisClient.HSet(ctx, metricsKey, "last_success", time.Now().Unix())
	} else {
		p.redisClient.HIncrBy(ctx, metricsKey, "failed_count", 1)
		p.redisClient.HSet(ctx, metricsKey, "last_failure", time.Now().Unix())
	}
	
	// Update average send time
	p.redisClient.HSet(ctx, metricsKey, "last_send_duration_ms", duration.Milliseconds())
	
	// Set expiry on metrics
	p.redisClient.Expire(ctx, metricsKey, 24*time.Hour)
}

func (p *OptimizedBroadcastProcessor) monitorQueues() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	ctx := context.Background()
	
	for {
		select {
		case <-ticker.C:
			// Get all queue keys
			keys, err := p.redisClient.Keys(ctx, "broadcast:queue:*").Result()
			if err != nil {
				continue
			}
			
			totalQueued := 0
			for _, key := range keys {
				count, _ := p.redisClient.ZCard(ctx, key).Result()
				totalQueued += int(count)
			}
			
			if totalQueued > 0 {
				logrus.Infof("Total messages in queues: %d across %d devices", totalQueued, len(keys))
			}
			
			// Check active workers
			activeCount := 0
			p.activeWorkers.Range(func(key, value interface{}) bool {
				activeCount++
				return true
			})
			
			logrus.Infof("Active workers: %d/%d", activeCount, MAX_CONCURRENT_WORKERS)
		}
	}
}

func (p *OptimizedBroadcastProcessor) checkWorkerHealth() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	ctx := context.Background()
	
	for {
		select {
		case <-ticker.C:
			// Check for stuck messages in processing sets
			keys, err := p.redisClient.Keys(ctx, "broadcast:processing:*").Result()
			if err != nil {
				continue
			}
			
			for _, key := range keys {
				members, _ := p.redisClient.SMembers(ctx, key).Result()
				if len(members) > 0 {
					// Extract device ID from key
					deviceID := key[len("broadcast:processing:"):]
					
					// Check if worker is still active
					if _, exists := p.activeWorkers.Load(deviceID); !exists {
						// Worker died, move messages back to queue
						queueKey := fmt.Sprintf("broadcast:queue:%s", deviceID)
						for _, member := range members {
							p.redisClient.ZAdd(ctx, queueKey, &redis.Z{
								Score:  float64(time.Now().Unix()),
								Member: member,
							})
							p.redisClient.SRem(ctx, key, member)
						}
						logrus.Warnf("Recovered %d stuck messages for device %s", len(members), deviceID)
					}
				}
			}
		}
	}
}
