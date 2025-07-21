// Fix for zombie pool bug - Add this to ultra_scale_broadcast_manager.go

// cleanup removes the pool after completion and removes it from manager
func (bwp *BroadcastWorkerPool) cleanup(manager *UltraScaleBroadcastManager) {
	bwp.mu.Lock()
	defer bwp.mu.Unlock()
	
	// Cancel all workers
	for deviceID, worker := range bwp.workers {
		worker.cancel()
		logrus.Debugf("Cancelled worker for device %s in pool %s:%s", 
			deviceID, bwp.broadcastType, bwp.broadcastID)
	}
	
	// Cancel pool context
	bwp.cancel()
	
	// CRITICAL FIX: Remove pool from manager to prevent zombie pools
	poolKey := fmt.Sprintf("%s:%s", bwp.broadcastType, bwp.broadcastID)
	
	manager.mu.Lock()
	delete(manager.pools, poolKey)
	manager.mu.Unlock()
	
	// Clear Redis queues for this pool
	ctx := context.Background()
	queueKey := fmt.Sprintf("ultra:queue:%s:%s", bwp.broadcastType, bwp.broadcastID)
	if err := bwp.redisClient.Del(ctx, queueKey).Err(); err != nil {
		logrus.Warnf("Failed to clear Redis queue %s: %v", queueKey, err)
	}
	
	// Log cleanup completion
	logrus.Infof("✅ Cleaned up broadcast pool %s:%s (removed from registry to prevent zombie pools)", 
		bwp.broadcastType, bwp.broadcastID)
}

// Update the monitorPoolCompletion to pass manager reference
func (bwp *BroadcastWorkerPool) monitorPoolCompletion(manager *UltraScaleBroadcastManager) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	consecutiveIdleChecks := 0
	requiredIdleChecks := 30 // 5 minutes of idle time
	
	for {
		select {
		case <-ticker.C:
			processed := atomic.LoadInt64(&bwp.processedCount)
			failed := atomic.LoadInt64(&bwp.failedCount)
			total := atomic.LoadInt64(&bwp.totalMessages)
			
			// Check if all messages are processed
			if processed+failed >= total && total > 0 {
				bwp.mu.RLock()
				activeWorkers := 0
				for _, worker := range bwp.workers {
					if worker.status == "processing" {
						activeWorkers++
					}
				}
				bwp.mu.RUnlock()
				
				if activeWorkers == 0 {
					consecutiveIdleChecks++
					logrus.Debugf("Pool %s:%s idle check %d/%d", 
						bwp.broadcastType, bwp.broadcastID, 
						consecutiveIdleChecks, requiredIdleChecks)
					
					if consecutiveIdleChecks >= requiredIdleChecks {
						// Mark completion time
						now := time.Now()
						bwp.completionTime = &now
						
						// Update campaign/sequence status to completed
						db := database.GetDB()
						if bwp.broadcastType == "campaign" {
							db.Exec(`UPDATE campaigns SET status = 'completed', 
									updated_at = NOW() WHERE id = $1`, bwp.broadcastID)
						}
						
						logrus.Infof("Pool %s:%s completed. Total: %d, Processed: %d, Failed: %d", 
							bwp.broadcastType, bwp.broadcastID, total, processed, failed)
						
						// Calculate cleanup delay
						cleanupDuration := 5 * time.Minute
						if bwp.config != nil && bwp.config.PoolCleanupDelayMinutes > 0 {
							cleanupDuration = time.Duration(bwp.config.PoolCleanupDelayMinutes) * time.Minute
						}
						
						logrus.Infof("Scheduling pool cleanup for %s:%s after %v", 
							bwp.broadcastType, bwp.broadcastID, cleanupDuration)
						
						// Schedule cleanup with manager reference
						go func() {
							time.Sleep(cleanupDuration)
							bwp.cleanup(manager)
						}()
						
						return // Exit monitor
					}
				} else {
					consecutiveIdleChecks = 0 // Reset if workers are active
				}
			} else {
				consecutiveIdleChecks = 0 // Reset if messages still pending
			}
			
		case <-bwp.ctx.Done():
			return
		}
	}
}
