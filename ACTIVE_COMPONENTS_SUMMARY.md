# Campaign & Sequence System - Active Components Summary

## Background Processes (Running in cmd/rest.go)

### 1. **UltraOptimizedBroadcastProcessor** ✅
- File: `usecase/ultra_optimized_broadcast_processor.go`
- Purpose: Main processor for 3000+ device support
- Creates broadcast-specific worker pools

### 2. **OptimizedCampaignTrigger** ✅
- File: `usecase/optimized_campaign_trigger.go`
- Purpose: Processes campaigns every minute
- Handles timezone-aware campaign execution

### 3. **SequenceTriggerProcessor** ✅
- File: `usecase/sequence_trigger_processor.go`
- Purpose: Unified processor for BOTH sequences AND campaigns
- Uses DirectBroadcastProcessor internally

### 4. **CampaignStatusMonitor** ✅
- File: `usecase/campaign_status_monitor.go`
- Purpose: Monitors campaign progress and updates status

### 5. **BroadcastCoordinator** ✅
- File: `usecase/broadcast_coordinator.go`
- Purpose: Coordinates broadcast message distribution

### 6. **BroadcastWorkerProcessor** ✅
- File: `usecase/broadcast_worker_processor.go`
- Purpose: Processes queued messages using Worker Pool System

### 7. **CleanupWorker** ✅
- File: `repository/cleanup_worker.go`
- Purpose: Cleans stuck messages and resets processing status

### 8. **CampaignCompletionChecker** ✅
- File: `usecase/campaign_completion_checker.go`
- Purpose: Checks if campaigns are complete and updates status

## Active API Routes

### Campaign Routes
```
GET    /api/campaigns                      # List campaigns
POST   /api/campaigns                      # Create campaign
PUT    /api/campaigns/:id                  # Update campaign
DELETE /api/campaigns/:id                  # Delete campaign
GET    /api/campaigns/summary              # Campaign summary
GET    /api/campaigns/:id/device-report    # Device report
POST   /api/campaigns/:id/device/:deviceId/retry-failed  # Retry failed
```

### Sequence Routes
```
GET    /api/sequences                      # List sequences
POST   /api/sequences                      # Create sequence
PUT    /api/sequences/:id                  # Update sequence
DELETE /api/sequences/:id                  # Delete sequence
GET    /api/sequences/summary              # Sequence summary
POST   /api/sequences/:id/toggle           # Toggle active/inactive
GET    /api/sequences/:id/device-report    # Device report
```

### Worker/System Routes
```
GET    /api/workers/status                 # Worker status
POST   /api/workers/resume-failed          # Resume failed workers
POST   /api/workers/stop-all               # Stop all workers
GET    /api/system/status                  # System status
```

## Core Infrastructure

### Broadcast Manager
- **Main**: `infrastructure/broadcast/ultra_scale_broadcast_manager.go`
- **Worker**: `infrastructure/broadcast/device_worker.go`
- **Interface**: `infrastructure/broadcast/interface.go`

### Key Features
1. Worker Pool System (prevents duplicates)
2. Device-based message distribution
3. Anti-spam with greeting processor
4. Platform support (WABLAS/WHACENTER)
5. Rate limiting (10-30 seconds between messages)
6. Automatic retry on failure
7. Real-time status updates

## Database Tables Used
- campaigns
- sequences
- sequence_steps
- sequence_contacts
- broadcast_messages
- leads
- users
- user_devices

## Files Removed
- All AI campaign processors
- Team member functionality
- Redundant worker status endpoints
- Redis status views
- Backup and temporary files
- Unused views (whatsapp_web, redis_status, etc.)

## Current Status
The system is now clean, optimized, and production-ready with only essential components for campaign and sequence functionality.