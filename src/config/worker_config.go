package config

// Worker configuration optimized for 200 users x 15 devices = 3,000 devices
const (
	// Worker Pool Settings
	MaxWorkersPerDevice   = 1      // Each device gets 1 dedicated worker
	MaxConcurrentWorkers  = 500    // Max concurrent workers system-wide
	WorkerQueueSize       = 1000   // Message queue size per worker
	WorkerHealthCheckSec  = 30     // Health check interval
	WorkerIdleTimeoutMin  = 10     // Worker idle timeout before shutdown
	
	// Message Processing
	DefaultMinDelaySeconds = 5     // Min delay between messages
	DefaultMaxDelaySeconds = 15    // Max delay between messages
	BatchSize             = 100    // Messages to process per batch
	RetryAttempts         = 3      // Retry failed messages
	RetryDelaySeconds     = 60     // Delay between retries
	
	// Campaign & Sequence Processing
	CampaignTriggerIntervalSec = 60  // Check for campaigns every minute
	SequenceTriggerIntervalSec = 300 // Process sequences every 5 minutes
	
	// Performance Tuning
	DatabaseMaxConnections = 200   // PostgreSQL connection pool
	DatabaseMaxIdleConns   = 50    // Idle connections
	DatabaseConnLifetime   = 3600  // Connection lifetime in seconds
	
	// Rate Limiting per Device
	MessagesPerMinute     = 20     // Max messages per minute per device
	MessagesPerHour       = 500    // Max messages per hour per device
	MessagesPerDay        = 5000   // Max messages per day per device
	
	// System Limits
	MaxDevicesPerUser     = 20     // Max devices per user (with buffer)
	MaxActiveUsers        = 250    // Max concurrent active users
	MaxTotalDevices       = 5000   // Max total devices system can handle
	
	// Memory Management
	GCPercent            = 50      // Garbage collection percentage
	MaxMemoryUsageGB     = 32      // Max memory usage
	WorkerMemoryLimitMB  = 100     // Memory limit per worker
	
	// Monitoring
	MetricsIntervalSec   = 10      // Metrics collection interval
	LogLevel             = "INFO"  // Log level
	EnableProfiling      = true    // Enable performance profiling
)

// GetWorkerConfig returns optimized worker configuration
func GetWorkerConfig() map[string]interface{} {
	return map[string]interface{}{
		"max_workers_per_device":   MaxWorkersPerDevice,
		"max_concurrent_workers":   MaxConcurrentWorkers,
		"worker_queue_size":        WorkerQueueSize,
		"worker_health_check_sec":  WorkerHealthCheckSec,
		"worker_idle_timeout_min":  WorkerIdleTimeoutMin,
		"default_min_delay_sec":    DefaultMinDelaySeconds,
		"default_max_delay_sec":    DefaultMaxDelaySeconds,
		"batch_size":               BatchSize,
		"retry_attempts":           RetryAttempts,
		"retry_delay_sec":          RetryDelaySeconds,
		"messages_per_minute":      MessagesPerMinute,
		"messages_per_hour":        MessagesPerHour,
		"messages_per_day":         MessagesPerDay,
	}
}

// CalculateOptimalWorkers calculates optimal worker count based on active devices
func CalculateOptimalWorkers(activeDevices int) int {
	// Formula: min(activeDevices, MaxConcurrentWorkers)
	if activeDevices > MaxConcurrentWorkers {
		return MaxConcurrentWorkers
	}
	return activeDevices
}

// GetDelayForDevice returns random delay between min and max for a device
func GetDelayForDevice(minDelay, maxDelay int) int {
	if minDelay >= maxDelay {
		return minDelay
	}
	return minDelay + (maxDelay-minDelay)/2 // Use middle value for consistency
}
