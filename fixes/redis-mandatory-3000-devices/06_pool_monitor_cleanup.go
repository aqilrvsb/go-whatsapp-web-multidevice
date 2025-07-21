// Pool monitoring and cleanup
package broadcast

// monitorPool monitors pool completion and triggers cleanup
func (pool *OptimizedBroadcastPool) monitorPool() {
	defer pool.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	idleChecks := 0
	maxIdleChecks := 30 // 5 minutes
	
	for {
		select {
		case <-ticker.C:
			// Get current stats
			processed := atomic.LoadInt64(&pool.processedCount)
			failed := atomic.LoadInt64(&pool.failedCount)
			skipped := atomic.LoadInt64(&pool.skippedCount)
			total := atomic.LoadInt64(&pool.totalMessages)
			
			// Check active workers
			pool.workerMutex.RLock()
			activeWorkers := 0
			for _, worker := range pool.deviceWorkers {
				if time.Since(worker.lastActivity) < 30*time.Second {
					activeWorkers++
				}
			}
			workerCount := len(pool.deviceWorkers)
			pool.workerMutex.RUnlock()
			
			// Log status
			logrus.Infof("Pool %s status: Total=%d, Processed=%d, Failed=%d, Skipped=%d, Workers=%d/%d", 
				pool.poolID, total, processed, failed, skipped, activeWorkers, workerCount)
			
			// Check completion
			totalHandled := processed + failed + skipped
			if totalHandled >= total && total > 0 && activeWorkers == 0 {
				idleChecks++
				
				if idleChecks >= maxIdleChecks {
					// Pool is complete
					now := time.Now()
					pool.completionTime = &now
					
					// Update database status
					pool.updateCompletionStatus()
					
					// Log completion
					duration := pool.completionTime.Sub(pool.startTime)
					messagesPerMinute := float64(processed) / duration.Minutes()
					
					logrus.Infof("🎉 Pool %s COMPLETED: Duration=%v, Rate=%.2f msg/min, Success=%.2f%%", 
						pool.poolID, duration, messagesPerMinute, 
						float64(processed)/float64(total)*100)
					
					// Schedule cleanup
					go pool.scheduleCleanup()
					return
				}
			} else {
				idleChecks = 0 // Reset if still active
			}
			
		case <-pool.ctx.Done():
			logrus.Infof("Pool %s monitor stopped", pool.poolID)
			return
		}
	}
}

// updateCompletionStatus updates campaign/sequence status in database
func (pool *OptimizedBroadcastPool) updateCompletionStatus() {
	db := database.GetDB()
	
	if pool.broadcastType == "campaign" {
		_, err := db.Exec(`
			UPDATE campaigns 
			SET status = 'completed',
				updated_at = NOW(),
				completed_at = NOW()
			WHERE id = $1`, pool.broadcastID)
		
		if err != nil {
			logrus.Errorf("Failed to update campaign status: %v", err)
		}
	}
	
	// Log final statistics
	ctx := context.Background()
	statsKey := fmt.Sprintf("ultra:stats:%s", pool.poolID)
	stats := map[string]interface{}{
		"total_messages": pool.totalMessages,
		"processed":      pool.processedCount,
		"failed":         pool.failedCount,
		"skipped":        pool.skippedCount,
		"start_time":     pool.startTime.Unix(),
		"end_time":       pool.completionTime.Unix(),
		"duration_sec":   pool.completionTime.Sub(pool.startTime).Seconds(),
	}
	
	for k, v := range stats {
		pool.redisClient.HSet(ctx, statsKey, k, fmt.Sprintf("%v", v))
	}
	pool.redisClient.Expire(ctx, statsKey, 7*24*time.Hour) // Keep stats for 7 days
}

// scheduleCleanup schedules pool cleanup after delay
func (pool *OptimizedBroadcastPool) scheduleCleanup() {
	// Wait before cleanup (configurable, default 5 minutes)
	cleanupDelay := 5 * time.Minute
	
	logrus.Infof("Scheduling cleanup for pool %s after %v", pool.poolID, cleanupDelay)
	time.Sleep(cleanupDelay)
	
	pool.cleanup()
}

// cleanup properly cleans up the pool and prevents zombie pools
func (pool *OptimizedBroadcastPool) cleanup() {
	logrus.Infof("Starting cleanup for pool %s", pool.poolID)
	
	// Cancel context to stop all workers
	pool.cancel()
	
	// Stop all workers
	pool.workerMutex.Lock()
	for deviceID, worker := range pool.deviceWorkers {
		worker.cancel()
		logrus.Debugf("Stopped worker for device %s", deviceID)
	}
	pool.workerMutex.Unlock()
	
	// Wait for all workers to finish
	done := make(chan struct{})
	go func() {
		pool.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		logrus.Debugf("All workers stopped for pool %s", pool.poolID)
	case <-time.After(30 * time.Second):
		logrus.Warnf("Timeout waiting for workers to stop in pool %s", pool.poolID)
	}
	
	// Clean Redis queues
	ctx := context.Background()
	pattern := fmt.Sprintf("%s*", pool.queueKey)
	keys, err := pool.redisClient.Keys(ctx, pattern).Result()
	if err == nil && len(keys) > 0 {
		if err := pool.redisClient.Del(ctx, keys...).Err(); err != nil {
			logrus.Warnf("Failed to clean Redis queues: %v", err)
		} else {
			logrus.Debugf("Cleaned %d Redis queues for pool %s", len(keys), pool.poolID)
		}
	}
	
	// CRITICAL: Remove pool from manager to prevent zombie pools
	pool.manager.mu.Lock()
	delete(pool.manager.pools, pool.poolID)
	pool.manager.mu.Unlock()
	
	logrus.Infof("✅ Pool %s cleaned up successfully (removed from registry)", pool.poolID)
}

// ForceCleanup forces immediate cleanup (for emergency use)
func (pool *OptimizedBroadcastPool) ForceCleanup() {
	logrus.Warnf("FORCE cleanup requested for pool %s", pool.poolID)
	pool.cleanup()
}
