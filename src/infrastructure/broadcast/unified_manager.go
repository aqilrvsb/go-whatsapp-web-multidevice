package broadcast

import (
	"os"
	"strings"
	"sync"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
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

// GetBroadcastManager returns Redis manager (MANDATORY for production)
func GetBroadcastManager() BroadcastManagerInterface {
	umOnce.Do(func() {
		// Initialize config first
		config.InitEnvironment()
		
		// Check if Redis is available
		redisURL := config.RedisURL
		if redisURL == "" {
			redisURL = os.Getenv("REDIS_URL")
		}
		
		// Redis is MANDATORY for 3000+ devices
		if redisURL == "" || 
		   strings.Contains(redisURL, "${{") || 
		   (!strings.Contains(redisURL, "redis://") && !strings.Contains(redisURL, "rediss://")) {
			logrus.Fatal("REDIS IS REQUIRED: Please set REDIS_URL environment variable. " +
				"This system requires Redis for handling 3000+ devices efficiently.")
		}
		
		logrus.Info("Initializing Ultra Scale Redis Manager for 3000+ devices")
		unifiedManager = NewUltraScaleRedisManager()
	})
	return unifiedManager
}
