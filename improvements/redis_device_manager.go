package usecase

import (
	"context"
	"fmt"
	"time"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// DeviceManager handles device selection with Redis-based locking
type DeviceManager struct {
	redis *redis.Client
	db    *sql.DB
}

// NewDeviceManager creates a new device manager
func NewDeviceManager(redisClient *redis.Client, db *sql.DB) *DeviceManager {
	return &DeviceManager{
		redis: redisClient,
		db:    db,
	}
}

// ReserveDeviceAtomic atomically reserves a device without race conditions
func (dm *DeviceManager) ReserveDeviceAtomic(ctx context.Context, preferredDeviceID string) (string, func(), error) {
	// Try preferred device first if specified
	if preferredDeviceID != "" {
		if deviceID, release, err := dm.tryReserveDevice(ctx, preferredDeviceID); err == nil {
			return deviceID, release, nil
		}
	}

	// Get all available devices
	devices, err := dm.getAvailableDevices()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get devices: %w", err)
	}

	// Try to reserve each device
	for _, device := range devices {
		if deviceID, release, err := dm.tryReserveDevice(ctx, device.ID); err == nil {
			return deviceID, release, nil
		}
	}

	return "", nil, fmt.Errorf("no available devices")
}

// tryReserveDevice attempts to reserve a specific device
func (dm *DeviceManager) tryReserveDevice(ctx context.Context, deviceID string) (string, func(), error) {
	// Keys for Redis
	lockKey := fmt.Sprintf("device:lock:%s", deviceID)
	hourCountKey := fmt.Sprintf("device:hour:%s:%s", deviceID, time.Now().Format("2006010215"))
	dayCountKey := fmt.Sprintf("device:day:%s:%s", deviceID, time.Now().Format("20060102"))
	
	// Try to acquire lock with 30-second expiration
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired := dm.redis.SetNX(ctx, lockKey, lockValue, 30*time.Second).Val()
	
	if !acquired {
		return "", nil, fmt.Errorf("device %s is locked", deviceID)
	}

	// Check current usage
	hourCount := dm.redis.Incr(ctx, hourCountKey).Val()
	dayCount := dm.redis.Incr(ctx, dayCountKey).Val()

	// Set expiration on counters
	dm.redis.Expire(ctx, hourCountKey, 2*time.Hour)
	dm.redis.Expire(ctx, dayCountKey, 25*time.Hour)

	// Check limits
	if hourCount > 80 || dayCount > 800 {
		// Over limit, rollback and release
		dm.redis.Decr(ctx, hourCountKey)
		dm.redis.Decr(ctx, dayCountKey)
		dm.redis.Del(ctx, lockKey)
		return "", nil, fmt.Errorf("device %s over limit: hour=%d, day=%d", deviceID, hourCount, dayCount)
	}

	// Create release function
	release := func() {
		// Only delete if we still own the lock
		if val := dm.redis.Get(ctx, lockKey).Val(); val == lockValue {
			dm.redis.Del(ctx, lockKey)
		}
		logrus.Debugf("Released device %s", deviceID)
	}

	logrus.Debugf("Reserved device %s (hour: %d/80, day: %d/800)", deviceID, hourCount, dayCount)
	return deviceID, release, nil
}

// getAvailableDevices returns online devices sorted by load
func (dm *DeviceManager) getAvailableDevices() ([]DeviceInfo, error) {
	query := `
		SELECT 
			d.id,
			d.status,
			COALESCE(dlb.messages_hour, 0) as messages_hour,
			COALESCE(dlb.messages_today, 0) as messages_today
		FROM user_devices d
		LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
		WHERE d.status = 'online'
		ORDER BY 
			COALESCE(dlb.messages_hour, 0) ASC,
			COALESCE(dlb.messages_today, 0) ASC
	`

	rows, err := dm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []DeviceInfo
	for rows.Next() {
		var device DeviceInfo
		if err := rows.Scan(&device.ID, &device.Status, &device.MessagesHour, &device.MessagesToday); err != nil {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDeviceStats returns current stats for a device from Redis
func (dm *DeviceManager) GetDeviceStats(ctx context.Context, deviceID string) (*DeviceStats, error) {
	hourCountKey := fmt.Sprintf("device:hour:%s:%s", deviceID, time.Now().Format("2006010215"))
	dayCountKey := fmt.Sprintf("device:day:%s:%s", deviceID, time.Now().Format("20060102"))
	
	hourCount, _ := dm.redis.Get(ctx, hourCountKey).Int()
	dayCount, _ := dm.redis.Get(ctx, dayCountKey).Int()
	
	// Check if device is locked
	lockKey := fmt.Sprintf("device:lock:%s", deviceID)
	isLocked := dm.redis.Exists(ctx, lockKey).Val() > 0
	
	return &DeviceStats{
		DeviceID:     deviceID,
		MessagesHour: hourCount,
		MessagesToday: dayCount,
		IsLocked:     isLocked,
		Timestamp:    time.Now(),
	}, nil
}

// ResetDeviceCounters resets all device counters (for testing/manual reset)
func (dm *DeviceManager) ResetDeviceCounters(ctx context.Context, deviceID string) error {
	pattern := fmt.Sprintf("device:*:%s:*", deviceID)
	
	// Find all keys for this device
	keys, err := dm.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	// Delete all keys
	if len(keys) > 0 {
		return dm.redis.Del(ctx, keys...).Err()
	}
	
	return nil
}

// GetAllDeviceStats returns stats for all devices
func (dm *DeviceManager) GetAllDeviceStats(ctx context.Context) (map[string]*DeviceStats, error) {
	// Get all devices
	devices, err := dm.getAvailableDevices()
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]*DeviceStats)
	for _, device := range devices {
		deviceStats, _ := dm.GetDeviceStats(ctx, device.ID)
		stats[device.ID] = deviceStats
	}
	
	return stats, nil
}

// MonitorDeviceHealth monitors device health and disables problematic devices
func (dm *DeviceManager) MonitorDeviceHealth(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			dm.checkDeviceHealth(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// checkDeviceHealth checks and updates device health status
func (dm *DeviceManager) checkDeviceHealth(ctx context.Context) {
	stats, err := dm.GetAllDeviceStats(ctx)
	if err != nil {
		logrus.Errorf("Failed to get device stats: %v", err)
		return
	}
	
	for deviceID, stat := range stats {
		// Check if device is hitting limits too often
		failureKey := fmt.Sprintf("device:failures:%s", deviceID)
		failures := dm.redis.Incr(ctx, failureKey).Val()
		
		// If device is at 90% of limit, increment failure count
		if stat.MessagesHour >= 72 { // 90% of 80
			dm.redis.Expire(ctx, failureKey, 1*time.Hour)
			
			if failures > 5 {
				// Disable device temporarily
				dm.disableDevice(ctx, deviceID, 30*time.Minute)
				logrus.Warnf("Device %s disabled due to high failure rate", deviceID)
			}
		} else {
			// Reset failure count if device is healthy
			dm.redis.Del(ctx, failureKey)
		}
	}
}

// disableDevice temporarily disables a device
func (dm *DeviceManager) disableDevice(ctx context.Context, deviceID string, duration time.Duration) error {
	disableKey := fmt.Sprintf("device:disabled:%s", deviceID)
	return dm.redis.Set(ctx, disableKey, "1", duration).Err()
}

// Struct definitions
type DeviceInfo struct {
	ID            string
	Status        string
	MessagesHour  int
	MessagesToday int
}

type DeviceStats struct {
	DeviceID      string
	MessagesHour  int
	MessagesToday int
	IsLocked      bool
	Timestamp     time.Time
}