package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
)

// RedisOptimizedBroadcastManager handles broadcasting with Redis queues for ultimate scale
type RedisOptimizedBroadcastManager struct {
	redisClient   *redis.Client
	workers       map[string]*DeviceWorker
	workersMutex  sync.RWMutex
	activeWorkers int32
	maxWorkers    int32
	
	// Metrics stored in Redis for multi-server support
	metricsPrefix string
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// DeviceWorker represents a worker for a specific device
type DeviceWorker struct {
	DeviceID        string
	Device          *whatsmeow.Client
	Status          string
	LastActivity    time.Time
	
	// Rate limiting (stored in Redis for persistence)
	rateLimitPrefix string
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.Mutex
}

// NewRedisOptimizedBroadcastManager creates manager with Redis support
func NewRedisOptimizedBroadcastManager() *RedisOptimizedBroadcastManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Get Redis URL from environment
	redisURL := config.GetRedisURL() // You'll need to add this to config
	if redisURL == "" {
		logrus.Warn("Redis URL not configured, falling back to localhost")
		redisURL = "redis://localhost:6379"
	}
	
	// Parse Redis URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logrus.Fatalf("Failed to parse Redis URL: %v", err)
	}
	
	// Configure connection pool for high performance
	opt.PoolSize = 100
	opt.MinIdleConns = 10
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	
	// Connect to Redis
	rdb := redis.NewClient(opt)
	
	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}
	
	manager := &RedisOptimizedBroadcastManager{
		redisClient:   rdb,
		workers:       make(map[string]*DeviceWorker),
		maxWorkers:    int32(config.MaxConcurrentWorkers),
		metricsPrefix: "broadcast:metrics:",
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Initialize metrics in Redis
	manager.initializeMetrics()
	
	// Start background routines
	go manager.healthCheckRoutine()
	go manager.metricsRoutine()
	go manager.queueMonitorRoutine()
	go manager.deadLetterQueueProcessor()
	
	logrus.Info("Redis-optimized broadcast manager initialized with ultimate performance")
	
	return manager
}

// initializeMetrics sets up Redis metrics keys
func (m *RedisOptimizedBroadcastManager) initializeMetrics() {
	keys := []string{
		m.metricsPrefix + "total_processed",
		m.metricsPrefix + "total_failed",
		m.metricsPrefix + "total_pending",
		m.metricsPrefix + "active_workers",
	}
	
	for _, key := range keys {
		m.redisClient.SetNX(m.ctx, key, 0, 0)
	}
}

// QueueMessage queues a message to Redis with priority support
func (m *RedisOptimizedBroadcastManager) QueueMessage(deviceID string, msg *domainBroadcast.BroadcastMessage) error {
	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	
	// Determine queue based on type (priority for campaigns)
	var queueKey string
	if msg.CampaignID != nil {
		queueKey = fmt.Sprintf("queue:priority:device:%s", deviceID)
	} else {
		queueKey = fmt.Sprintf("queue:device:%s", deviceID)
	}
	
	// Push to Redis list (FIFO queue)
	pipe := m.redisClient.Pipeline()
	pipe.RPush(m.ctx, queueKey, data)
	pipe.Incr(m.ctx, m.metricsPrefix+"total_pending")
	pipe.Expire(m.ctx, queueKey, 7*24*time.Hour) // 7 days expiry
	
	// Track queue size
	pipe.HIncrBy(m.ctx, "queue:sizes", deviceID, 1)
	
	_, err = pipe.Exec(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to queue message: %v", err)
	}
	
	// Wake up worker if sleeping
	m.notifyWorker(deviceID)
	
	logrus.Debugf("Queued message to device %s (type: %s)", deviceID, msg.Type)
	
	return nil
}

