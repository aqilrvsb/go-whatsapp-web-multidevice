package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

const (
	// Redis queue keys - using unique names to avoid conflicts
	ultraCampaignQueuePrefix  = "ultra:queue:campaign:"
	ultraSequenceQueuePrefix  = "ultra:queue:sequence:"
	ultraDeadLetterPrefix     = "ultra:queue:dead:"
	ultraMetricsPrefix        = "ultra:metrics:"
	ultraRateLimitPrefix      = "ultra:ratelimit:"
	ultraWorkerStatusKey      = "ultra:workers"
	ultraWorkerLockPrefix     = "ultra:lock:"
	
	// Performance settings for 3000 devices
	maxConcurrentWorkers = 3000
	workerBatchSize      = 100
	queueCheckInterval   = 100 * time.Millisecond
	metricsInterval      = 5 * time.Second
	healthCheckInterval  = 30 * time.Second
	lockTTL              = 5 * time.Minute
)

// UltraScaleRedisManager is optimized for 3000+ devices
type UltraScaleRedisManager struct {
	redisClient   *redis.Client
	workers       map[string]*DeviceWorker
	workersMutex  sync.RWMutex
	activeWorkers int32
	
	// Performance optimization
	workerPools   map[int]*sync.Pool // Worker pools by priority
	metricsBatch  map[string]int64
	metricsMutex  sync.Mutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// UltraRedisMessage represents a message in Redis queue (unique name to avoid conflict)
type UltraRedisMessage struct {
	Message   domainBroadcast.BroadcastMessage `json:"message"`
	Priority  int                              `json:"priority"`
	Timestamp time.Time                        `json:"timestamp"`
	Retries   int                              `json:"retries"`
}

// NewUltraScaleRedisManager creates a manager optimized for 3000+ devices
func NewUltraScaleRedisManager() *UltraScaleRedisManager {
	// Initialize Redis client
	redisURL := config.RedisURL
	if redisURL == "" {
		redisHost := config.RedisHost
		redisPort := config.RedisPort
		redisPassword := config.RedisPassword
		
		if redisHost == "" {
			redisHost = "localhost"
		}
		if redisPort == "" {
			redisPort = "6379"
		}
		
		redisURL = fmt.Sprintf("redis://:%s@%s:%s/0", redisPassword, redisHost, redisPort)
	}
	
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logrus.Fatalf("Failed to parse Redis URL: %v", err)
	}
	
	// Optimize Redis client for high performance
	opt.PoolSize = 100           // Increase pool size for 3000 devices
	opt.MinIdleConns = 20        // Keep connections ready
	opt.MaxRetries = 3           // Retry on failure
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	
	redisClient := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}
	
	logrus.Info("Successfully connected to Redis (Ultra Scale Mode)")
	
	ctx, cancel := context.WithCancel(context.Background())
	manager := &UltraScaleRedisManager{
		redisClient:  redisClient,
		workers:      make(map[string]*DeviceWorker),
		workerPools:  make(map[int]*sync.Pool),
		metricsBatch: make(map[string]int64),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// Initialize worker pools for different priorities
	for priority := 1; priority <= 10; priority++ {
		manager.workerPools[priority] = &sync.Pool{
			New: func() interface{} {
				return &DeviceWorker{}
			},
		}
	}
	
	// Start manager routines
	manager.wg.Add(4)
	go manager.processQueues()
	go manager.monitorWorkers()
	go manager.cleanupDeadLetters()
	go manager.flushMetrics()
	
	logrus.Infof("Ultra Scale Redis Manager started - Ready for %d devices", maxConcurrentWorkers)
	return manager
}

