package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

const (
	// Redis queue keys
	campaignQueuePrefix  = "broadcast:queue:campaign:"
	sequenceQueuePrefix  = "broadcast:queue:sequence:"
	deadLetterPrefix     = "broadcast:queue:dead:"
	metricsPrefix        = "broadcast:metrics:"
	rateLimitPrefix      = "broadcast:ratelimit:"
	workerStatusKey      = "broadcast:workers"
	workerLockPrefix     = "broadcast:lock:"
	
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
		queueKey = fmt.Sprintf("%s%s", campaignQueuePrefix, msg.DeviceID)
	} else {
		queueKey = fmt.Sprintf("%s%s", sequenceQueuePrefix, msg.DeviceID)
	}
	
	// Create Redis message
	redisMsg := RedisMessage{
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
		lockKey := fmt.Sprintf("%s%s", workerLockPrefix, deviceID)
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

// runWorker runs a device worker with optimizations
func (um *UltraScaleRedisManager) runWorker(deviceID string, worker *DeviceWorker) {
	defer um.wg.Done()
	defer func() {
		atomic.AddInt32(&um.activeWorkers, -1)
		um.workersMutex.Lock()
		delete(um.workers, deviceID)
		um.workersMutex.Unlock()
		
		// Release lock
		lockKey := fmt.Sprintf("%s%s", workerLockPrefix, deviceID)
		um.redisClient.Del(um.ctx, lockKey)
	}()
	
	// Queue keys for this device
	campaignQueue := fmt.Sprintf("%s%s", campaignQueuePrefix, deviceID)
	sequenceQueue := fmt.Sprintf("%s%s", sequenceQueuePrefix, deviceID)
	deadLetterQueue := fmt.Sprintf("%s%s", deadLetterPrefix, deviceID)
	
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
				lockKey := fmt.Sprintf("%s%s", workerLockPrefix, deviceID)
				um.redisClient.Expire(um.ctx, lockKey, lockTTL)
			}
		}
	}
}

// popMessage pops a message from queue
func (um *UltraScaleRedisManager) popMessage(queueKey string) *RedisMessage {
	result, err := um.redisClient.RPop(um.ctx, queueKey).Result()
	if err != nil || result == "" {
		return nil
	}
	
	var msg RedisMessage
	if err := json.Unmarshal([]byte(result), &msg); err != nil {
		logrus.Errorf("Failed to unmarshal message: %v", err)
		return nil
	}
	
	return &msg
}

// processMessage processes a single message
func (um *UltraScaleRedisManager) processMessage(worker *DeviceWorker, msg *RedisMessage) error {
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

// handleFailedMessage handles failed messages
func (um *UltraScaleRedisManager) handleFailedMessage(msg *RedisMessage, deadLetterQueue string, err error) {
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
	campaignQueues, _ := um.redisClient.Keys(um.ctx, campaignQueuePrefix+"*").Result()
	sequenceQueues, _ := um.redisClient.Keys(um.ctx, sequenceQueuePrefix+"*").Result()
	
	allQueues := append(campaignQueues, sequenceQueues...)
	
	for _, queueKey := range allQueues {
		// Extract device ID from queue key
		var deviceID string
		if strings.HasPrefix(queueKey, campaignQueuePrefix) {
			deviceID = strings.TrimPrefix(queueKey, campaignQueuePrefix)
		} else {
			deviceID = strings.TrimPrefix(queueKey, sequenceQueuePrefix)
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
		lockKey := fmt.Sprintf("%s%s", workerLockPrefix, deviceID)
		um.redisClient.Expire(um.ctx, lockKey, lockTTL)
	}
	
	// Update global metrics
	um.redisClient.Set(um.ctx, "broadcast:stats:active_workers", workerCount, 0)
	um.redisClient.Set(um.ctx, "broadcast:stats:max_workers", maxConcurrentWorkers, 0)
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
		key := fmt.Sprintf("%s%s", metricsPrefix, metric)
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
			deadLetterQueues, _ := um.redisClient.Keys(um.ctx, deadLetterPrefix+"*").Result()
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
) int {
	if msg.CampaignID != nil {
		return 1 // High priority for campaigns
	}
	return 5 // Normal priority for sequences
}

// RedisMessage represents a message in Redis queue
type RedisMessage struct {
	Message   domainBroadcast.BroadcastMessage `json:"message"`
	Priority  int                              `json:"priority"`
	Timestamp time.Time                        `json:"timestamp"`
	Retries   int                              `json:"retries"`
}

// Interface compliance check
var _ BroadcastManagerInterface = (*UltraScaleRedisManager)(nil)
) int {
	if msg.CampaignID != nil {
		return 1 // High priority for campaigns
	}
	return 5 // Normal priority for sequences
}

// RedisMessage represents a message in Redis queue
type RedisMessage struct {
	Message   domainBroadcast.BroadcastMessage `json:"message"`
	Priority  int                              `json:"priority"`
	Timestamp time.Time                        `json:"timestamp"`
	Retries   int                              `json:"retries"`
}