// CreateOrGetWorker creates a new worker or returns existing one
func (m *RedisOptimizedBroadcastManager) CreateOrGetWorker(deviceID string, device *whatsmeow.Client) (*DeviceWorker, error) {
	m.workersMutex.Lock()
	defer m.workersMutex.Unlock()
	
	// Check if worker already exists
	if worker, exists := m.workers[deviceID]; exists {
		return worker, nil
	}
	
	// Check if we can create more workers
	currentWorkers := atomic.LoadInt32(&m.activeWorkers)
	if currentWorkers >= m.maxWorkers {
		return nil, fmt.Errorf("max workers limit reached: %d", m.maxWorkers)
	}
	
	// Create new worker
	workerCtx, workerCancel := context.WithCancel(m.ctx)
	worker := &DeviceWorker{
		DeviceID:        deviceID,
		Device:          device,
		Status:          "active",
		LastActivity:    time.Now(),
		rateLimitPrefix: fmt.Sprintf("ratelimit:device:%s:", deviceID),
		ctx:             workerCtx,
		cancel:          workerCancel,
	}
	
	// Start worker routine
	m.wg.Add(1)
	go m.workerRoutine(worker)
	
	// Store worker
	m.workers[deviceID] = worker
	atomic.AddInt32(&m.activeWorkers, 1)
	
	// Update Redis metrics
	m.redisClient.Incr(m.ctx, m.metricsPrefix+"active_workers")
	
	logrus.Infof("Created worker for device %s (total workers: %d)", deviceID, currentWorkers+1)
	return worker, nil
}

