package broadcast

import (
	"context"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// BroadcastManager manages broadcasting for all devices
type BroadcastManager struct {
	workers      map[string]*DeviceWorker // deviceID -> worker
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	maxWorkers   int
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
	
	// Process messages for each active worker
	bm.mu.RLock()
	deviceIDs := make([]string, 0, len(bm.workers))
	for deviceID := range bm.workers {
		deviceIDs = append(deviceIDs, deviceID)
	}
	bm.mu.RUnlock()
	
	for _, deviceID := range deviceIDs {
		messages, err := repo.GetPendingMessages(deviceID, 50)
		if err != nil {
			logrus.Errorf("Failed to get pending messages for device %s: %v", deviceID, err)
			continue
		}
		
		for _, msg := range messages {
			// Get worker for device
			worker := bm.GetOrCreateWorker(msg.DeviceID)
			if worker != nil {
				// Send to worker queue
				err := worker.QueueMessage(msg)
				if err != nil {
					logrus.Warnf("Failed to queue message: %v", err)
					repo.UpdateMessageStatus(msg.ID, "failed", err.Error())
				} else {
					// Mark as processing
					repo.UpdateMessageStatus(msg.ID, "processing", "")
				}
			}
		}
	}
}

// GetOrCreateWorker gets or creates a worker for a device
func (bm *BroadcastManager) GetOrCreateWorker(deviceID string) *DeviceWorker {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	// Check if worker exists
	if worker, exists := bm.workers[deviceID]; exists {
		status := worker.GetStatus()
		if status.Status != "stopped" {
			return worker
		}
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
	worker := NewDeviceWorker(deviceID, client, device.MinDelaySeconds, device.MaxDelaySeconds)
	
	// Start worker
	worker.Start()
	
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
	deviceIDs := make([]string, 0, len(bm.workers))
	for deviceID, worker := range bm.workers {
		status := worker.GetStatus()
		// Check if worker is stuck (no activity for 10 minutes while having queued messages)
		if status.Status == "processing" && time.Since(status.LastActivity) > 10*time.Minute && status.QueueSize > 0 {
			deviceIDs = append(deviceIDs, deviceID)
			logrus.Warnf("Worker for device %s appears stuck, will restart...", deviceID)
		}
	}
	bm.mu.RUnlock()
	
	// Restart stuck workers
	for _, deviceID := range deviceIDs {
		bm.RestartWorker(deviceID)
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
		status := worker.GetStatus()
		workerStats := map[string]interface{}{
			"device_id":     deviceID,
			"status":        status.Status,
			"queue_size":    status.QueueSize,
			"processed":     status.ProcessedCount,
			"failed":        status.FailedCount,
			"last_activity": status.LastActivity,
		}
		stats["workers"] = append(stats["workers"].([]map[string]interface{}), workerStats)
	}
	
	return stats
}