// SendMessage implements the interface - optimized for high throughput
func (um *UltraScaleRedisManager) SendMessage(msg domainBroadcast.BroadcastMessage) error {
	// Use device-specific queue for better distribution
	var queueKey string
	if msg.CampaignID != nil {
		queueKey = fmt.Sprintf("%s%s", ultraCampaignQueuePrefix, msg.DeviceID)
	} else {
		queueKey = fmt.Sprintf("%s%s", ultraSequenceQueuePrefix, msg.DeviceID)
	}
	
	// Create Redis message
	redisMsg := UltraRedisMessage{
		Message:   msg,
		Priority:  getPriority(msg),
		Timestamp: time.Now(),
		Retries:   0,
	}
	
	data, err := json.Marshal(redisMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	
	// Push to device-specific queue
	if err := um.redisClient.LPush(um.ctx, queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push to queue: %v", err)
	}
	
	// Ensure worker exists for this device
	um.ensureWorker(msg.DeviceID)
	
	// Update metrics asynchronously
	go um.incrementMetricBatch("messages_queued")
	
	return nil
}

// ensureWorker creates a worker if it doesn't exist
func (um *UltraScaleRedisManager) ensureWorker(deviceID string) {
	um.workersMutex.RLock()
	_, exists := um.workers[deviceID]
	um.workersMutex.RUnlock()
	
	if !exists {
		// Try to acquire lock for this device
		lockKey := fmt.Sprintf("%s%s", ultraWorkerLockPrefix, deviceID)
		locked, err := um.redisClient.SetNX(um.ctx, lockKey, "1", lockTTL).Result()
		if err != nil || !locked {
			// Another server is handling this device
			return
		}
		
		// Check worker limit
		currentWorkers := atomic.LoadInt32(&um.activeWorkers)
		if currentWorkers >= maxConcurrentWorkers {
			logrus.Warnf("Worker limit reached (%d/%d), device %s will wait", 
				currentWorkers, maxConcurrentWorkers, deviceID)
			return
		}
		
		// Create worker
		um.workersMutex.Lock()
		if _, exists := um.workers[deviceID]; !exists {
			worker := um.createDeviceWorker(deviceID)
			um.workers[deviceID] = worker
			atomic.AddInt32(&um.activeWorkers, 1)
			
			// Start worker
			um.wg.Add(1)
			go um.runWorker(deviceID, worker)
			
			logrus.Infof("Started worker for device %s (total: %d)", 
				deviceID, atomic.LoadInt32(&um.activeWorkers))
		}
		um.workersMutex.Unlock()
	}
}

// createDeviceWorker creates a new device worker
func (um *UltraScaleRedisManager) createDeviceWorker(deviceID string) *DeviceWorker {
	// Get WhatsApp client
	clientManager := whatsapp.GetClientManager()
	client, err := clientManager.GetClient(deviceID)
	if err != nil || client == nil {
		logrus.Errorf("Failed to get WhatsApp client for device %s: %v", deviceID, err)
		return nil
	}
	
	// Create worker using the existing constructor
	// Pass default delays that will be overridden by message-specific delays
	worker := NewDeviceWorker(deviceID, client, 10, 30)
	
	return worker
}

// runWorker runs a device worker with optimizations
func (um *UltraScaleRedisManager) runWorker(deviceID string, worker *DeviceWorker) {
	defer um.wg.Done()
	defer func() {
		atomic.AddInt32(&um.activeWorkers, -1)
		um.workersMutex.Lock()
		delete(um.workers, deviceID)
		um.workersMutex.Unlock()
		
		// Release lock
		lockKey := fmt.Sprintf("%s%s", ultraWorkerLockPrefix, deviceID)
		um.redisClient.Del(um.ctx, lockKey)
		
		// Stop the worker
		worker.Stop()
	}()
	
	// Start the worker
	worker.Start()
	
	// Queue keys for this device
	campaignQueue := fmt.Sprintf("%s%s", ultraCampaignQueuePrefix, deviceID)
	sequenceQueue := fmt.Sprintf("%s%s", ultraSequenceQueuePrefix, deviceID)
	deadLetterQueue := fmt.Sprintf("%s%s", ultraDeadLetterPrefix, deviceID)
	
	idleCount := 0
	maxIdle := 100 // Stop worker after 100 idle checks (10 seconds)
	
	for {
		select {
		case <-um.ctx.Done():
			return
		case <-worker.ctx.Done():
			return
		default:
			// Try campaign queue first (higher priority)
			processed := false
			
			// Check campaign queue
			if msg := um.popMessage(campaignQueue); msg != nil {
				if err := um.processMessage(worker, msg); err != nil {
					um.handleFailedMessage(msg, deadLetterQueue, err)
				}
				processed = true
				idleCount = 0
			} else if msg := um.popMessage(sequenceQueue); msg != nil {
				// Then check sequence queue
				if err := um.processMessage(worker, msg); err != nil {
					um.handleFailedMessage(msg, deadLetterQueue, err)
				}
				processed = true
				idleCount = 0
			}
			
			if !processed {
				idleCount++
				if idleCount >= maxIdle {
					// No messages for 10 seconds, stop worker
					logrus.Infof("Worker for device %s idle, stopping", deviceID)
					return
				}
				time.Sleep(queueCheckInterval)
			}
			
			// Extend lock periodically
			if idleCount%20 == 0 {
				lockKey := fmt.Sprintf("%s%s", ultraWorkerLockPrefix, deviceID)
				um.redisClient.Expire(um.ctx, lockKey, lockTTL)
			}
		}
	}
}

// popMessage pops a message from queue
func (um *UltraScaleRedisManager) popMessage(queueKey string) *UltraRedisMessage {
	result, err := um.redisClient.RPop(um.ctx, queueKey).Result()
	if err != nil || result == "" {
		return nil
	}
	
	var msg UltraRedisMessage
	if err := json.Unmarshal([]byte(result), &msg); err != nil {
		logrus.Errorf("Failed to unmarshal message: %v", err)
		return nil
	}
	
	return &msg
}

// processMessage processes a single message
func (um *UltraScaleRedisManager) processMessage(worker *DeviceWorker, msg *UltraRedisMessage) error {
	start := time.Now()
	
	// Send the message
	err := worker.SendMessage(msg.Message)
	
	// Update metrics
	duration := time.Since(start)
	um.updateProcessingTime(msg.Message.DeviceID, duration)
	
	if err != nil {
		um.incrementMetricBatch("messages_failed")
		return err
	}
	
	um.incrementMetricBatch("messages_sent")
	return nil
}

// updateProcessingTime updates processing time metrics
func (um *UltraScaleRedisManager) updateProcessingTime(deviceID string, duration time.Duration) {
	key := fmt.Sprintf("%s%s:processing_time", ultraMetricsPrefix, deviceID)
	um.redisClient.LPush(um.ctx, key, duration.Milliseconds())
	um.redisClient.LTrim(um.ctx, key, 0, 99) // Keep last 100 values
}

// handleFailedMessage handles failed messages
func (um *UltraScaleRedisManager) handleFailedMessage(msg *UltraRedisMessage, deadLetterQueue string, err error) {
	msg.Retries++
	
	// If retries exceeded, move to dead letter
	if msg.Retries > 3 {
		data, _ := json.Marshal(msg)
		um.redisClient.LPush(um.ctx, deadLetterQueue, data)
		um.incrementMetricBatch("messages_dead_letter")
		return
	}
	
	// Otherwise, requeue with exponential backoff
	backoff := time.Duration(msg.Retries) * time.Minute
	time.Sleep(backoff)
	
	um.SendMessage(msg.Message)
}

// processQueues monitors all device queues
func (um *UltraScaleRedisManager) processQueues() {
	defer um.wg.Done()
	
	ticker := time.NewTicker(metricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-um.ctx.Done():
			return
		case <-ticker.C:
			// Check for devices with pending messages but no workers
			um.checkPendingQueues()
		}
	}
}

