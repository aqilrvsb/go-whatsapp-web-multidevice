// ZOMBIE POOL FIX for ultra_scale_broadcast_manager.go

// 1. First, add manager reference to BroadcastWorkerPool struct (around line 18):
// Find this struct:
type BroadcastWorkerPool struct {
	broadcastID   string
	broadcastType string // "campaign" or "sequence"
	workers       map[string]*BroadcastWorker // key: deviceID
	maxWorkers    int
	config        *config.BroadcastConfig
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	redisClient   *redis.Client
	manager       *UltraScaleBroadcastManager // ADD THIS LINE
	
	// Statistics
	totalMessages    int64
	processedCount   int64
	failedCount      int64
	startTime        time.Time
	completionTime   *time.Time
}

// 2. Update StartBroadcastPool to set manager reference (around line 120):
func (ubm *UltraScaleBroadcastManager) StartBroadcastPool(broadcastType string, broadcastID string, userID string) (*BroadcastWorkerPool, error) {
	poolKey := fmt.Sprintf("%s:%s", broadcastType, broadcastID)
	
	ubm.mu.Lock()
	defer ubm.mu.Unlock()
	
	// Check if pool already exists
	if pool, exists := ubm.pools[poolKey]; exists {
		return pool, nil
	}
	
	// Create new pool
	ctx, cancel := context.WithCancel(context.Background())
	pool := &BroadcastWorkerPool{
		broadcastID:   broadcastID,
		broadcastType: broadcastType,
		workers:       make(map[string]*BroadcastWorker),
		maxWorkers:    ubm.maxWorkersPerPool,
		config:        ubm.config,
		ctx:           ctx,
		cancel:        cancel,
		redisClient:   ubm.redisClient,
		startTime:     time.Now(),
		manager:       ubm, // ADD THIS LINE
	}
	
	ubm.pools[poolKey] = pool
	
	// Start pool monitor
	go pool.monitor()
	
	logrus.Infof("Started broadcast pool for %s:%s with capacity for %d devices", 
		broadcastType, broadcastID, pool.maxWorkers)
	
	return pool, nil
}

// 3. Fix the cleanup function (around line 475):
func (bwp *BroadcastWorkerPool) cleanup() {
	bwp.mu.Lock()
	defer bwp.mu.Unlock()
	
	// Cancel all workers
	for _, worker := range bwp.workers {
		worker.cancel()
	}
	
	// Cancel pool context
	bwp.cancel()
	
	// CRITICAL FIX: Remove pool from manager to prevent zombie pools
	poolKey := fmt.Sprintf("%s:%s", bwp.broadcastType, bwp.broadcastID)
	
	if bwp.manager != nil {
		bwp.manager.mu.Lock()
		delete(bwp.manager.pools, poolKey)
		bwp.manager.mu.Unlock()
		logrus.Infof("✅ Cleaned up pool %s and removed from registry (no more zombie pools!)", poolKey)
	} else {
		logrus.Warnf("⚠️ Pool %s has no manager reference - might become zombie pool!", poolKey)
	}
}
