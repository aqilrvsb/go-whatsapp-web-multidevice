package broadcast

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// BroadcastWorkerPool manages workers per broadcast (campaign/sequence)
type BroadcastWorkerPool struct {
	broadcastID   string
	broadcastType string // "campaign" or "sequence"
	workers       map[string]*BroadcastWorker // key: deviceID
	maxWorkers    int
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	redisClient   *redis.Client
	
	// Statistics
	totalMessages    int64
	processedCount   int64
	failedCount      int64
	startTime        time.Time
	completionTime   *time.Time
}

// BroadcastWorker handles messages for a specific device within a broadcast
type BroadcastWorker struct {
	poolID        string
	deviceID      string
	broadcastID   string
	broadcastType string
	whatsappClient interface{} // WhatsApp client
	
	// Message processing
	messageQueue  chan *domainBroadcast.BroadcastMessage
	status        string
	processedCount int64
	failedCount    int64
	
	// Control
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	lastActivity  time.Time
}

// UltraScaleBroadcastManager manages broadcast-specific worker pools
type UltraScaleBroadcastManager struct {
	pools         map[string]*BroadcastWorkerPool // key: broadcastType:broadcastID
	redisClient   *redis.Client
	mu            sync.RWMutex
	
	// Global limits
	maxPoolsPerUser      int
	maxWorkersPerPool    int
	maxDevicesPerWorker  int
}

var (
	ultraBroadcastManager *UltraScaleBroadcastManager
	ultraOnce            sync.Once
)

// GetUltraScaleBroadcastManager returns singleton instance
func GetUltraScaleBroadcastManager() *UltraScaleBroadcastManager {
	ultraOnce.Do(func() {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
			Password: config.RedisPassword,
			DB:       0,
		})
		
		ultraBroadcastManager = &UltraScaleBroadcastManager{
			pools:               make(map[string]*BroadcastWorkerPool),
			redisClient:        redisClient,
			maxPoolsPerUser:    10,    // 10 simultaneous broadcasts per user
			maxWorkersPerPool:  3000,  // Support 3000 devices
			maxDevicesPerWorker: 1,    // 1:1 device to worker for maximum throughput
		}
		
		logrus.Info("UltraScale Broadcast Manager initialized for 3000+ devices")
	})
	
	return ultraBroadcastManager
}

// StartBroadcastPool creates a worker pool for a specific broadcast
func (ubm *UltraScaleBroadcastManager) StartBroadcastPool(broadcastType string, broadcastID string, userID string) (*BroadcastWorkerPool, error) {
	poolKey := fmt.Sprintf("%s:%s", broadcastType, broadcastID)
	
	ubm.mu.Lock()
	defer ubm.mu.Unlock()
	
	// Check if pool already exists
	if pool, exists := ubm.pools[poolKey]; exists {
		return pool, nil
	}
	
	// Create new pool
	ctx, cancel := context.WithCancel(context.Background())
	pool := &BroadcastWorkerPool{
		broadcastID:   broadcastID,
		broadcastType: broadcastType,
		workers:       make(map[string]*BroadcastWorker),
		maxWorkers:    ubm.maxWorkersPerPool,
		ctx:           ctx,
		cancel:        cancel,
		redisClient:   ubm.redisClient,
		startTime:     time.Now(),
	}
	
	ubm.pools[poolKey] = pool
	
	// Start pool monitor
	go pool.monitor()
	
	logrus.Infof("Started broadcast pool for %s:%s with capacity for %d devices", 
		broadcastType, broadcastID, pool.maxWorkers)
	
	return pool, nil
}