// checkPendingQueues checks for queues with messages but no workers
func (um *UltraScaleRedisManager) checkPendingQueues() {
	// Get all queue keys
	campaignQueues, _ := um.redisClient.Keys(um.ctx, ultraCampaignQueuePrefix+"*").Result()
	sequenceQueues, _ := um.redisClient.Keys(um.ctx, ultraSequenceQueuePrefix+"*").Result()
	
	allQueues := append(campaignQueues, sequenceQueues...)
	
	for _, queueKey := range allQueues {
		// Extract device ID from queue key
		var deviceID string
		if strings.HasPrefix(queueKey, ultraCampaignQueuePrefix) {
			deviceID = strings.TrimPrefix(queueKey, ultraCampaignQueuePrefix)
		} else {
			deviceID = strings.TrimPrefix(queueKey, ultraSequenceQueuePrefix)
		}
		
		// Check if queue has messages
		count, _ := um.redisClient.LLen(um.ctx, queueKey).Result()
		if count > 0 {
			// Ensure worker exists
			um.ensureWorker(deviceID)
		}
	}
}

// monitorWorkers monitors worker health
func (um *UltraScaleRedisManager) monitorWorkers() {
	defer um.wg.Done()
	
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-um.ctx.Done():
			return
		case <-ticker.C:
			um.performHealthCheck()
		}
	}
}

