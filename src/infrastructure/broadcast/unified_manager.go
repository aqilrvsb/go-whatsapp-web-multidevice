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
		// Check if Redis is available - try multiple env vars
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			redisURL = os.Getenv("redis_url")
		}
		if redisURL == "" {
			redisURL = os.Getenv("RedisURL")
		}
		
		// Log what we found
		logrus.Infof("Checking for Redis - REDIS_URL env: '%s'", redisURL)
		
		// For now, let's disable Redis to get the app working
		// We'll enable it once we fix the connection issue
		if redisURL != "" && redisURL != "redis://default:${{REDIS_PASSWORD}}@${{RAILWAY_PRIVATE_DOMAIN}}:6379" {
			// Only use Redis if we have a real URL, not a template
			if redisURL != "redis://[::1]:6379" && redisURL != "redis://localhost:6379" {
				logrus.Info("Valid Redis URL found, but using in-memory manager for now")
				// unifiedManager = NewRedisOptimizedBroadcastManager()
			}
		}
		
		// Use in-memory manager for now
		logrus.Info("Using in-memory broadcast manager")
		unifiedManager = NewBasicBroadcastManager()
	})
	return unifiedManager
}