// QueueMessageToBroadcast queues a message to the appropriate broadcast pool
func (ubm *UltraScaleBroadcastManager) QueueMessageToBroadcast(msg *domainBroadcast.BroadcastMessage) error {
	var poolKey string
	
	// Determine which pool this message belongs to
	if msg.CampaignID != nil {
		poolKey = fmt.Sprintf("campaign:%d", *msg.CampaignID)
	} else if msg.SequenceID != nil {
		poolKey = fmt.Sprintf("sequence:%s", *msg.SequenceID)
	} else {
		return fmt.Errorf("message has no campaign or sequence ID")
	}
	
	ubm.mu.RLock()
	pool, exists := ubm.pools[poolKey]
	ubm.mu.RUnlock()
	
	if !exists {
		// Create pool if it doesn't exist
		broadcastType := "campaign"
		broadcastID := fmt.Sprintf("%d", *msg.CampaignID)
		if msg.SequenceID != nil {
			broadcastType = "sequence"
			broadcastID = *msg.SequenceID
		}
		
		var err error
		pool, err = ubm.StartBroadcastPool(broadcastType, broadcastID, msg.UserID)
		if err != nil {
			return err
		}
	}
	
	// Queue to pool
	return pool.QueueMessage(msg)
}

// QueueMessage adds a message to the broadcast pool
func (bwp *BroadcastWorkerPool) QueueMessage(msg *domainBroadcast.BroadcastMessage) error {
	atomic.AddInt64(&bwp.totalMessages, 1)
	
	// Get or create worker for this device
	worker := bwp.getOrCreateWorker(msg.DeviceID)
	
	// Queue to worker
	select {
	case worker.messageQueue <- msg:
		// Update message status to queued
		db := database.GetDB()
		_, err := db.Exec(`UPDATE broadcast_messages SET status = 'queued' WHERE id = $1`, msg.ID)
		if err != nil {
			logrus.Errorf("Failed to update message status: %v", err)
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout queueing message to worker")
	}
}

// getOrCreateWorker gets existing worker or creates new one
func (bwp *BroadcastWorkerPool) getOrCreateWorker(deviceID string) *BroadcastWorker {
	bwp.mu.RLock()
	worker, exists := bwp.workers[deviceID]
	bwp.mu.RUnlock()
	
	if exists {
		return worker
	}
	
	bwp.mu.Lock()
	defer bwp.mu.Unlock()
	
	// Double-check after acquiring write lock
	if worker, exists = bwp.workers[deviceID]; exists {
		return worker
	}
	
	// Create new worker
	ctx, cancel := context.WithCancel(bwp.ctx)
	worker = &BroadcastWorker{
		poolID:        fmt.Sprintf("%s:%s", bwp.broadcastType, bwp.broadcastID),
		deviceID:      deviceID,
		broadcastID:   bwp.broadcastID,
		broadcastType: bwp.broadcastType,
		messageQueue:  make(chan *domainBroadcast.BroadcastMessage, 1000), // Large buffer
		status:        "idle",
		ctx:           ctx,
		cancel:        cancel,
		lastActivity:  time.Now(),
	}
	
	// Get WhatsApp client
	// TODO: Get actual WhatsApp client from device manager
	// For now, just create the worker without client
	// device, _ := whatsapp.GetDeviceByID(deviceID)
	// if device != nil {
	//     worker.whatsappClient = device.Client
	// }
	
	bwp.workers[deviceID] = worker
	
	// Start worker
	go worker.process()
	
	return worker
}

// process handles messages for this worker
func (bw *BroadcastWorker) process() {
	logrus.Infof("Broadcast worker started for %s device %s", bw.poolID, bw.deviceID)
	
	for {
		select {
		case <-bw.ctx.Done():
			logrus.Infof("Broadcast worker stopped for %s device %s", bw.poolID, bw.deviceID)
			return
			
		case msg := <-bw.messageQueue:
			bw.processMessage(msg)
		}
	}
}

// processMessage sends a single message
func (bw *BroadcastWorker) processMessage(msg *domainBroadcast.BroadcastMessage) {
	bw.mu.Lock()
	bw.status = "processing"
	bw.lastActivity = time.Now()
	bw.mu.Unlock()
	
	// Send via WhatsApp
	err := bw.sendWhatsAppMessage(msg)
	
	db := database.GetDB()
	if err != nil {
		atomic.AddInt64(&bw.failedCount, 1)
		// Update status to failed
		db.Exec(`UPDATE broadcast_messages SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`, 
			err.Error(), msg.ID)
		logrus.Errorf("Failed to send message %s: %v", msg.ID, err)
	} else {
		atomic.AddInt64(&bw.processedCount, 1)
		// Update status to sent
		db.Exec(`UPDATE broadcast_messages SET status = 'sent', sent_at = NOW() WHERE id = $1`, msg.ID)
		
		// Apply delay if configured
		if msg.MinDelay > 0 && msg.MaxDelay > 0 {
			delay := calculateRandomDelay(msg.MinDelay, msg.MaxDelay)
			time.Sleep(delay)
		}
	}
	
	bw.mu.Lock()
	bw.status = "idle"
	bw.mu.Unlock()
}

// sendWhatsAppMessage sends message via WhatsApp
func (bw *BroadcastWorker) sendWhatsAppMessage(msg *domainBroadcast.BroadcastMessage) error {
	if bw.whatsappClient == nil {
		return fmt.Errorf("no WhatsApp client for device %s", bw.deviceID)
	}
	
	// Implementation depends on your WhatsApp client
	// This is a placeholder
	return nil
}

// monitor checks pool health and completion
func (bwp *BroadcastWorkerPool) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bwp.ctx.Done():
			return
			
		case <-ticker.C:
			bwp.checkCompletion()
		}
	}
}

