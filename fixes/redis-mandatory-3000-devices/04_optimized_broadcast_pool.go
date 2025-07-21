// Optimized Broadcast Pool for 3000+ devices
package broadcast

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// OptimizedBroadcastPool manages workers for campaigns and sequences
type OptimizedBroadcastPool struct {
	poolID        string
	broadcastType string // "campaign" or "sequence"
	broadcastID   string
	userID        string
	
	// Worker management
	deviceWorkers map[string]*DeviceWorker // One worker per device
	workerMutex   sync.RWMutex
	
	// Redis
	redisClient *redis.Client
	queueKey    string
	
	// Rate limiting
	rateLimiter *DeviceRateLimiter
	
	// Statistics
	totalMessages    int64
	processedCount   int64
	failedCount      int64
	skippedCount     int64
	startTime        time.Time
	completionTime   *time.Time
	
	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	manager    *UltraScaleBroadcastManager // Reference to prevent zombie pools
}

// DeviceWorker handles messages for a specific device
type DeviceWorker struct {
	workerID      string
	deviceID      string
	pool          *OptimizedBroadcastPool
	messageSender *WhatsAppMessageSender
	
	// Statistics
	processedCount int64
	failedCount    int64
	lastActivity   time.Time
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// CreateOptimizedPool creates a new broadcast pool
func (manager *UltraScaleBroadcastManager) CreateOptimizedPool(
	broadcastType string, 
	broadcastID string, 
	userID string,
) (*OptimizedBroadcastPool, error) {
	
	poolID := fmt.Sprintf("%s:%s", broadcastType, broadcastID)
	
	// Check if pool already exists
	manager.mu.RLock()
	if existingPool, exists := manager.pools[poolID]; exists {
		manager.mu.RUnlock()
		logrus.Infof("Pool %s already exists, reusing", poolID)
		return existingPool, nil
	}
	manager.mu.RUnlock()
	
	// Create new pool
	ctx, cancel := context.WithCancel(context.Background())
	pool := &OptimizedBroadcastPool{
		poolID:        poolID,
		broadcastType: broadcastType,
		broadcastID:   broadcastID,
		userID:        userID,
		deviceWorkers: make(map[string]*DeviceWorker),
		redisClient:   manager.redisClient,
		queueKey:      fmt.Sprintf("ultra:queue:%s:%s", broadcastType, broadcastID),
		rateLimiter:   manager.rateLimiter,
		startTime:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
		manager:       manager,
	}
	
	// Register pool
	manager.mu.Lock()
	manager.pools[poolID] = pool
	manager.mu.Unlock()
	
	// Start pool monitor
	pool.wg.Add(1)
	go pool.monitorPool()
	
	logrus.Infof("Created optimized pool %s for user %s", poolID, userID)
	return pool, nil
}

// QueueMessage adds a message to the pool's Redis queue
func (pool *OptimizedBroadcastPool) QueueMessage(msg *domainBroadcast.BroadcastMessage) error {
	// Increment total messages
	atomic.AddInt64(&pool.totalMessages, 1)
	
	// Check rate limit for device
	canSend, reason := pool.rateLimiter.CheckRateLimit(msg.DeviceID)
	if !canSend {
		atomic.AddInt64(&pool.skippedCount, 1)
		logrus.Warnf("Rate limit hit for device %s: %s", msg.DeviceID, reason)
		
		// Update message status
		db := database.GetDB()
		db.Exec(`UPDATE broadcast_messages SET status = 'rate_limited', 
				error_message = $1 WHERE id = $2`, reason, msg.ID)
		return nil
	}
	
	// Ensure worker exists for device
	pool.ensureWorkerForDevice(msg.DeviceID)
	
	// Queue to Redis
	ctx := context.Background()
	msgData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Use device-specific queue for better distribution
	deviceQueueKey := fmt.Sprintf("%s:device:%s", pool.queueKey, msg.DeviceID)
	if err := pool.redisClient.LPush(ctx, deviceQueueKey, msgData).Err(); err != nil {
		return fmt.Errorf("failed to queue message: %w", err)
	}
	
	logrus.Debugf("Queued message to %s for device %s", deviceQueueKey, msg.DeviceID)
	return nil
}
