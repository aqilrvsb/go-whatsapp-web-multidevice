# WhatsApp Multi-Device Campaign & Sequence System - Complete Technical Documentation

## Table of Contents
1. [System Architecture Overview](#system-architecture-overview)
2. [Database Schema](#database-schema)
3. [Campaign System](#campaign-system)
4. [Sequence System](#sequence-system)
5. [Broadcast Message Queue](#broadcast-message-queue)
6. [Worker Pool System](#worker-pool-system)
7. [Background Processes](#background-processes)
8. [Anti-Spam & Greeting System](#anti-spam-greeting-system)
9. [Platform Integration](#platform-integration)
10. [Processing Flow Diagrams](#processing-flow-diagrams)
11. [Key Implementation Details](#key-implementation-details)

---

## System Architecture Overview

The system is built with Go and uses a multi-worker architecture to handle mass WhatsApp broadcasts across thousands of devices simultaneously. It supports both:

1. **Campaigns**: One-time broadcasts to filtered leads
2. **Sequences**: Multi-day automated drip campaigns

### Core Components:
- **MySQL Database**: Stores all data (campaigns, sequences, leads, messages)
- **Redis**: Optional caching for high-scale operations
- **Worker Pool System**: Manages concurrent message sending
- **Background Processors**: Handle scheduling and automation
- **WhatsApp Web Client**: Uses go-whatsmeow library
- **Platform Integration**: Supports WABLAS and WHACENTER APIs

---

## Database Schema

### Key Tables:

#### 1. campaigns
```sql
CREATE TABLE campaigns (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(255), -- DEPRECATED, uses all user devices
    title VARCHAR(255) NOT NULL,
    niche VARCHAR(255) NOT NULL,
    target_status VARCHAR(50), -- prospect, customer, all
    message TEXT NOT NULL,
    image_url TEXT,
    campaign_date DATE NOT NULL,
    scheduled_date VARCHAR(255), -- DEPRECATED
    time_schedule TIME,
    min_delay_seconds INT DEFAULT 10,
    max_delay_seconds INT DEFAULT 30,
    status VARCHAR(50) DEFAULT 'pending', -- pending, triggered, processing, sent, failed
    ai VARCHAR(10), -- 'ai' for AI campaigns, NULL for regular
    `limit` INT DEFAULT 0, -- Device limit for AI campaigns
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### 2. sequences
```sql
CREATE TABLE sequences (
    id VARCHAR(36) PRIMARY KEY, -- UUID
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(36), -- NULL, sequences use all devices
    name VARCHAR(255) NOT NULL,
    description TEXT,
    niche VARCHAR(255),
    target_status VARCHAR(50), -- prospect, customer, all
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, paused
    `trigger` VARCHAR(255), -- Main trigger keyword
    start_trigger VARCHAR(255), -- DEPRECATED
    end_trigger VARCHAR(255), -- DEPRECATED
    total_days INT,
    is_active BOOLEAN DEFAULT true,
    schedule_time TIME DEFAULT '09:00:00',
    min_delay_seconds INT DEFAULT 10,
    max_delay_seconds INT DEFAULT 30,
    -- Progress tracking fields
    total_contacts INT DEFAULT 0,
    active_contacts INT DEFAULT 0,
    completed_contacts INT DEFAULT 0,
    failed_contacts INT DEFAULT 0,
    progress_percentage FLOAT DEFAULT 0,
    last_activity_at TIMESTAMP,
    estimated_completion_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### 3. sequence_steps
```sql
CREATE TABLE sequence_steps (
    id VARCHAR(36) PRIMARY KEY,
    sequence_id VARCHAR(36) NOT NULL,
    day_number INT NOT NULL,
    trigger VARCHAR(255),
    next_trigger VARCHAR(255),
    trigger_delay_hours INT DEFAULT 24,
    is_entry_point BOOLEAN DEFAULT false,
    message_type VARCHAR(50), -- text, image, video, document
    time_schedule TIME,
    content TEXT,
    media_url TEXT,
    caption TEXT,
    min_delay_seconds INT DEFAULT 10,
    max_delay_seconds INT DEFAULT 30,
    delay_days INT DEFAULT 1
);
```

#### 4. sequence_contacts
```sql
CREATE TABLE sequence_contacts (
    id VARCHAR(36) PRIMARY KEY,
    sequence_id VARCHAR(36) NOT NULL,
    contact_phone VARCHAR(50) NOT NULL,
    contact_name VARCHAR(255),
    current_step INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active', -- active, completed, paused, failed
    completed_at TIMESTAMP,
    current_trigger VARCHAR(255),
    next_trigger_time TIMESTAMP,
    processing_device_id VARCHAR(36),
    last_error TEXT,
    retry_count INT DEFAULT 0,
    assigned_device_id VARCHAR(36),
    processing_started_at TIMESTAMP,
    sequence_stepid VARCHAR(36),
    user_id VARCHAR(255),
    UNIQUE KEY (sequence_id, contact_phone)
);
```

#### 5. broadcast_messages (Central Queue)
```sql
CREATE TABLE broadcast_messages (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    device_id VARCHAR(36) NOT NULL,
    campaign_id INT,
    sequence_id VARCHAR(36),
    sequence_stepid VARCHAR(36),
    recipient_phone VARCHAR(50) NOT NULL,
    recipient_name VARCHAR(255),
    message_type VARCHAR(50), -- text, image, video, document
    content TEXT,
    media_url TEXT,
    status VARCHAR(50) DEFAULT 'pending', -- pending, queued, processing, sent, failed, skipped
    error_message TEXT,
    scheduled_at TIMESTAMP,
    sent_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    group_id VARCHAR(255), -- For grouping related messages
    group_order INT, -- Order within group
    INDEX idx_status (status),
    INDEX idx_device_status (device_id, status),
    INDEX idx_scheduled (scheduled_at)
);
```

#### 6. leads
```sql
CREATE TABLE leads (
    id INT AUTO_INCREMENT PRIMARY KEY,
    device_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    niche VARCHAR(255),
    journey TEXT,
    status VARCHAR(50) DEFAULT 'new', -- new, prospect, customer
    `trigger` VARCHAR(255), -- Trigger keyword for sequences
    last_interaction TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE KEY (phone, user_id)
);
```

---

## Campaign System

### Campaign Processing Flow:

1. **Creation** (`repository/campaign_repository.go`):
   - User creates campaign with target niche, status filter, and schedule
   - Duplicate prevention: Same title + date = rejected
   - Auto-scheduling: If no time set, uses Malaysia time + 5 minutes

2. **Triggering** (`usecase/campaign_trigger.go`):
   - `ProcessCampaignTriggers()` runs every 5 minutes
   - Checks for campaigns where:
     - Status = 'pending'
     - campaign_date <= today
     - time_schedule <= current time
   - Updates status to 'triggered'

3. **Lead Selection**:
   - Filters leads by:
     - User ID
     - Niche
     - Status (prospect/customer/all)
   - Excludes leads already in broadcast_messages for this campaign

4. **Message Creation**:
   - Creates broadcast_message entries for each lead
   - Assigns to online devices using round-robin
   - Sets scheduled_at = NOW() for immediate processing

### Key Functions:

```go
// Campaign creation with auto-scheduling
func (r *campaignRepository) CreateCampaign(campaign *models.Campaign) error {
    // Auto-set schedule if empty
    if campaign.TimeSchedule == "" {
        malaysiaTime := time.Now().UTC().Add(8 * time.Hour).Add(5 * time.Minute)
        campaign.TimeSchedule = malaysiaTime.Format("15:04:00")
    }
    // ... duplicate check and insert
}

// Campaign execution
func (cts *CampaignTriggerService) executeCampaign(campaign *models.Campaign) {
    // 1. Get all connected devices for user
    // 2. Filter leads by niche and status
    // 3. Create broadcast messages
    // 4. Distribute to devices round-robin
}
```

---

## Sequence System

### Sequence Processing Flow:

1. **Creation** (`usecase/sequence.go`):
   - User creates sequence with multiple steps (days)
   - Each step has trigger keywords and delays
   - Steps can be entry points or follow-ups

2. **Enrollment** (`usecase/direct_broadcast_processor.go`):
   - Leads are enrolled when their trigger matches sequence trigger
   - Creates sequence_contacts entry
   - Sets next_trigger_time based on step delays

3. **Step Processing**:
   - Checks sequence_contacts for next_trigger_time <= NOW()
   - Creates broadcast_message for current step
   - Updates contact to next step
   - Calculates next trigger time

### Key Functions:

```go
// Sequence enrollment
func (d *DirectBroadcastProcessor) enrollLeadInSequence(lead Lead, sequence Sequence) error {
    // 1. Check if already enrolled
    // 2. Find entry point step
    // 3. Create sequence_contact
    // 4. Set initial trigger time
}

// Process sequence step
func (d *DirectBroadcastProcessor) processSequenceContacts() {
    // 1. Get contacts ready for next message
    // 2. Create broadcast messages
    // 3. Update contact progress
    // 4. Handle completion/failure
}
```

---

## Broadcast Message Queue

The `broadcast_messages` table is the central queue for ALL outgoing messages.

### Message Flow:
1. **Creation**: Campaign/Sequence processors create entries
2. **Status**: `pending` → `queued` → `processing` → `sent`/`failed`
3. **Processing**: Worker pool picks up messages by device
4. **Deduplication**: Unique constraints prevent duplicates

### Status Lifecycle:
- `pending`: Just created, waiting for processing
- `queued`: Picked up by worker, in device queue
- `processing`: Currently being sent
- `sent`: Successfully delivered
- `failed`: Send failed after retries
- `skipped`: Duplicate or invalid

---

## Worker Pool System

### Architecture (`infrastructure/broadcast/`):

1. **UltraScaleBroadcastManager**: Main coordinator
   - Manages worker pools per broadcast
   - Handles device assignment
   - Monitors health and performance

2. **BroadcastWorkerPool**: Per-campaign/sequence pool
   - Contains DeviceWorkerGroups
   - Tracks statistics
   - Manages lifecycle

3. **DeviceWorkerGroup**: Per-device workers
   - Multiple workers per device (configurable)
   - Sequential sending with mutex
   - Rate limiting

4. **DeviceWorker**: Actual message sender
   - Handles WhatsApp API calls
   - Applies anti-spam delays
   - Manages retries

### Worker Configuration:
```go
const (
    MaxWorkersPerDevice = 3  // Concurrent workers per device
    WorkerQueueSize = 1000   // Messages per worker queue
    MessageTimeout = 30      // Seconds to wait for queue
)
```

---

## Background Processes

Started in `cmd/rest.go`:

### 1. Sequence Trigger Processor
```go
go usecase.StartSequenceTriggerProcessor()
// Runs every 5 minutes
// Processes BOTH sequences AND campaigns
// Uses DirectBroadcastProcessor
```

### 2. Broadcast Worker Processor
```go
go usecase.StartBroadcastWorkerProcessor()
// Runs every 5 seconds
// Gets pending messages from broadcast_messages
// Queues to worker pool
```

### 3. Campaign Status Monitor
```go
go usecase.StartCampaignStatusMonitor()
// Monitors campaign progress
// Updates status based on completion
```

### 4. Campaign Completion Checker
```go
go usecase.StartCampaignCompletionChecker()
// Checks if all messages sent
// Updates campaign status to 'sent'
```

### 5. Cleanup Worker
```go
go repository.StartCleanupWorker()
// Cleans stuck messages
// Resets processing status
```

### 6. Device Health Monitor
```go
go whatsapp.StartDeviceHealthMonitor()
// Monitors device connections
// Auto-reconnects failed devices
// Updates device status
```

---

## Anti-Spam & Greeting System

Located in `pkg/antipattern/`:

### 1. Greeting Processor
- Adds personalized greetings based on time
- Malaysian timezone (UTC+8)
- Different greetings for morning/afternoon/evening
- Name cleaning and formatting

### 2. Message Randomizer
- Spintax support: `{Hello|Hi|Hey} {friend|there}`
- Random word insertion
- Synonym replacement
- Character variation

### 3. Anti-Pattern Manager
- Tracks message patterns per device
- Ensures variation between messages
- Configurable thresholds

Example flow:
```
Original: "Check out our {product|item}"
→ Randomized: "Check out our item"
→ With greeting: "Selamat pagi John, Check out our item"
```

---

## Platform Integration

### Supported Platforms:
1. **WABLAS**: Indonesian WhatsApp API
2. **WHACENTER**: Alternative API service

### Platform Detection:
```go
if device.Platform != "" {
    // Use platform-specific sender
    sender := GetPlatformSender(device.Platform)
    return sender.SendMessage(msg)
}
// Regular WhatsApp Web sending
```

### Platform Features:
- No QR code needed
- Higher rate limits
- Built-in anti-spam
- Webhook support

---

## Processing Flow Diagrams

### Campaign Flow:
```
User Creates Campaign
    ↓
[campaigns table]
    ↓
Campaign Trigger (every 5 min)
    ↓
Filter Leads (niche + status)
    ↓
Create broadcast_messages
    ↓
Worker Pool picks up
    ↓
Device Worker sends
    ↓
Update status to 'sent'
```

### Sequence Flow:
```
Lead receives message with trigger
    ↓
Trigger matches sequence
    ↓
Enroll in sequence_contacts
    ↓
Wait for trigger time
    ↓
Create broadcast_message for step
    ↓
Worker sends message
    ↓
Update to next step
    ↓
Repeat until complete
```

---

## Key Implementation Details

### 1. Device Distribution Algorithm
```go
// Round-robin assignment
deviceIndex := i % len(connectedDevices)
device := connectedDevices[deviceIndex]
```

### 2. Duplicate Prevention
```sql
-- For campaigns
UNIQUE(campaign_id, recipient_phone, device_id)

-- For sequences
UNIQUE(sequence_stepid, recipient_phone, device_id)
```

### 3. Rate Limiting
- Per-device delays: 10-30 seconds default
- Configurable per campaign/sequence
- Platform devices may have different limits

### 4. Error Handling
- 3 retry attempts
- Exponential backoff
- Failed messages marked with error
- Automatic device switching on failure

### 5. Performance Optimizations
- Batch message creation (100 at a time)
- Connection pooling
- Redis caching (optional)
- Concurrent processing with goroutines

---

## To Clone This System:

1. **Database**: Create all tables as shown in schema
2. **Models**: Copy all structs from `models/` directory
3. **Repositories**: Implement CRUD operations for each table
4. **Use Cases**: Implement business logic processors
5. **Workers**: Build worker pool system
6. **Background Jobs**: Start all processors
7. **Anti-Spam**: Implement greeting and randomization
8. **API/UI**: Create REST endpoints and dashboard

Key files to replicate:
- `models/campaign.go`, `models/sequence.go`
- `repository/campaign_repository.go`, `repository/sequence_repository.go`
- `repository/broadcast_repository.go`
- `usecase/campaign_trigger.go`, `usecase/sequence_trigger_processor.go`
- `usecase/direct_broadcast_processor.go`
- `infrastructure/broadcast/device_worker.go`
- `infrastructure/broadcast/ultra_scale_broadcast_manager.go`
- `pkg/antipattern/*`

The system is designed to handle thousands of devices sending millions of messages with proper queuing, rate limiting, and error handling.