// workerRoutine processes messages from Redis queue with ultimate efficiency
func (m *RedisOptimizedBroadcastManager) workerRoutine(worker *DeviceWorker) {
	defer m.wg.Done()
	defer func() {
		atomic.AddInt32(&m.activeWorkers, -1)
		m.redisClient.Decr(m.ctx, m.metricsPrefix+"active_workers")
		worker.Status = "stopped"
		logrus.Infof("Worker for device %s stopped", worker.DeviceID)
	}()
	
	priorityQueue := fmt.Sprintf("queue:priority:device:%s", worker.DeviceID)
	normalQueue := fmt.Sprintf("queue:device:%s", worker.DeviceID)
	workerKey := fmt.Sprintf("worker:status:%s", worker.DeviceID)
	
	// Update worker status in Redis
	m.updateWorkerStatus(worker, "active", 0)
	
	// Idle counter
	idleCount := 0
	maxIdleCount := 600 // 10 minutes of idle (1 second checks)
	
	for {
		select {
		case <-worker.ctx.Done():
			return
		default:
			// Check rate limits
			if !m.checkRateLimits(worker) {
				time.Sleep(time.Second)
				continue
			}
			
			// Try to get message from priority queue first, then normal queue
			var data []byte
			var err error
			
			// Use BLPOP for efficient waiting (1 second timeout)
			result, err := m.redisClient.BLPop(m.ctx, time.Second, priorityQueue, normalQueue).Result()
			
			if err == redis.Nil {
				// No messages, increment idle counter
				idleCount++
				if idleCount >= maxIdleCount {
					logrus.Infof("Worker %s idle timeout", worker.DeviceID)
					return
				}
				continue
			} else if err != nil {
				logrus.Errorf("Redis error for worker %s: %v", worker.DeviceID, err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			// Reset idle counter
			idleCount = 0
			
			// result[0] is the queue name, result[1] is the data
			data = []byte(result[1])
			
			// Update queue size
			m.redisClient.HIncrBy(m.ctx, "queue:sizes", worker.DeviceID, -1)
			
			// Deserialize message
			var msg domainBroadcast.BroadcastMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				logrus.Errorf("Failed to unmarshal message: %v", err)
				continue
			}
			
			// Update worker status
			m.updateWorkerStatus(worker, "processing", 0)
			
			// Process message with proper delays
			m.processMessageWithDelay(worker, &msg)
		}
	}
}

// processMessageWithDelay sends a message with proper delay handling
func (m *RedisOptimizedBroadcastManager) processMessageWithDelay(worker *DeviceWorker, msg *domainBroadcast.BroadcastMessage) {
	startTime := time.Now()
	
	// Update metrics
	m.redisClient.Decr(m.ctx, m.metricsPrefix+"total_pending")
	
	// Prepare recipient JID
	recipientJID, err := whatsmeow.ParseJID(msg.RecipientJID)
	if err != nil {
		logrus.Errorf("Invalid JID %s: %v", msg.RecipientJID, err)
		m.handleFailedMessage(worker, msg, err)
		return
	}
	
	// Determine if this is a two-part message (image + text)
	hasImage := msg.ImageURL != "" || msg.MediaURL != ""
	hasText := msg.Message != "" || msg.Content != ""
	
	var sendErr error
	
	if hasImage && hasText {
		// TWO-PART MESSAGE: Send image first, then text
		logrus.Debugf("Sending two-part message to %s (image + text)", recipientJID.String())
		
		// 1. Send image without caption
		imageURL := msg.ImageURL
		if imageURL == "" {
			imageURL = msg.MediaURL
		}
		sendErr = m.sendImageMessage(worker.Device, recipientJID, imageURL, "")
		if sendErr != nil {
			m.handleFailedMessage(worker, msg, sendErr)
			return
		}
		
		// 2. Wait 3 seconds between image and text
		time.Sleep(3 * time.Second)
		
		// 3. Send text message
		text := msg.Message
		if text == "" {
			text = msg.Content
		}
		sendErr = m.sendTextMessage(worker.Device, recipientJID, text)
		if sendErr != nil {
			m.handleFailedMessage(worker, msg, sendErr)
			return
		}
	} else if hasImage {
		// Image only
		imageURL := msg.ImageURL
		if imageURL == "" {
			imageURL = msg.MediaURL
		}
		sendErr = m.sendImageMessage(worker.Device, recipientJID, imageURL, "")
	} else if hasText {
		// Text only
		text := msg.Message
		if text == "" {
			text = msg.Content
		}
		sendErr = m.sendTextMessage(worker.Device, recipientJID, text)
	} else {
		logrus.Warnf("Message has no content: %s", msg.ID)
		return
	}
	
	if sendErr != nil {
		m.handleFailedMessage(worker, msg, sendErr)
	} else {
		m.handleSuccessMessage(worker, msg)
	}
	
	// Record processing time
	processingTime := time.Since(startTime)
	m.recordProcessingTime(worker.DeviceID, processingTime)
	
	// Apply random delay between different leads
	minDelay := msg.MinDelay
	maxDelay := msg.MaxDelay
	if minDelay == 0 || maxDelay == 0 {
		minDelay = config.DefaultMinDelaySeconds
		maxDelay = config.DefaultMaxDelaySeconds
	}
	
	// Calculate random delay
	delay := minDelay
	if maxDelay > minDelay {
		delay = minDelay + rand.Intn(maxDelay-minDelay+1)
	}
	
	logrus.Debugf("Waiting %d seconds before next message (min: %d, max: %d)", delay, minDelay, maxDelay)
	time.Sleep(time.Duration(delay) * time.Second)
}

// Additional helper methods...

func (m *RedisOptimizedBroadcastManager) sendTextMessage(device *whatsmeow.Client, recipient whatsmeow.JID, message string) error {
	// TODO: Implement actual WhatsApp sending
	logrus.Infof("Sending text to %s: %s", recipient.String(), message)
	time.Sleep(100 * time.Millisecond) // Simulate sending
	return nil
}

func (m *RedisOptimizedBroadcastManager) sendImageMessage(device *whatsmeow.Client, recipient whatsmeow.JID, imageURL, caption string) error {
	// TODO: Implement actual WhatsApp image sending
	if caption != "" {
		logrus.Infof("Sending image with caption to %s: %s (caption: %s)", recipient.String(), imageURL, caption)
	} else {
		logrus.Infof("Sending image to %s: %s", recipient.String(), imageURL)
	}
	time.Sleep(200 * time.Millisecond) // Simulate sending
	return nil
}

func (m *RedisOptimizedBroadcastManager) handleSuccessMessage(worker *DeviceWorker, msg *domainBroadcast.BroadcastMessage) {
	// Update metrics in Redis
	pipe := m.redisClient.Pipeline()
	pipe.Incr(m.ctx, m.metricsPrefix+"total_processed")
	pipe.HIncrBy(m.ctx, fmt.Sprintf("worker:stats:%s", worker.DeviceID), "processed", 1)
	pipe.Exec(m.ctx)
	
	// Update rate limit counters
	m.incrementRateLimit(worker)
	
	// Update broadcast message status in database
	broadcastRepo := repository.GetBroadcastRepository()
	_ = broadcastRepo.UpdateBroadcastStatus(msg.ID, "sent", "")
	
	// Log based on message type
	if msg.CampaignID != nil {
		logrus.Infof("Campaign message sent to %s via device %s", msg.RecipientJID, worker.DeviceID)
	} else if msg.SequenceID != nil {
		logrus.Infof("Sequence message sent to %s via device %s", msg.RecipientJID, worker.DeviceID)
	}
}

func (m *RedisOptimizedBroadcastManager) handleFailedMessage(worker *DeviceWorker, msg *domainBroadcast.BroadcastMessage, err error) {
	// Update metrics
	pipe := m.redisClient.Pipeline()
	pipe.Incr(m.ctx, m.metricsPrefix+"total_failed")
	pipe.HIncrBy(m.ctx, fmt.Sprintf("worker:stats:%s", worker.DeviceID), "failed", 1)
	pipe.Exec(m.ctx)
	
	msg.RetryCount++
	
	// Retry logic with exponential backoff
	if msg.RetryCount < config.RetryAttempts {
		retryDelay := time.Duration(msg.RetryCount*msg.RetryCount) * time.Minute
		retryTime := time.Now().Add(retryDelay)
		
		// Add to retry queue with score as retry time
		retryKey := fmt.Sprintf("queue:retry:device:%s", worker.DeviceID)
		data, _ := json.Marshal(msg)
		m.redisClient.ZAdd(m.ctx, retryKey, &redis.Z{
			Score:  float64(retryTime.Unix()),
			Member: data,
		})
		
		logrus.Warnf("Message to %s queued for retry #%d in %v", msg.RecipientJID, msg.RetryCount, retryDelay)
	} else {
		// Max retries reached, move to dead letter queue
		dlqKey := "queue:dead_letter"
		data, _ := json.Marshal(msg)
		m.redisClient.RPush(m.ctx, dlqKey, data)
		
		// Update status in database
		broadcastRepo := repository.GetBroadcastRepository()
		_ = broadcastRepo.UpdateBroadcastStatus(msg.ID, "failed", err.Error())
		
		logrus.Errorf("Message to %s failed after %d retries: %v", msg.RecipientJID, msg.RetryCount, err)
	}
}

func (m *RedisOptimizedBroadcastManager) checkRateLimits(worker *DeviceWorker) bool {
	now := time.Now()
	
	// Get current counters from Redis
	pipe := m.redisClient.Pipeline()
	minuteKey := worker.rateLimitPrefix + "minute:" + now.Format("200601021504")
	hourKey := worker.rateLimitPrefix + "hour:" + now.Format("2006010215")
	dayKey := worker.rateLimitPrefix + "day:" + now.Format("20060102")
	
	minuteCmd := pipe.Get(m.ctx, minuteKey)
	hourCmd := pipe.Get(m.ctx, hourKey)
	dayCmd := pipe.Get(m.ctx, dayKey)
	pipe.Exec(m.ctx)
	
	// Check limits
	minuteCount, _ := minuteCmd.Int()
	if minuteCount >= config.MessagesPerMinute {
		logrus.Warnf("Device %s hit minute rate limit", worker.DeviceID)
		return false
	}
	
	hourCount, _ := hourCmd.Int()
	if hourCount >= config.MessagesPerHour {
		logrus.Warnf("Device %s hit hour rate limit", worker.DeviceID)
		return false
	}
	
	dayCount, _ := dayCmd.Int()
	if dayCount >= config.MessagesPerDay {
		logrus.Warnf("Device %s hit day rate limit", worker.DeviceID)
		return false
	}
	
	return true
}

func (m *RedisOptimizedBroadcastManager) incrementRateLimit(worker *DeviceWorker) {
	now := time.Now()
	
	pipe := m.redisClient.Pipeline()
	
	// Increment counters with appropriate expiry
	minuteKey := worker.rateLimitPrefix + "minute:" + now.Format("200601021504")
	pipe.Incr(m.ctx, minuteKey)
	pipe.Expire(m.ctx, minuteKey, 2*time.Minute)
	
	hourKey := worker.rateLimitPrefix + "hour:" + now.Format("2006010215")
	pipe.Incr(m.ctx, hourKey)
	pipe.Expire(m.ctx, hourKey, 2*time.Hour)
	
	dayKey := worker.rateLimitPrefix + "day:" + now.Format("20060102")
	pipe.Incr(m.ctx, dayKey)
	pipe.Expire(m.ctx, dayKey, 25*time.Hour)
	
	pipe.Exec(m.ctx)
}

func (m *RedisOptimizedBroadcastManager) updateWorkerStatus(worker *DeviceWorker, status string, queueSize int) {
	worker.mutex.Lock()
	worker.Status = status
	worker.LastActivity = time.Now()
	worker.mutex.Unlock()
	
	// Update in Redis
	workerKey := fmt.Sprintf("worker:status:%s", worker.DeviceID)
	m.redisClient.HSet(m.ctx, workerKey,
		"status", status,
		"last_activity", worker.LastActivity.Unix(),
		"queue_size", queueSize,
	)
	m.redisClient.Expire(m.ctx, workerKey, 24*time.Hour)
}

func (m *RedisOptimizedBroadcastManager) notifyWorker(deviceID string) {
	// Publish notification to wake up sleeping workers
	m.redisClient.Publish(m.ctx, fmt.Sprintf("worker:notify:%s", deviceID), "wake")
}

func (m *RedisOptimizedBroadcastManager) recordProcessingTime(deviceID string, duration time.Duration) {
	key := fmt.Sprintf("worker:performance:%s", deviceID)
	m.redisClient.HIncrByFloat(m.ctx, key, "total_time_ms", duration.Seconds()*1000)
	m.redisClient.HIncrBy(m.ctx, key, "message_count", 1)
}

// Background routines

func (m *RedisOptimizedBroadcastManager) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

func (m *RedisOptimizedBroadcastManager) performHealthCheck() {
	m.workersMutex.RLock()
	workers := make([]*DeviceWorker, 0, len(m.workers))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	m.workersMutex.RUnlock()
	
	for _, worker := range workers {
		// Check worker health in Redis
		workerKey := fmt.Sprintf("worker:status:%s", worker.DeviceID)
		lastActivity, err := m.redisClient.HGet(m.ctx, workerKey, "last_activity").Int64()
		if err == nil {
			if time.Now().Unix()-lastActivity > 600 { // 10 minutes
				logrus.Warnf("Worker %s appears stuck", worker.DeviceID)
				// Could implement restart logic here
			}
		}
	}
}

func (m *RedisOptimizedBroadcastManager) metricsRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.reportMetrics()
		}
	}
}

