## Sequence Worker System Design (Similar to Campaign Workers)

### Overview
Adopt the campaign worker pattern for sequences to handle 3000+ devices efficiently with dedicated workers per device.

### Architecture

```
SequenceWorkerManager
├── Workers map[deviceID]*SequenceDeviceWorker
├── ProcessTriggers() - runs every 30 seconds
├── GetOrCreateWorker(deviceID) - on-demand worker creation
└── HealthCheck() - monitors worker health

SequenceDeviceWorker
├── deviceID string
├── contactQueue chan SequenceContact (size: 100)
├── processContacts() - sequential processing
├── applyHumanDelay() - random delays
└── updateProgress() - updates next_trigger_time
```

### Implementation Plan

#### 1. **Create SequenceWorkerManager** (`infrastructure/sequence/worker_manager.go`)
```go
type SequenceWorkerManager struct {
    workers      map[string]*SequenceDeviceWorker
    mu           sync.RWMutex
    maxWorkers   int
    db           *sql.DB
    checkInterval time.Duration
}

func (swm *SequenceWorkerManager) Start() {
    go swm.processLoop()
    go swm.healthCheckLoop()
}

func (swm *SequenceWorkerManager) processLoop() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        swm.assignContactsToWorkers()
    }
}
```

#### 2. **Create SequenceDeviceWorker** (`infrastructure/sequence/device_worker.go`)
```go
type SequenceDeviceWorker struct {
    deviceID       string
    client         *whatsmeow.Client
    contactQueue   chan SequenceContact
    status         string
    processedCount int
    lastActivity   time.Time
}

func (sdw *SequenceDeviceWorker) Run() {
    for contact := range sdw.contactQueue {
        sdw.processSequenceContact(contact)
        sdw.applyHumanDelay()
    }
}
```

#### 3. **Modified Sequence Processing Flow**

**Current Flow (Direct Processing):**
```
Timer → Query contacts → Process one by one → Update
```

**New Flow (Worker-Based):**
```
Timer → Query contacts → Assign to workers → Workers process → Update
```

#### 4. **Key Changes to sequence_trigger_processor.go**

```go
// Instead of direct processing
func (s *SequenceTriggerProcessor) processSequenceContacts() {
    // Get contacts grouped by device
    contactsByDevice := s.getContactsGroupedByDevice()
    
    // Assign to workers
    for deviceID, contacts := range contactsByDevice {
        worker := s.workerManager.GetOrCreateWorker(deviceID)
        worker.QueueContacts(contacts)
    }
}
```

### Database Changes

#### 1. **Add to sequence_contacts:**
```sql
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS assigned_worker_id UUID,
ADD COLUMN IF NOT EXISTS worker_assigned_at TIMESTAMP;
```

#### 2. **New indexes for performance:**
```sql
CREATE INDEX idx_sc_worker_assignment 
ON sequence_contacts(assigned_worker_id, next_trigger_time) 
WHERE status = 'active';
```

### Benefits

1. **Scalability**
   - Each device has dedicated worker
   - No competition between devices
   - Parallel processing across 3000 devices

2. **Resource Management**
   - Workers created on-demand
   - Idle workers can be cleaned up
   - Memory efficient

3. **Rate Limiting**
   - Per-device rate limits
   - Human-like delays built-in
   - WhatsApp-safe message rates

4. **Fault Tolerance**
   - Worker health monitoring
   - Automatic worker restart
   - Failed message retry

### Configuration

```go
const (
    MaxWorkersPerInstance = 500      // Max workers per server
    WorkerQueueSize      = 100       // Messages per worker queue
    WorkerIdleTimeout    = 5 * time.Minute
    HealthCheckInterval  = 1 * time.Minute
    MaxMessagesPerHour   = 80        // Per device
)
```

### Monitoring Metrics

```go
type WorkerMetrics struct {
    ActiveWorkers      int
    TotalProcessed     int64
    AverageProcessTime time.Duration
    QueueDepth         map[string]int
    ErrorRate          float64
}
```

### Migration Path

1. **Phase 1**: Implement worker system alongside current system
2. **Phase 2**: Route 10% traffic to new system
3. **Phase 3**: Monitor and increase to 50%
4. **Phase 4**: Full migration after validation

### Performance Expectations

With worker-based system:
- **Current**: 1,000 messages/minute (all devices)
- **Expected**: 50,000 messages/minute (3000 devices × 16 msg/min each)
- **Bottleneck**: Database queries (solution: add read replicas)

### Timeline

- Week 1: Implement core worker system
- Week 2: Add monitoring and metrics
- Week 3: Testing with subset of devices
- Week 4: Gradual rollout

This design leverages the proven campaign worker pattern while adapting it for the continuous nature of sequence processing.
