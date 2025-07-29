package config

import (
	"os"
	"strconv"
	"time"
)

// BroadcastConfig holds configuration for broadcast system
type BroadcastConfig struct {
	// Pool cleanup duration after completion (default: 5 minutes)
	PoolCleanupDuration time.Duration
	
	// Maximum workers per pool (default: 5000 for high volume)
	MaxWorkersPerPool int
	
	// Maximum pools per user (default: 50)
	MaxPoolsPerUser int
	
	// Worker queue size (default: 5000 for 5K messages)
	WorkerQueueSize int
	
	// Completion check interval (default: 10 seconds)
	CompletionCheckInterval time.Duration
	
	// Progress log interval (default: 30 seconds)
	ProgressLogInterval time.Duration
}

// GetBroadcastConfig returns broadcast configuration from environment
func GetBroadcastConfig() *BroadcastConfig {
	config := &BroadcastConfig{
		PoolCleanupDuration:     5 * time.Minute,  // Default 5 minutes
		MaxWorkersPerPool:       5000,             // Increased from 3000
		MaxPoolsPerUser:         50,               // Increased from 10
		WorkerQueueSize:         5000,             // Increased from 1000
		CompletionCheckInterval: 10 * time.Second,
		ProgressLogInterval:     30 * time.Second,
	}
	
	// Override with environment variables if set
	if val := os.Getenv("BROADCAST_POOL_CLEANUP_MINUTES"); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil {
			config.PoolCleanupDuration = time.Duration(minutes) * time.Minute
		}
	}
	
	if val := os.Getenv("BROADCAST_MAX_WORKERS_PER_POOL"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.MaxWorkersPerPool = num
		}
	}
	
	if val := os.Getenv("BROADCAST_MAX_POOLS_PER_USER"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.MaxPoolsPerUser = num
		}
	}
	
	if val := os.Getenv("BROADCAST_WORKER_QUEUE_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil {
			config.WorkerQueueSize = size
		}
	}
	
	if val := os.Getenv("BROADCAST_COMPLETION_CHECK_SECONDS"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil {
			config.CompletionCheckInterval = time.Duration(seconds) * time.Second
		}
	}
	
	if val := os.Getenv("BROADCAST_PROGRESS_LOG_SECONDS"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil {
			config.ProgressLogInterval = time.Duration(seconds) * time.Second
		}
	}
	
	return config
}
