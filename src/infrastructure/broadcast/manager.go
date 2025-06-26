package broadcast

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
)

// BroadcastManager manages broadcasting for all devices
type BroadcastManager struct {
	workers      map[string]*DeviceWorker // deviceID -> worker
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	maxWorkers   int
}

// DeviceWorker handles broadcasting for a single device
type DeviceWorker struct {
	deviceID      string
	client        *whatsmeow.Client
	minDelay      int
	maxDelay      int
	queue         chan domainBroadcast.BroadcastMessage
	ctx           context.Context
	cancel        context.CancelFunc
	isRunning     bool
	mu            sync.Mutex
	messagesSent  int
	lastSentTime  time.Time
}

var (
	broadcastManager *BroadcastManager
	bmOnce          sync.Once
)

// GetBroadcastManager returns singleton broadcast manager
func GetBroadcastManager() *BroadcastManager {
	bmOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		broadcastManager = &BroadcastManager{
			workers:    make(map[string]*DeviceWorker),
			ctx:        ctx,
			cancel:     cancel,
			maxWorkers: 100, // Limit concurrent workers
		}
		
		// Start manager
		go broadcastManager.Run()
		
		// Start queue processor
		go broadcastManager.ProcessQueue()
	})
	return broadcastManager
}

// Run manages the broadcast manager
func (bm *BroadcastManager) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bm.ctx.Done():
			bm.StopAllWorkers()
			return
		case <-ticker.C:
			bm.CheckWorkerHealth()
		}
	}
}

// ProcessQueue processes messages from database queue
func (bm *BroadcastManager) ProcessQueue() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bm.ctx.Done():
			return
		case <-ticker.C:
			bm.processQueueBatch()
		}
	}
}

// processQueueBatch processes a batch of queued messages
func (bm *BroadcastManager) processQueueBatch() {
	repo := repository.GetBroadcastRepository()
	messages, err := repo.GetPendingMessages(100) // Get 100 messages at a time
	
	if err != nil {
		logrus.Errorf("Failed to get pending messages: %v", err)
		return
	}
	
	for _, msg := range messages {
		// Get or create worker for device
		worker := bm.GetOrCreateWorker(msg.DeviceID)
		if worker != nil {
			// Send to worker queue
			select {
			case worker.queue <- msg:
				// Mark as processing
				repo.UpdateMessageStatus(msg.ID, "processing")
			default:
				// Queue is full, skip this message
				logrus.Warnf("Queue full for device %s", msg.DeviceID)
			}
		}
	}
}

// GetOrCreateWorker gets or creates a worker for a device
func (bm *BroadcastManager) GetOrCreateWorker(deviceID string) *DeviceWorker {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	// Check if worker exists
	if worker, exists := bm.workers[deviceID]; exists && worker.isRunning {
		return worker
	}
	
	// Check worker limit
	if len(bm.workers) >= bm.maxWorkers {
		logrus.Warnf("Worker limit reached, cannot create worker for device %s", deviceID)
		return nil
	}
	
	// Get device info
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get device %s: %v", deviceID, err)
		return nil
	}
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get client for device %s: %v", deviceID, err)
		return nil
	}
	
	// Create worker
	ctx, cancel := context.WithCancel(bm.ctx)
	worker := &DeviceWorker{
		deviceID:     deviceID,
		client:       client,
		minDelay:     device.MinDelaySeconds,
		maxDelay:     device.MaxDelaySeconds,
		queue:        make(chan domainBroadcast.BroadcastMessage, 1000), // Buffer 1000 messages
		ctx:          ctx,
		cancel:       cancel,
		isRunning:    true,
		lastSentTime: time.Now().Add(-time.Minute), // Allow immediate first send
	}
	
	// Start worker
	go worker.Run()
	
	bm.workers[deviceID] = worker
	logrus.Infof("Created worker for device %s with delay %d-%d seconds", deviceID, device.MinDelaySeconds, device.MaxDelaySeconds)
	
	return worker
}
// StopAllWorkers stops all device workers
func (bm *BroadcastManager) StopAllWorkers() {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	for deviceID, worker := range bm.workers {
		worker.Stop()
		delete(bm.workers, deviceID)
		logrus.Infof("Stopped worker for device %s", deviceID)
	}
}

// CheckWorkerHealth checks health of all workers
func (bm *BroadcastManager) CheckWorkerHealth() {
	bm.mu.RLock()
	workers := make([]*DeviceWorker, 0, len(bm.workers))
	for _, worker := range bm.workers {
		workers = append(workers, worker)
	}
	bm.mu.RUnlock()
	
	for _, worker := range workers {
		if !worker.IsHealthy() {
			logrus.Warnf("Worker for device %s is unhealthy, restarting...", worker.deviceID)
			bm.RestartWorker(worker.deviceID)
		}
	}
}

// RestartWorker restarts a worker
func (bm *BroadcastManager) RestartWorker(deviceID string) {
	bm.mu.Lock()
	if worker, exists := bm.workers[deviceID]; exists {
		worker.Stop()
		delete(bm.workers, deviceID)
	}
	bm.mu.Unlock()
	
	// Create new worker
	bm.GetOrCreateWorker(deviceID)
}

// QueueMessage queues a message for broadcasting
func (bm *BroadcastManager) QueueMessage(deviceID string, msg domainBroadcast.BroadcastMessage) error {
	// Save to database queue
	repo := repository.GetBroadcastRepository()
	return repo.QueueMessage(msg)
}

// GetWorkerStats gets statistics for all workers
func (bm *BroadcastManager) GetWorkerStats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_workers": len(bm.workers),
		"workers": make([]map[string]interface{}, 0),
	}
	
	for deviceID, worker := range bm.workers {
		workerStats := map[string]interface{}{
			"device_id":     deviceID,
			"is_running":    worker.isRunning,
			"messages_sent": worker.messagesSent,
			"queue_size":    len(worker.queue),
			"last_sent":     worker.lastSentTime,
		}
		stats["workers"] = append(stats["workers"].([]map[string]interface{}), workerStats)
	}
	
	return stats
}