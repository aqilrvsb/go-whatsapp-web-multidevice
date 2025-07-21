// CRITICAL FIX: Zombie Pool Prevention
// Add this to ultra_scale_broadcast_manager.go

func (bwp *BroadcastWorkerPool) cleanup(manager *UltraScaleBroadcastManager) {
	bwp.mu.Lock()
	defer bwp.mu.Unlock()
	
	// Cancel all workers
	for deviceID, worker := range bwp.workers {
		worker.cancel()
		logrus.Debugf("Cancelled worker for device %s", deviceID)
	}
	
	// Cancel pool context
	bwp.cancel()
	
	// CRITICAL: Remove pool from manager map to prevent zombie pools
	poolKey := fmt.Sprintf("%s:%s", bwp.broadcastType, bwp.broadcastID)
	
	manager.mu.Lock()
	delete(manager.pools, poolKey)
	manager.mu.Unlock()
	
	logrus.Infof("✅ Pool %s cleaned up and removed from registry", poolKey)
}

// Update monitorPoolCompletion to pass manager
func (bwp *BroadcastWorkerPool) monitorPoolCompletion(manager *UltraScaleBroadcastManager) {
	// ... existing code ...
	
	// When scheduling cleanup:
	go func() {
		time.Sleep(cleanupDuration)
		bwp.cleanup(manager) // Pass manager reference
	}()
}
