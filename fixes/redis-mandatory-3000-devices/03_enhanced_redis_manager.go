// Enhanced UltraScaleRedisManager for 3000+ devices with rate limiting

package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

const (
	// Redis queue keys
	ultraCampaignQueuePrefix  = "ultra:queue:campaign:"
	ultraSequenceQueuePrefix  = "ultra:queue:sequence:"
	ultraDeadLetterPrefix     = "ultra:queue:dead:"
	ultraMetricsPrefix        = "ultra:metrics:"
	ultraRateLimitPrefix      = "ultra:ratelimit:"
	ultraWorkerStatusKey      = "ultra:workers"
	ultraWorkerLockPrefix     = "ultra:lock:"
	
	// Performance settings for 3000+ devices
	maxConcurrentWorkers = 3000
	workerBatchSize      = 100
	queueCheckInterval   = 50 * time.Millisecond  // Faster for 3000 devices
	metricsInterval      = 5 * time.Second
	healthCheckInterval  = 30 * time.Second
	lockTTL              = 5 * time.Minute
	
	// Rate limiting per device
	maxMessagesPerMinute = 60    // WhatsApp limit
	maxMessagesPerHour   = 1000  // WhatsApp limit
	maxMessagesPerDay    = 10000 // Safety limit
)

// Enhanced UltraScaleRedisManager
type UltraScaleRedisManager struct {
	redisClient   *redis.Client
	workers       map[string]*DeviceWorker
	workersMutex  sync.RWMutex
	activeWorkers int32
	
	// Rate limiting
	rateLimiter   *DeviceRateLimiter
	
	// Performance optimization
	workerPools   map[int]*sync.Pool
	metricsBatch  map[string]int64
	metricsMutex  sync.Mutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// DeviceRateLimiter handles rate limiting for each device
type DeviceRateLimiter struct {
	redis   *redis.Client
	mu      sync.RWMutex
	limits  map[string]*DeviceLimit
}

type DeviceLimit struct {
	DeviceID         string
	MessagesMinute   int64
	MessagesHour     int64
	MessagesToday    int64
	LastResetMinute  time.Time
	LastResetHour    time.Time
	LastResetDay     time.Time
}

// VerifyConnection verifies Redis connection
func (m *UltraScaleRedisManager) VerifyConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return m.redisClient.Ping(ctx).Err()
}

// CheckRateLimit checks if device can send more messages
func (rl *DeviceRateLimiter) CheckRateLimit(deviceID string) (bool, string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limit, exists := rl.limits[deviceID]
	if !exists {
		limit = &DeviceLimit{
			DeviceID:        deviceID,
			LastResetMinute: time.Now(),
			LastResetHour:   time.Now(),
			LastResetDay:    time.Now(),
		}
		rl.limits[deviceID] = limit
	}
	
	now := time.Now()
	
	// Reset counters if needed
	if now.Sub(limit.LastResetMinute) > time.Minute {
		limit.MessagesMinute = 0
		limit.LastResetMinute = now
	}
	
	if now.Sub(limit.LastResetHour) > time.Hour {
		limit.MessagesHour = 0
		limit.LastResetHour = now
	}
	
	if now.Sub(limit.LastResetDay) > 24*time.Hour {
		limit.MessagesToday = 0
		limit.LastResetDay = now
	}
	
	// Check limits
	if limit.MessagesMinute >= maxMessagesPerMinute {
		return false, fmt.Sprintf("Rate limit: %d messages/minute reached", maxMessagesPerMinute)
	}
	
	if limit.MessagesHour >= maxMessagesPerHour {
		return false, fmt.Sprintf("Rate limit: %d messages/hour reached", maxMessagesPerHour)
	}
	
	if limit.MessagesToday >= maxMessagesPerDay {
		return false, fmt.Sprintf("Rate limit: %d messages/day reached", maxMessagesPerDay)
	}
	
	// Increment counters
	limit.MessagesMinute++
	limit.MessagesHour++
	limit.MessagesToday++
	
	// Update Redis with current limits
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", ultraRateLimitPrefix, deviceID)
	data, _ := json.Marshal(limit)
	rl.redis.Set(ctx, key, data, 24*time.Hour)
	
	return true, ""
}