// performHealthCheck checks worker health
func (um *UltraScaleRedisManager) performHealthCheck() {
	um.workersMutex.RLock()
	workerCount := len(um.workers)
	deviceIDs := make([]string, 0, workerCount)
	for deviceID := range um.workers {
		deviceIDs = append(deviceIDs, deviceID)
	}
	um.workersMutex.RUnlock()
	
	// Log statistics
	logrus.Infof("Health check: %d active workers out of %d max", 
		workerCount, maxConcurrentWorkers)
	
	// Check and extend locks
	for _, deviceID := range deviceIDs {
		lockKey := fmt.Sprintf("%s%s", ultraWorkerLockPrefix, deviceID)
		um.redisClient.Expire(um.ctx, lockKey, lockTTL)
	}
	
	// Update global metrics
	um.redisClient.Set(um.ctx, "ultra:stats:active_workers", workerCount, 0)
	um.redisClient.Set(um.ctx, "ultra:stats:max_workers", maxConcurrentWorkers, 0)
}

// flushMetrics periodically flushes batched metrics to Redis
func (um *UltraScaleRedisManager) flushMetrics() {
	defer um.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-um.ctx.Done():
			return
		case <-ticker.C:
			um.flushMetricsBatch()
		}
	}
}

// flushMetricsBatch flushes batched metrics
func (um *UltraScaleRedisManager) flushMetricsBatch() {
	um.metricsMutex.Lock()
	batch := um.metricsBatch
	um.metricsBatch = make(map[string]int64)
	um.metricsMutex.Unlock()
	
	// Use pipeline for efficiency
	pipe := um.redisClient.Pipeline()
	for metric, count := range batch {
		key := fmt.Sprintf("%s%s", ultraMetricsPrefix, metric)
		pipe.IncrBy(um.ctx, key, count)
	}
	pipe.Exec(um.ctx)
}

// incrementMetricBatch increments a metric in batch
func (um *UltraScaleRedisManager) incrementMetricBatch(metric string) {
	um.metricsMutex.Lock()
	um.metricsBatch[metric]++
	um.metricsMutex.Unlock()
}

// cleanupDeadLetters cleans up old dead letter messages
func (um *UltraScaleRedisManager) cleanupDeadLetters() {
	defer um.wg.Done()
	
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-um.ctx.Done():
			return
		case <-ticker.C:
			// Clean up old dead letters
			deadLetterQueues, _ := um.redisClient.Keys(um.ctx, ultraDeadLetterPrefix+"*").Result()
			for _, queue := range deadLetterQueues {
				count, _ := um.redisClient.LLen(um.ctx, queue).Result()
				if count > 1000 {
					// Keep only recent 1000
					um.redisClient.LTrim(um.ctx, queue, 0, 999)
				}
			}
		}
	}
}