// checkCompletion checks if all messages are processed
func (bwp *BroadcastWorkerPool) checkCompletion() {
	processed := atomic.LoadInt64(&bwp.processedCount)
	failed := atomic.LoadInt64(&bwp.failedCount)
	total := atomic.LoadInt64(&bwp.totalMessages)
	
	if processed+failed >= total && total > 0 {
		// All messages processed
		bwp.mu.Lock()
		if bwp.completionTime == nil {
			now := time.Now()
			bwp.completionTime = &now
			duration := now.Sub(bwp.startTime)
			
			logrus.Infof("Broadcast %s:%s completed in %v - Total: %d, Sent: %d, Failed: %d",
				bwp.broadcastType, bwp.broadcastID, duration, total, processed, failed)
			
			// Update campaign/sequence status
			db := database.GetDB()
			if bwp.broadcastType == "campaign" {
				db.Exec(`UPDATE campaigns SET status = 'finished' WHERE id = $1`, bwp.broadcastID)
			}
		}
		bwp.mu.Unlock()
	}
}

// GetPoolStatus returns status of a broadcast pool
func (ubm *UltraScaleBroadcastManager) GetPoolStatus(broadcastType, broadcastID string) map[string]interface{} {
	poolKey := fmt.Sprintf("%s:%s", broadcastType, broadcastID)
	
	ubm.mu.RLock()
	pool, exists := ubm.pools[poolKey]
	ubm.mu.RUnlock()
	
	if !exists {
		return map[string]interface{}{
			"status": "not_found",
		}
	}
	
	pool.mu.RLock()
	workerCount := len(pool.workers)
	pool.mu.RUnlock()
	
	return map[string]interface{}{
		"broadcast_id":   pool.broadcastID,
		"broadcast_type": pool.broadcastType,
		"status":        "active",
		"workers":       workerCount,
		"total_messages": atomic.LoadInt64(&pool.totalMessages),
		"processed":     atomic.LoadInt64(&pool.processedCount),
		"failed":        atomic.LoadInt64(&pool.failedCount),
		"start_time":    pool.startTime,
		"duration":      time.Since(pool.startTime).Seconds(),
	}
}

// calculateRandomDelay calculates random delay between min and max
func calculateRandomDelay(minSeconds, maxSeconds int) time.Duration {
	if minSeconds >= maxSeconds {
		return time.Duration(minSeconds) * time.Second
	}
	// Simple random between min and max
	delay := minSeconds + (int(time.Now().UnixNano()) % (maxSeconds - minSeconds))
	return time.Duration(delay) * time.Second
}
