package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// OptimizedDeviceManager handles 3000 devices efficiently
type OptimizedDeviceManager struct {
	redis         *redis.Client
	db            *sql.DB
	deviceCache   sync.Map  // Local cache to reduce Redis calls
	pipeline      redis.Pipeliner
	mu            sync.Mutex
}

// NewOptimizedDeviceManager creates manager for 3000 devices
func NewOptimizedDeviceManager(redisClient *redis.Client, db *sql.DB) *OptimizedDeviceManager {
	dm := &OptimizedDeviceManager{
		redis:    redisClient,
		db:       db,
		pipeline: redisClient.Pipeline(),
	}
	
	// Start cache refresher
	go dm.refreshDeviceCache()
	
	return dm
}

// BatchReserveDevices reserves multiple devices at once for better performance
func (dm *OptimizedDeviceManager) BatchReserveDevices(ctx context.Context, count int) ([]DeviceReservation, error) {
	reservations := make([]DeviceReservation, 0, count)
	
	// Get device list from cache
	devices := dm.getCachedDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices available")
	}

	// Use Redis pipeline for batch operations
	pipe := dm.redis.Pipeline()
	defer pipe.Close()

	type attempt struct {
		device   DeviceInfo
		lockKey  string
		acquired *redis.BoolCmd
	}

	attempts := make([]attempt, 0, len(devices))

	// Try to lock devices in batch
	for _, device := range devices {
		lockKey := fmt.Sprintf("d:l:%s", device.ID) // Shorter key for performance
		lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
		
		acquired := pipe.SetNX(ctx, lockKey, lockValue, 30*time.Second)
		attempts = append(attempts, attempt{
			device:   device,
			lockKey:  lockKey,
			acquired: acquired,
		})
		
		if len(attempts) >= count*2 { // Try 2x devices to ensure we get enough
			break
		}
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Check which locks we got and validate limits
	hourKey := time.Now().Format("2006010215")
	dayKey := time.Now().Format("20060102")
	
	for _, att := range attempts {
		if !att.acquired.Val() {
			continue // Device already locked
		}

		// Check limits using Lua script for atomicity
		luaScript := `
			local deviceID = ARGV[1]
			local hourKey = ARGV[2]
			local dayKey = ARGV[3]
			
			local hourCount = redis.call('HINCRBY', 'h:' .. hourKey, deviceID, 1)
			local dayCount = redis.call('HINCRBY', 'd:' .. dayKey, deviceID, 1)
			
			if hourCount > 80 or dayCount > 800 then
				redis.call('HINCRBY', 'h:' .. hourKey, deviceID, -1)
				redis.call('HINCRBY', 'd:' .. dayKey, deviceID, -1)
				return 0
			end
			
			return 1
		`

		result := dm.redis.Eval(ctx, luaScript, []string{}, att.device.ID, hourKey, dayKey).Val()
		
		if result == int64(1) {
			// Success! Device reserved
			reservation := DeviceReservation{
				DeviceID:  att.device.ID,
				LockKey:   att.lockKey,
				ExpiresAt: time.Now().Add(30 * time.Second),
				Release: func() {
					dm.redis.Del(ctx, att.lockKey)
				},
			}
			reservations = append(reservations, reservation)
			
			if len(reservations) >= count {
				break
			}
		} else {
			// Over limit, release lock
			dm.redis.Del(ctx, att.lockKey)
		}
	}

	// Set TTL on counter hashes
	dm.redis.Expire(ctx, "h:"+hourKey, 2*time.Hour)
	dm.redis.Expire(ctx, "d:"+dayKey, 25*time.Hour)

	if len(reservations) == 0 {
		return nil, fmt.Errorf("no devices available under limits")
	}

	return reservations, nil
}

// FastReserveDevice uses optimized logic for single device
func (dm *OptimizedDeviceManager) FastReserveDevice(ctx context.Context, preferredID string) (*DeviceReservation, error) {
	// Try preferred device first
	if preferredID != "" {
		if res, err := dm.tryFastReserve(ctx, preferredID); err == nil {
			return res, nil
		}
	}

	// Get least loaded device from cache
	deviceID := dm.getLeastLoadedDevice()
	if deviceID == "" {
		return nil, fmt.Errorf("no devices available")
	}

	return dm.tryFastReserve(ctx, deviceID)
}

// tryFastReserve attempts to reserve with minimal Redis calls
func (dm *OptimizedDeviceManager) tryFastReserve(ctx context.Context, deviceID string) (*DeviceReservation, error) {
	lockKey := fmt.Sprintf("d:l:%s", deviceID)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	
	// Use Lua script to do everything atomically
	luaScript := `
		local lockKey = KEYS[1]
		local deviceID = ARGV[1]
		local lockValue = ARGV[2]
		local hourKey = ARGV[3]
		local dayKey = ARGV[4]
		
		-- Try to acquire lock
		if redis.call('SET', lockKey, lockValue, 'NX', 'EX', 30) then
			-- Got lock, check limits
			local hourCount = redis.call('HINCRBY', 'h:' .. hourKey, deviceID, 1)
			local dayCount = redis.call('HINCRBY', 'd:' .. dayKey, deviceID, 1)
			
			if hourCount <= 80 and dayCount <= 800 then
				return {1, hourCount, dayCount} -- Success
			else
				-- Over limit, rollback
				redis.call('HINCRBY', 'h:' .. hourKey, deviceID, -1)
				redis.call('HINCRBY', 'd:' .. dayKey, deviceID, -1)
				redis.call('DEL', lockKey)
				return {0, hourCount-1, dayCount-1} -- Failed
			end
		end
		
		return {0, 0, 0} -- Lock failed
	`

	hourKey := time.Now().Format("2006010215")
	dayKey := time.Now().Format("20060102")
	
	result := dm.redis.Eval(ctx, luaScript, []string{lockKey}, 
		deviceID, lockValue, hourKey, dayKey).Val().([]interface{})
	
	success := result[0].(int64)
	if success == 1 {
		hourCount := result[1].(int64)
		dayCount := result[2].(int64)
		
		logrus.Debugf("Reserved device %s (h:%d/80, d:%d/800)", 
			deviceID, hourCount, dayCount)
		
		return &DeviceReservation{
			DeviceID:  deviceID,
			LockKey:   lockKey,
			LockValue: lockValue,
			ExpiresAt: time.Now().Add(30 * time.Second),
			Release: func() {
				// Only release if we still own the lock
				delScript := `
					if redis.call('GET', KEYS[1]) == ARGV[1] then
						return redis.call('DEL', KEYS[1])
					end
					return 0
				`
				dm.redis.Eval(ctx, delScript, []string{lockKey}, lockValue)
			},
		}, nil
	}

	return nil, fmt.Errorf("device %s unavailable or over limit", deviceID)
}