// GetWorkerStatus returns status for a specific device
func (um *UltraScaleRedisManager) GetWorkerStatus(deviceID string) (domainBroadcast.WorkerStatus, bool) {
	// Check in-memory first
	um.workersMutex.RLock()
	worker, exists := um.workers[deviceID]
	um.workersMutex.RUnlock()
	
	if exists && worker != nil {
		return worker.GetStatus(), true
	}
	
	// Check Redis for distributed status
	data, err := um.redisClient.HGet(um.ctx, ultraWorkerStatusKey, deviceID).Result()
	if err != nil {
		return domainBroadcast.WorkerStatus{}, false
	}
	
	var status map[string]interface{}
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return domainBroadcast.WorkerStatus{}, false
	}
	
	// Get queue lengths
	campaignQueue := fmt.Sprintf("%s%s", ultraCampaignQueuePrefix, deviceID)
	sequenceQueue := fmt.Sprintf("%s%s", ultraSequenceQueuePrefix, deviceID)
	
	campaignCount, _ := um.redisClient.LLen(um.ctx, campaignQueue).Result()
	sequenceCount, _ := um.redisClient.LLen(um.ctx, sequenceQueue).Result()
	
	return domainBroadcast.WorkerStatus{
		DeviceID:     deviceID,
		Status:       status["status"].(string),
		QueueSize:    int(campaignCount + sequenceCount),
		LastActivity: time.Unix(int64(status["last_activity"].(float64)), 0),
	}, true
}

// GetAllWorkerStatus returns status for all workers
func (um *UltraScaleRedisManager) GetAllWorkerStatus() []domainBroadcast.WorkerStatus {
	var statuses []domainBroadcast.WorkerStatus
	
	// Get all workers from Redis (includes workers on other servers)
	workers, _ := um.redisClient.HGetAll(um.ctx, ultraWorkerStatusKey).Result()
	
	for deviceID, data := range workers {
		var status map[string]interface{}
		if err := json.Unmarshal([]byte(data), &status); err != nil {
			continue
		}
		
		// Get queue lengths
		campaignQueue := fmt.Sprintf("%s%s", ultraCampaignQueuePrefix, deviceID)
		sequenceQueue := fmt.Sprintf("%s%s", ultraSequenceQueuePrefix, deviceID)
		
		campaignCount, _ := um.redisClient.LLen(um.ctx, campaignQueue).Result()
		sequenceCount, _ := um.redisClient.LLen(um.ctx, sequenceQueue).Result()
		
		statuses = append(statuses, domainBroadcast.WorkerStatus{
			DeviceID:     deviceID,
			Status:       status["status"].(string),
			QueueSize:    int(campaignCount + sequenceCount),
			LastActivity: time.Unix(int64(status["last_activity"].(float64)), 0),
		})
	}
	
	return statuses
}

// StopAllWorkers stops all workers gracefully
func (um *UltraScaleRedisManager) StopAllWorkers() error {
	logrus.Info("Stopping all workers...")
	
	um.workersMutex.Lock()
	for deviceID, worker := range um.workers {
		worker.Stop()
		logrus.Infof("Stopped worker for device %s", deviceID)
	}
	um.workersMutex.Unlock()
	
	// Wait for all to stop
	time.Sleep(2 * time.Second)
	
	return nil
}

// ResumeFailedWorkers resumes workers that have failed
func (um *UltraScaleRedisManager) ResumeFailedWorkers() error {
	logrus.Info("Resuming failed workers...")
	
	// Check all queues for pending messages
	um.checkPendingQueues()
	
	return nil
}

// Stop gracefully stops the manager
func (um *UltraScaleRedisManager) Stop() {
	logrus.Info("Stopping Ultra Scale Redis Manager...")
	
	// Stop accepting new messages
	um.cancel()
	
	// Stop all workers
	um.StopAllWorkers()
	
	// Wait for all goroutines
	um.wg.Wait()
	
	// Close Redis connection
	um.redisClient.Close()
	
	logrus.Info("Ultra Scale Redis Manager stopped")
}

// Helper function to determine message priority
func getPriority(msg domainBroadcast.BroadcastMessage) int {
	if msg.CampaignID != nil {
		return 1 // High priority for campaigns
	}
	return 5 // Normal priority for sequences
}

// Interface compliance check
var _ BroadcastManagerInterface = (*UltraScaleRedisManager)(nil)
