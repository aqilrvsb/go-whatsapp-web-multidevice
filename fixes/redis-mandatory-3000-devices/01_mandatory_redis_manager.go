package broadcast

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	
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

// GetBroadcastManager returns Redis-based manager (MANDATORY for 3000+ devices)
func GetBroadcastManager() BroadcastManagerInterface {
	umOnce.Do(func() {
		// Initialize config first
		config.InitEnvironment()
		
		// Check if Redis is available
		redisURL := config.RedisURL
		if redisURL == "" {
			redisURL = os.Getenv("REDIS_URL")
		}
		
		// Validate Redis URL
		if redisURL == "" || 
		   strings.Contains(redisURL, "${{") || 
		   (!strings.Contains(redisURL, "redis://") && !strings.Contains(redisURL, "rediss://")) {
			// Redis is MANDATORY for production use
			logrus.Fatal("REDIS IS REQUIRED: Please set REDIS_URL environment variable. " +
				"This system requires Redis for handling 3000+ devices efficiently. " +
				"Example: redis://user:password@host:port/db")
		}
		
		// Initialize Redis manager with enhanced features
		logrus.Info("Initializing Ultra Scale Redis Manager for 3000+ devices")
		logrus.Infof("Redis URL: %s", maskRedisPassword(redisURL))
		
		manager := NewUltraScaleRedisManager()
		
		// Verify Redis connection
		if err := manager.VerifyConnection(); err != nil {
			logrus.Fatalf("Failed to connect to Redis: %v. Please check your REDIS_URL", err)
		}
		
		logrus.Info("✅ Redis connected successfully - System ready for 3000+ devices")
		unifiedManager = manager
	})
	return unifiedManager
}

// maskRedisPassword masks the password in Redis URL for logging
func maskRedisPassword(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 && strings.Contains(parts[0], ":") {
			credsEnd := strings.LastIndex(parts[0], ":")
			if credsEnd > 0 {
				masked := parts[0][:credsEnd] + ":****"
				return masked + "@" + parts[1]
			}
		}
	}
	return url
}