func (m *RedisOptimizedBroadcastManager) reportMetrics() {
	// Get metrics from Redis
	pipe := m.redisClient.Pipeline()
	processedCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_processed")
	failedCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_failed")
	pendingCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_pending")
	workersCmd := pipe.Get(m.ctx, m.metricsPrefix+"active_workers")
	pipe.Exec(m.ctx)
	
	processed, _ := processedCmd.Int64()
	failed, _ := failedCmd.Int64()
	pending, _ := pendingCmd.Int64()
	workers, _ := workersCmd.Int()
	
	logrus.WithFields(logrus.Fields{
		"active_workers":  workers,
		"total_processed": processed,
		"total_failed":    failed,
		"total_pending":   pending,
		"max_workers":     m.maxWorkers,
	}).Info("Broadcast manager metrics")
}

func (m *RedisOptimizedBroadcastManager) queueMonitorRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.monitorQueues()
		}
	}
}

func (m *RedisOptimizedBroadcastManager) monitorQueues() {
	// Get all queue sizes
	queueSizes, err := m.redisClient.HGetAll(m.ctx, "queue:sizes").Result()
	if err != nil {
		return
	}
	
	// Process retry queues
	m.workersMutex.RLock()
	for deviceID := range m.workers {
		retryKey := fmt.Sprintf("queue:retry:device:%s", deviceID)
		now := float64(time.Now().Unix())
		
		// Get messages ready for retry
		messages, err := m.redisClient.ZRangeByScoreWithScores(m.ctx, retryKey, &redis.ZRangeBy{
			Min: "0",
			Max: fmt.Sprintf("%f", now),
		}).Result()
		
		if err == nil && len(messages) > 0 {
			// Re-queue messages for retry
			for _, z := range messages {
				data := z.Member.(string)
				normalQueue := fmt.Sprintf("queue:device:%s", deviceID)
				
				pipe := m.redisClient.Pipeline()
				pipe.RPush(m.ctx, normalQueue, data)
				pipe.ZRem(m.ctx, retryKey, z.Member)
				pipe.Exec(m.ctx)
			}
			
			logrus.Infof("Re-queued %d messages for retry on device %s", len(messages), deviceID)
		}
	}
	m.workersMutex.RUnlock()
	
	// Log queue status
	totalQueued := 0
	for deviceID, sizeStr := range queueSizes {
		size, _ := redis.NewStringResult(sizeStr, nil).Int()
		totalQueued += size
		if size > 100 {
			logrus.Warnf("Device %s has %d messages queued", deviceID, size)
		}
	}
	
	if totalQueued > 0 {
		logrus.Infof("Total messages queued across all devices: %d", totalQueued)
	}
}

