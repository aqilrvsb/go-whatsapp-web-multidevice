# WhatsApp Campaign & Sequence System - Complete Implementation Guide

## Table of Contents
1. [Files to Copy](#files-to-copy)
2. [URL Routes & Endpoints](#url-routes--endpoints)
3. [Database Setup](#database-setup)
4. [Frontend Implementation](#frontend-implementation)
5. [Backend Implementation](#backend-implementation)
6. [Configuration Files](#configuration-files)
7. [Directory Structure](#directory-structure)
8. [Step-by-Step Implementation](#step-by-step-implementation)

---

## Files to Copy

### 1. Model Files
```
src/models/
├── campaign.go                 # Campaign data structure
├── sequence.go                 # Sequence, SequenceStep, SequenceContact models
├── sequence_with_contacts.go   # Extended sequence model with contacts
├── lead.go                     # Lead model
├── user.go                     # User model
├── user_device.go              # Device model
└── broadcast.go                # Broadcast message model
```

### 2. Repository Files
```
src/repository/
├── campaign_repository.go      # Campaign CRUD operations
├── sequence_repository.go      # Sequence CRUD operations
├── broadcast_repository.go     # Broadcast queue management
├── lead_repository.go          # Lead management
├── user_repository.go          # User/device management
└── worker_repository.go        # Worker status tracking
```

### 3. Use Case Files
```
src/usecase/
├── campaign_trigger.go                    # Campaign triggering logic
├── optimized_campaign_trigger.go          # Optimized campaign processor
├── sequence.go                            # Sequence business logic
├── sequence_trigger_processor.go          # Sequence trigger handler
├── sequence_trigger_starter.go            # Sequence starter
├── direct_broadcast_processor.go          # Direct enrollment processor
├── broadcast_worker_processor.go          # Message queue processor
├── campaign_completion_checker.go         # Campaign completion monitor
├── campaign_status_monitor.go             # Campaign status updater
├── ai_campaign_processor.go               # AI campaign handler
└── ultra_fast_ai_campaign_processor.go    # Optimized AI processor
```

### 4. Infrastructure Files
```
src/infrastructure/broadcast/
├── device_worker.go                # Individual device worker
├── ultra_scale_broadcast_manager.go # Main broadcast manager
├── interface.go                    # Broadcast interfaces
├── whatsapp_message_sender.go      # WhatsApp sending logic
├── platform_sender.go              # Platform API sender
└── performance_optimizer.go        # Performance optimization

src/infrastructure/whatsapp/
├── worker_client_manager.go        # WhatsApp client management
└── device_health_monitor.go        # Device health monitoring

src/infrastructure/sequence/
└── worker_manager_example.go       # Sequence worker example
```

### 5. REST API Files
```
src/ui/rest/
├── app.go                  # Main REST setup
├── campaign.go             # Campaign endpoints
├── sequence.go             # Sequence endpoints
├── sequence_helper.go      # Sequence helper functions
├── broadcast.go            # Broadcast endpoints
├── lead.go                 # Lead endpoints
├── api_worker_control.go   # Worker control API
└── api_worker_status.go    # Worker status API
```

### 6. View Files (HTML/JS)
```
src/views/
├── dashboard.html          # Main dashboard with campaign calendar
├── sequences.html          # Sequence list view
├── sequence_detail.html    # Sequence detail/edit view
├── leads.html              # Lead management
└── dashboard_campaign_update.js # Campaign UI updates

src/statics/js/
├── worker_control.js       # Worker control functions
├── campaign.js             # Campaign management
├── sequence.js             # Sequence management
└── broadcast.js            # Broadcast monitoring
```

### 7. Anti-Spam/Pattern Files
```
src/pkg/antipattern/
├── greeting_processor.go   # Greeting system
├── message_randomizer.go   # Spintax & randomization
└── antipattern_manager.go  # Pattern management
```

### 8. Database Files
```
src/database/
├── connection.go           # Database connection setup
├── migrate.go              # Migration runner
└── migrations/
    ├── 001_initial_schema.sql
    ├── 002_phase2_tables.sql
    ├── 003_fix_campaigns_nulls.sql
    ├── 004_whatsapp_storage.sql
    ├── 005_add_ai_campaign_columns.sql
    ├── 007_sequence_trigger_optimization.sql
    └── 008_remove_sequence_trigger.sql
```

---

## URL Routes & Endpoints

### Frontend Pages
```
GET  /                          # Main dashboard (redirects to /home)
GET  /home                      # Dashboard with tabs
GET  /campaigns                 # Campaign calendar view
GET  /sequences                 # Sequence list
GET  /sequences/:id             # Sequence detail
GET  /leads                     # Lead management
GET  /devices                   # Device management
GET  /broadcast                 # Broadcast monitor
GET  /analytics                 # Analytics dashboard
```

### Campaign API Endpoints
```
# Campaign CRUD
GET    /api/campaigns                      # List all campaigns
POST   /api/campaigns                      # Create campaign
GET    /api/campaigns/:id                  # Get campaign details
PUT    /api/campaigns/:id                  # Update campaign
DELETE /api/campaigns/:id                  # Delete campaign

# Campaign Operations
POST   /api/campaigns/:id/trigger          # Manually trigger campaign
GET    /api/campaigns/:id/stats            # Get campaign statistics
GET    /api/campaigns/:id/messages         # Get campaign messages

# Campaign Summary
GET    /api/campaign-summary               # Get summary by date range
GET    /api/campaign-summary/stats         # Get overall statistics
GET    /api/campaign-summary/broadcast-stats # Get broadcast statistics
```

### Sequence API Endpoints
```
# Sequence CRUD
GET    /api/sequences                      # List all sequences
POST   /api/sequences                      # Create sequence
GET    /api/sequences/:id                  # Get sequence details
PUT    /api/sequences/:id                  # Update sequence
DELETE /api/sequences/:id                  # Delete sequence

# Sequence Operations
POST   /api/sequences/:id/activate         # Activate sequence
POST   /api/sequences/:id/pause            # Pause sequence
POST   /api/sequences/:id/enroll           # Manually enroll leads
GET    /api/sequences/:id/contacts         # Get enrolled contacts
GET    /api/sequences/:id/logs             # Get sequence logs

# Sequence Steps
GET    /api/sequences/:id/steps            # Get sequence steps
POST   /api/sequences/:id/steps            # Add step
PUT    /api/sequences/:id/steps/:step_id   # Update step
DELETE /api/sequences/:id/steps/:step_id   # Delete step

# Sequence Summary
GET    /api/sequence-summary               # Get all sequences summary
GET    /api/sequence-summary/stats         # Get sequence statistics
```

### Lead API Endpoints
```
GET    /api/leads                          # List leads with filters
POST   /api/leads                          # Create lead
GET    /api/leads/:id                      # Get lead details
PUT    /api/leads/:id                      # Update lead
DELETE /api/leads/:id                      # Delete lead
POST   /api/leads/import                   # Import leads (CSV)
POST   /api/leads/bulk-update              # Bulk update leads
```

### Broadcast/Worker API Endpoints
```
# Broadcast Queue
GET    /api/broadcast/queue                # Get queue status
GET    /api/broadcast/messages             # Get broadcast messages
POST   /api/broadcast/retry/:id            # Retry failed message
DELETE /api/broadcast/clear-failed         # Clear failed messages

# Worker Control
GET    /api/workers/status                 # Get all workers status
POST   /api/workers/:device_id/restart     # Restart device worker
POST   /api/workers/:device_id/stop        # Stop device worker
GET    /api/workers/:device_id/health      # Check worker health
```

### WebSocket Endpoints
```
WS     /ws                                 # Main WebSocket connection
       # Events:
       # - device_status: Device online/offline updates
       # - broadcast_progress: Real-time sending progress
       # - campaign_update: Campaign status changes
       # - sequence_update: Sequence progress updates
```

---

## Database Setup

### 1. Create Database
```sql
CREATE DATABASE whatsapp_broadcast CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. Run Migrations (in order)
```sql
-- 1. Basic tables (users, devices, leads)
source 001_initial_schema.sql

-- 2. Campaign & broadcast tables
source 002_phase2_tables.sql

-- 3. Campaign improvements
source 003_fix_campaigns_nulls.sql

-- 4. WhatsApp storage
source 004_whatsapp_storage.sql

-- 5. AI campaign support
source 005_add_ai_campaign_columns.sql

-- 6. Sequence optimizations
source 007_sequence_trigger_optimization.sql
```

### 3. Create Indexes
```sql
-- Performance indexes
CREATE INDEX idx_broadcast_device_status ON broadcast_messages(device_id, status);
CREATE INDEX idx_broadcast_scheduled ON broadcast_messages(scheduled_at);
CREATE INDEX idx_sequence_contacts_trigger ON sequence_contacts(next_trigger_time);
CREATE INDEX idx_leads_trigger ON leads(user_id, trigger);
```

---

## Frontend Implementation

### 1. Dashboard HTML Structure
```html
<!-- Main Dashboard (dashboard.html) -->
<div class="container-fluid">
    <!-- Navigation Tabs -->
    <ul class="nav nav-tabs">
        <li class="nav-item">
            <a class="nav-link" href="#devices" data-bs-toggle="tab">Devices</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" href="#campaigns" data-bs-toggle="tab">Campaigns</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" href="#sequences" data-bs-toggle="tab">Sequences</a>
        </li>
    </ul>
    
    <!-- Tab Content -->
    <div class="tab-content">
        <!-- Campaigns Tab -->
        <div class="tab-pane" id="campaigns">
            <div id="campaignCalendar"></div>
            <button onclick="showCampaignModal()">Create Campaign</button>
        </div>
        
        <!-- Sequences Tab -->
        <div class="tab-pane" id="sequences">
            <div id="sequencesList"></div>
            <button onclick="createSequence()">Create Sequence</button>
        </div>
    </div>
</div>
```

### 2. JavaScript Functions
```javascript
// Campaign Management
function loadCampaigns() {
    $.get('/api/campaigns', function(data) {
        displayCampaignsInCalendar(data);
    });
}

function saveCampaign() {
    const campaign = {
        title: $('#campaignTitle').val(),
        niche: $('#campaignNiche').val(),
        target_status: $('#campaignTargetStatus').val(),
        message: $('#campaignMessage').val(),
        image_url: $('#campaignImageUrl').val(),
        campaign_date: $('#campaignDate').val(),
        time_schedule: $('#campaignTime').val(),
        min_delay_seconds: parseInt($('#campaignMinDelay').val()),
        max_delay_seconds: parseInt($('#campaignMaxDelay').val())
    };
    
    $.ajax({
        url: '/api/campaigns',
        method: 'POST',
        data: JSON.stringify(campaign),
        contentType: 'application/json',
        success: function() {
            $('#campaignModal').modal('hide');
            loadCampaigns();
        }
    });
}

// Sequence Management
function loadSequences() {
    $.get('/api/sequences', function(sequences) {
        renderSequenceList(sequences);
    });
}

function createSequence() {
    window.location.href = '/sequences/new';
}
```

---

## Backend Implementation

### 1. Main Application Setup (`cmd/rest.go`)
```go
func restServer() {
    // Initialize database
    db := database.InitDB()
    
    // Initialize repositories
    campaignRepo := repository.NewCampaignRepository(db)
    sequenceRepo := repository.NewSequenceRepository(db)
    broadcastRepo := repository.NewBroadcastRepository(db)
    
    // Initialize use cases
    campaignUsecase := usecase.NewCampaignUsecase(campaignRepo)
    sequenceUsecase := usecase.NewSequenceUsecase(sequenceRepo)
    
    // Setup Fiber app
    app := fiber.New()
    
    // Register routes
    rest.InitCampaignRoutes(app, campaignUsecase)
    rest.InitSequenceRoutes(app, sequenceUsecase)
    
    // Start background workers
    go usecase.StartSequenceTriggerProcessor()
    go usecase.StartBroadcastWorkerProcessor()
    go usecase.StartCampaignStatusMonitor()
    
    app.Listen(":3000")
}
```

### 2. Campaign Handler Example
```go
func CreateCampaign(c *fiber.Ctx) error {
    var campaign models.Campaign
    if err := c.BodyParser(&campaign); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }
    
    // Set user ID from context
    campaign.UserID = c.Locals("user_id").(string)
    
    // Create campaign
    repo := repository.GetCampaignRepository()
    if err := repo.CreateCampaign(&campaign); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    
    return c.JSON(campaign)
}
```

---

## Configuration Files

### 1. Environment Variables (.env)
```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=whatsapp_broadcast

# Redis (optional)
REDIS_HOST=localhost
REDIS_PORT=6379

# Worker Configuration
WORKER_POOL_SIZE=10
WORKER_QUEUE_SIZE=1000
MESSAGE_BATCH_SIZE=100

# Anti-spam
MIN_DELAY_SECONDS=10
MAX_DELAY_SECONDS=30
```

### 2. Go Module (go.mod)
```go
module github.com/your-project/whatsapp-broadcast

go 1.21

require (
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/go-sql-driver/mysql v1.7.1
    github.com/google/uuid v1.5.0
    github.com/sirupsen/logrus v1.9.3
    go.mau.fi/whatsmeow v0.0.0-20240625
    github.com/go-redis/redis/v8 v8.11.5
)
```

---

## Directory Structure

```
project-root/
├── src/
│   ├── cmd/
│   │   └── rest.go                 # Main application entry
│   ├── config/
│   │   └── config.go               # Configuration loader
│   ├── database/
│   │   ├── connection.go           # DB connection
│   │   └── migrations/             # SQL migrations
│   ├── models/                     # Data models
│   ├── repository/                 # Database operations
│   ├── usecase/                    # Business logic
│   ├── infrastructure/
│   │   ├── broadcast/              # Broadcast system
│   │   └── whatsapp/               # WhatsApp integration
│   ├── ui/
│   │   ├── rest/                   # REST handlers
│   │   └── websocket/              # WebSocket handlers
│   ├── views/                      # HTML templates
│   └── statics/
│       ├── js/                     # JavaScript files
│       └── css/                    # Stylesheets
├── database/
│   └── schema.sql                  # Initial schema
├── docker-compose.yml              # Docker setup
├── Dockerfile                      # Container definition
├── go.mod                          # Go modules
└── .env.example                    # Environment template
```

---

## Step-by-Step Implementation

### Phase 1: Database Setup
1. Create MySQL database
2. Run all migration files in order
3. Create necessary indexes
4. Insert test data

### Phase 2: Backend Core
1. Copy all model files
2. Implement repositories
3. Create use case handlers
4. Setup broadcast infrastructure

### Phase 3: API Layer
1. Setup Fiber routes
2. Implement REST endpoints
3. Add authentication middleware
4. Create WebSocket handlers

### Phase 4: Frontend
1. Copy HTML templates
2. Implement JavaScript functions
3. Add CSS styling
4. Test UI interactions

### Phase 5: Background Workers
1. Start sequence processor
2. Start broadcast worker
3. Enable campaign monitor
4. Setup health checks

### Phase 6: Testing
1. Test campaign creation
2. Test sequence enrollment
3. Monitor broadcast queue
4. Verify message delivery

### Phase 7: Production
1. Configure environment
2. Setup monitoring
3. Enable logging
4. Deploy application

---

## Critical Implementation Notes

1. **Timezone Handling**: All times stored in UTC, converted to Malaysia time (UTC+8) for display
2. **Device Distribution**: Round-robin assignment across online devices
3. **Rate Limiting**: 10-30 second delays between messages (configurable)
4. **Duplicate Prevention**: Unique constraints on campaign_id + phone + device
5. **Error Recovery**: 3 retry attempts with exponential backoff
6. **Queue Management**: FIFO processing with priority for older messages
7. **Anti-Spam**: Greeting system + message randomization
8. **Platform Support**: Special handling for WABLAS/WHACENTER devices

This implementation guide provides everything needed to clone the campaign and sequence functionality 100%.