// refreshDeviceCache updates local cache periodically
func (dm *OptimizedDeviceManager) refreshDeviceCache() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		devices, err := dm.loadDevicesFromDB()
		if err != nil {
			logrus.Errorf("Failed to refresh device cache: %v", err)
			continue
		}
		
		// Update cache
		for _, device := range devices {
			dm.deviceCache.Store(device.ID, device)
		}
		
		// Remove offline devices
		dm.deviceCache.Range(func(key, value interface{}) bool {
			found := false
			for _, d := range devices {
				if d.ID == key.(string) {
					found = true
					break
				}
			}
			if !found {
				dm.deviceCache.Delete(key)
			}
			return true
		})
	}
}

// getCachedDevices returns devices from cache
func (dm *OptimizedDeviceManager) getCachedDevices() []DeviceInfo {
	devices := make([]DeviceInfo, 0, 3000)
	dm.deviceCache.Range(func(key, value interface{}) bool {
		devices = append(devices, value.(DeviceInfo))
		return true
	})
	return devices
}

// getLeastLoadedDevice uses cached stats to find best device
func (dm *OptimizedDeviceManager) getLeastLoadedDevice() string {
	ctx := context.Background()
	hourKey := time.Now().Format("2006010215")
	
	// Get all device loads in one call
	loads, _ := dm.redis.HGetAll(ctx, "h:"+hourKey).Result()
	
	var bestDevice string
	minLoad := 999
	
	dm.deviceCache.Range(func(key, value interface{}) bool {
		deviceID := key.(string)
		load := 0
		if val, ok := loads[deviceID]; ok {
			fmt.Sscanf(val, "%d", &load)
		}
		
		if load < minLoad && load < 80 {
			minLoad = load
			bestDevice = deviceID
		}
		
		return true
	})
	
	return bestDevice
}

// GetBulkStats returns stats for all devices efficiently
func (dm *OptimizedDeviceManager) GetBulkStats(ctx context.Context) (map[string]*DeviceStats, error) {
	hourKey := time.Now().Format("2006010215")
	dayKey := time.Now().Format("20060102")
	
	// Use pipeline to get all stats at once
	pipe := dm.redis.Pipeline()
	
	hourCmd := pipe.HGetAll(ctx, "h:"+hourKey)
	dayCmd := pipe.HGetAll(ctx, "d:"+dayKey)
	lockedCmd := pipe.Keys(ctx, "d:l:*")
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	
	hourCounts := hourCmd.Val()
	dayCounts := dayCmd.Val()
	lockedKeys := lockedCmd.Val()
	
	// Build locked set
	locked := make(map[string]bool)
	for _, key := range lockedKeys {
		deviceID := key[4:] // Remove "d:l:" prefix
		locked[deviceID] = true
	}
	
	// Build stats
	stats := make(map[string]*DeviceStats)
	dm.deviceCache.Range(func(key, value interface{}) bool {
		deviceID := key.(string)
		
		hourCount := 0
		if val, ok := hourCounts[deviceID]; ok {
			fmt.Sscanf(val, "%d", &hourCount)
		}
		
		dayCount := 0
		if val, ok := dayCounts[deviceID]; ok {
			fmt.Sscanf(val, "%d", &dayCount)
		}
		
		stats[deviceID] = &DeviceStats{
			DeviceID:      deviceID,
			MessagesHour:  hourCount,
			MessagesToday: dayCount,
			IsLocked:      locked[deviceID],
			Timestamp:     time.Now(),
		}
		
		return true
	})
	
	return stats, nil
}

// loadDevicesFromDB loads devices from database
func (dm *OptimizedDeviceManager) loadDevicesFromDB() ([]DeviceInfo, error) {
	query := `
		SELECT id, status
		FROM user_devices 
		WHERE status = 'online'
		LIMIT 3000
	`
	
	rows, err := dm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	devices := make([]DeviceInfo, 0, 3000)
	for rows.Next() {
		var device DeviceInfo
		if err := rows.Scan(&device.ID, &device.Status); err != nil {
			continue
		}
		devices = append(devices, device)
	}
	
	return devices, nil
}

// Cleanup old data periodically
func (dm *OptimizedDeviceManager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			// Clean old hour/day hashes
			oldHour := time.Now().Add(-3 * time.Hour).Format("2006010215")
			oldDay := time.Now().Add(-2 * 24 * time.Hour).Format("20060102")
			
			dm.redis.Del(ctx, "h:"+oldHour)
			dm.redis.Del(ctx, "d:"+oldDay)
		}
	}()
}

type DeviceReservation struct {
	DeviceID  string
	LockKey   string
	LockValue string
	ExpiresAt time.Time
	Release   func()
}