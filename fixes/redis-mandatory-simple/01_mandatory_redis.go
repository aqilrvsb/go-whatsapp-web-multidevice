package broadcast

import (
	"os"
	"strings"
	"sync"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/sirupsen/logrus"
)

var (
	unifiedManager BroadcastManagerInterface
	umOnce        sync.Once
)

// GetBroadcastManager returns Redis manager (MANDATORY)
func GetBroadcastManager() BroadcastManagerInterface {
	umOnce.Do(func() {
		config.InitEnvironment()
		
		redisURL := config.RedisURL
		if redisURL == "" {
			redisURL = os.Getenv("REDIS_URL")
		}
		
		// Redis is MANDATORY
		if redisURL == "" || !strings.Contains(redisURL, "redis") {
			logrus.Fatal("REDIS IS REQUIRED: Set REDIS_URL environment variable")
		}
		
		logrus.Info("Initializing Redis Manager for 3000+ devices")
		unifiedManager = NewUltraScaleRedisManager()
	})
	return unifiedManager
}
