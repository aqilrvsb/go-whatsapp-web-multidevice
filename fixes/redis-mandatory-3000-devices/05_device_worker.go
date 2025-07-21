// Worker implementation for 3000+ devices
package broadcast

// ensureWorkerForDevice creates a worker if it doesn't exist
func (pool *OptimizedBroadcastPool) ensureWorkerForDevice(deviceID string) {
	pool.workerMutex.RLock()
	_, exists := pool.deviceWorkers[deviceID]
	pool.workerMutex.RUnlock()
	
	if !exists {
		pool.workerMutex.Lock()
		// Double-check after acquiring write lock
		if _, exists := pool.deviceWorkers[deviceID]; !exists {
			worker := pool.createDeviceWorker(deviceID)
			pool.deviceWorkers[deviceID] = worker
			
			// Start worker
			worker.wg.Add(1)
			go worker.processMessages()
		}
		pool.workerMutex.Unlock()
	}
}

// createDeviceWorker creates a new worker for a device
func (pool *OptimizedBroadcastPool) createDeviceWorker(deviceID string) *DeviceWorker {
	ctx, cancel := context.WithCancel(pool.ctx)
	
	worker := &DeviceWorker{
		workerID:      fmt.Sprintf("%s:worker:%s", pool.poolID, deviceID),
		deviceID:      deviceID,
		pool:          pool,
		messageSender: NewWhatsAppMessageSender(),
		lastActivity:  time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}
	
	logrus.Debugf("Created worker for device %s in pool %s", deviceID, pool.poolID)
	return worker
}

// processMessages is the main worker loop
func (worker *DeviceWorker) processMessages() {
	defer worker.wg.Done()
	
	// Device-specific queue
	queueKey := fmt.Sprintf("%s:device:%s", worker.pool.queueKey, worker.deviceID)
	logrus.Infof("Worker %s started processing queue %s", worker.workerID, queueKey)
	
	// Batch processing for efficiency
	batchSize := 10
	messages := make([]*domainBroadcast.BroadcastMessage, 0, batchSize)
	
	for {
		select {
		case <-worker.ctx.Done():
			logrus.Infof("Worker %s stopped", worker.workerID)
			return
			
		default:
			// Get messages from Redis (blocking with timeout)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			result, err := worker.pool.redisClient.BRPop(ctx, 1*time.Second, queueKey).Result()
			cancel()
			
			if err == redis.Nil {
				// No messages, check if we should continue
				if worker.shouldStop() {
					logrus.Debugf("Worker %s idle, stopping", worker.workerID)
					return
				}
				continue
			}
			
			if err != nil {
				logrus.Errorf("Worker %s error reading queue: %v", worker.workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}
			
			// Parse message
			var msg domainBroadcast.BroadcastMessage
			if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
				logrus.Errorf("Worker %s failed to parse message: %v", worker.workerID, err)
				continue
			}
			
			// Process message with rate limiting
			worker.processMessage(&msg)
		}
	}
}

// processMessage handles a single message
func (worker *DeviceWorker) processMessage(msg *domainBroadcast.BroadcastMessage) {
	startTime := time.Now()
	worker.lastActivity = startTime
	
	// Apply delay between messages
	delay := calculateRandomDelay(msg.MinDelay, msg.MaxDelay)
	time.Sleep(delay)
	
	// Send message
	err := worker.messageSender.SendMessage(worker.deviceID, msg)
	
	// Update statistics
	if err != nil {
		atomic.AddInt64(&worker.failedCount, 1)
		atomic.AddInt64(&worker.pool.failedCount, 1)
		
		// Update database
		db := database.GetDB()
		db.Exec(`UPDATE broadcast_messages SET 
				status = 'failed', 
				error_message = $1,
				sent_at = NOW()
			WHERE id = $2`, err.Error(), msg.ID)
		
		logrus.Errorf("Worker %s failed to send message: %v", worker.workerID, err)
		
		// Add to dead letter queue
		deadLetterKey := fmt.Sprintf("%s:%s", ultraDeadLetterPrefix, worker.deviceID)
		msgData, _ := json.Marshal(msg)
		worker.pool.redisClient.LPush(context.Background(), deadLetterKey, msgData)
	} else {
		atomic.AddInt64(&worker.processedCount, 1)
		atomic.AddInt64(&worker.pool.processedCount, 1)
		
		// Update database
		db := database.GetDB()
		db.Exec(`UPDATE broadcast_messages SET 
				status = 'sent',
				sent_at = NOW()
			WHERE id = $1`, msg.ID)
		
		// Update metrics
		worker.updateMetrics(time.Since(startTime))
		
		logrus.Debugf("Worker %s sent message to %s (took %v with delay %v)", 
			worker.workerID, msg.RecipientPhone, time.Since(startTime), delay)
	}
}

// shouldStop checks if worker should stop
func (worker *DeviceWorker) shouldStop() bool {
	// Stop if idle for more than 30 seconds
	return time.Since(worker.lastActivity) > 30*time.Second
}

// updateMetrics updates performance metrics
func (worker *DeviceWorker) updateMetrics(duration time.Duration) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", ultraMetricsPrefix, worker.deviceID)
	
	// Update various metrics
	pipe := worker.pool.redisClient.Pipeline()
	pipe.HIncrBy(ctx, key, "messages_sent", 1)
	pipe.HIncrBy(ctx, key, "total_duration_ms", duration.Milliseconds())
	pipe.HSet(ctx, key, "last_activity", time.Now().Unix())
	pipe.Expire(ctx, key, 24*time.Hour)
	pipe.Exec(ctx)
}