func (m *RedisOptimizedBroadcastManager) deadLetterQueueProcessor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.processDLQ()
		}
	}
}

func (m *RedisOptimizedBroadcastManager) processDLQ() {
	dlqKey := "queue:dead_letter"
	count, err := m.redisClient.LLen(m.ctx, dlqKey).Result()
	if err != nil || count == 0 {
		return
	}
	
	logrus.Warnf("Dead letter queue has %d failed messages", count)
	
	// Optional: Implement logic to retry DLQ messages or alert admins
}

// GetWorkerStatus returns comprehensive status from Redis
func (m *RedisOptimizedBroadcastManager) GetWorkerStatus() map[string]interface{} {
	// Get metrics from Redis
	pipe := m.redisClient.Pipeline()
	processedCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_processed")
	failedCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_failed")
	pendingCmd := pipe.Get(m.ctx, m.metricsPrefix+"total_pending")
	workersCmd := pipe.Get(m.ctx, m.metricsPrefix+"active_workers")
	pipe.Exec(m.ctx)
	
	processed, _ := processedCmd.Int64()
	failed, _ := failedCmd.Int64()
	pending, _ := pendingCmd.Int64()
	activeWorkers, _ := workersCmd.Int()
	
	// Get queue sizes
	queueSizes, _ := m.redisClient.HGetAll(m.ctx, "queue:sizes").Result()
	
	// Get worker details
	workers := make([]map[string]interface{}, 0)
	m.workersMutex.RLock()
	for deviceID, worker := range m.workers {
		// Get stats from Redis
		statsKey := fmt.Sprintf("worker:stats:%s", deviceID)
		stats, _ := m.redisClient.HGetAll(m.ctx, statsKey).Result()
		
		processedCount, _ := redis.NewStringResult(stats["processed"], nil).Int64()
		failedCount, _ := redis.NewStringResult(stats["failed"], nil).Int64()
		queueSize, _ := redis.NewStringResult(queueSizes[deviceID], nil).Int()
		
		// Get performance metrics
		perfKey := fmt.Sprintf("worker:performance:%s", deviceID)
		perf, _ := m.redisClient.HGetAll(m.ctx, perfKey).Result()
		
		totalTimeMs, _ := redis.NewStringResult(perf["total_time_ms"], nil).Float64()
		messageCount, _ := redis.NewStringResult(perf["message_count"], nil).Int64()
		
		avgProcessingTime := float64(0)
		if messageCount > 0 {
			avgProcessingTime = totalTimeMs / float64(messageCount)
		}
		
		workerInfo := map[string]interface{}{
			"device_id":           deviceID,
			"status":              worker.Status,
			"queue_size":          queueSize,
			"processed":           processedCount,
			"failed":              failedCount,
			"last_activity":       worker.LastActivity,
			"avg_processing_time": fmt.Sprintf("%.2f ms", avgProcessingTime),
		}
		workers = append(workers, workerInfo)
	}
	m.workersMutex.RUnlock()
	
	// Get DLQ count
	dlqCount, _ := m.redisClient.LLen(m.ctx, "queue:dead_letter").Result()
	
	return map[string]interface{}{
		"active_workers":    activeWorkers,
		"max_workers":       m.maxWorkers,
		"total_processed":   processed,
		"total_failed":      failed,
		"total_pending":     pending,
		"dead_letter_count": dlqCount,
		"workers":           workers,
		"redis_connected":   m.redisClient.Ping(m.ctx).Err() == nil,
	}
}

// Shutdown gracefully shuts down the broadcast manager
func (m *RedisOptimizedBroadcastManager) Shutdown() {
	logrus.Info("Shutting down Redis-optimized broadcast manager...")
	
	// Cancel context
	m.cancel()
	
	// Stop all workers
	m.workersMutex.Lock()
	for _, worker := range m.workers {
		worker.cancel()
	}
	m.workersMutex.Unlock()
	
	// Wait for all workers to finish
	m.wg.Wait()
	
	// Close Redis connection
	m.redisClient.Close()
	
	logrus.Info("Redis-optimized broadcast manager shutdown complete")
}

// Global instance with Redis support
var globalRedisManager *RedisOptimizedBroadcastManager
var redisOnce sync.Once

// GetRedisOptimizedBroadcastManager returns the global Redis-optimized instance
func GetRedisOptimizedBroadcastManager() *RedisOptimizedBroadcastManager {
	redisOnce.Do(func() {
		globalRedisManager = NewRedisOptimizedBroadcastManager()
	})
	return globalRedisManager
}

// For backward compatibility, redirect to Redis version
func GetBroadcastManager() *RedisOptimizedBroadcastManager {
	return GetRedisOptimizedBroadcastManager()
}
