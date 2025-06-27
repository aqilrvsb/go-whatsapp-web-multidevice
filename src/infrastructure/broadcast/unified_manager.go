package broadcast

import (
	"os"
	"sync"
	
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/sirupsen/logrus"
)

// BroadcastManagerInterface defines the interface for broadcast managers
type BroadcastManagerInterface interface {
	SendMessage(msg domainBroadcast.BroadcastMessage) error
	GetWorkerStatus(deviceID string) (domainBroadcast.WorkerStatus, bool)
	GetAllWorkerStatus() []domainBroadcast.WorkerStatus
	StopAllWorkers() error
	ResumeFailedWorkers() error
}

var (
	unifiedManager BroadcastManagerInterface
	umOnce        sync.Once
)

// GetBroadcastManager returns the appropriate broadcast manager based on configuration
func GetBroadcastManager() BroadcastManagerInterface {
	umOnce.Do(func() {
		// Check if Redis is available
		redisURL := os.Getenv("REDIS_URL")
		if redisURL != "" {
			logrus.Info("Redis URL found, initializing Redis-based broadcast manager")
			unifiedManager = NewRedisOptimizedBroadcastManager()
		} else {
			logrus.Info("No Redis URL found, using in-memory broadcast manager")
			unifiedManager = NewBasicBroadcastManager()
		}
	})
	return unifiedManager
}
