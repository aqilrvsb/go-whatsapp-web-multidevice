# WhatsApp Web Multi-Device Broadcast System

A comprehensive WhatsApp broadcast system supporting multi-device operations, campaign management, and automated messaging sequences.

## Recent Updates (August 2025)

### 🔧 Critical Fixes Applied

#### 1. **Duplicate Message Prevention** ✅
- **Issue**: Messages were being sent multiple times by device workers
- **Root Cause**: `GetPendingMessages` was being called instead of `GetPendingMessagesAndLock`
- **Fix**: 
  - Changed to use `GetPendingMessagesAndLock` with atomic worker ID locking
  - Added `processing_worker_id` column for message claiming
  - Each message can now only be processed by one worker
  - Added 'processing' status to all duplicate checks

#### 2. **Sequence Duplicate Prevention** ✅
- **Issue**: Same sequence message sent multiple times to same recipient
- **Fix**: 
  - Duplicate check based on: `sequence_stepid + recipient_phone + device_id`
  - Checks all statuses: pending, processing, queued, sent
  - Database unique constraint available in `add_unique_constraints.sql`

#### 3. **Campaign Duplicate Prevention** ✅
- **Issue**: Campaign messages could be duplicated
- **Fix**:
  - Duplicate check based on: `campaign_id + recipient_phone + device_id`
  - Comprehensive status checking
  - Database unique constraint available

#### 4. **Sequence Modal Date Filter** ✅
- **Issue**: Sequence step details modal showed all historical messages ignoring date filter
- **Fix**:
  - `GetSequenceStepLeads` now respects date filters
  - Frontend passes date parameters correctly
  - Modal shows only messages from selected date range

#### 5. **Worker ID Implementation** ✅
- **Issue**: Worker ID column existed but wasn't being used
- **Fix**:
  - Implemented atomic message claiming with unique worker IDs
  - Prevents race conditions
  - Messages stuck in 'processing' auto-reset after 5 minutes

## System Architecture

### Message Flow

#### Sequences (A to Z):
1. **Creation**: Admin creates sequence with multiple day steps
2. **Enrollment**: Leads enrolled based on triggers (e.g., COLDVITAC)
3. **Daily Processing**: `ProcessDailySequenceMessages` creates messages for each contact's current day
4. **Duplicate Check**: Verifies no existing message for step+phone+device
5. **Queue**: Message added to `broadcast_messages` table
6. **Processing**: Worker claims message with atomic lock
7. **Sending**: WhatsApp API sends message
8. **Status Update**: Marked as 'sent'

#### Campaigns (A to Z):
1. **Creation**: Admin creates campaign with target criteria
2. **Triggering**: `ProcessCampaignTriggers` runs every minute
3. **Lead Matching**: Finds leads matching campaign criteria
4. **Duplicate Check**: Verifies no existing message for campaign+phone+device
5. **Queue**: Message added to `broadcast_messages` table
6. **Processing**: Same as sequences - atomic worker locking
7. **Sending**: Same as sequences

### Duplicate Prevention (3 Layers)

1. **Application Level**
   - Pre-insert checks in `QueueMessage()`
   - Comprehensive status checking (pending, processing, queued, sent)

2. **Worker Level**
   - Atomic locking with `processing_worker_id`
   - `GetPendingMessagesAndLock()` prevents concurrent processing
   - Unique worker ID per execution

3. **Database Level**
   - Unique constraints (run `add_unique_constraints.sql`)
   - Prevents duplicates even if application logic fails

## Database Schema

### Key Tables
- `broadcast_messages`: Central message queue
- `campaigns`: Campaign definitions
- `sequences`: Sequence definitions
- `sequence_steps`: Individual sequence messages
- `sequence_contacts`: Enrolled contacts
- `leads`: Contact database

### Important Columns in broadcast_messages
- `processing_worker_id`: Atomic lock for message claiming
- `processing_started_at`: Timestamp for stuck message detection
- `sequence_stepid`: For sequence duplicate prevention
- `campaign_id`: For campaign duplicate prevention
- `device_id`: Device that will send the message
- `recipient_phone`: Target contact

## Configuration

### Required Database Changes
Run `add_unique_constraints.sql` to add:
```sql
-- Sequence unique constraint
ALTER TABLE broadcast_messages 
ADD UNIQUE INDEX unique_sequence_message (
    sequence_stepid, 
    recipient_phone, 
    device_id
);

-- Campaign unique constraint  
ALTER TABLE broadcast_messages 
ADD UNIQUE INDEX unique_campaign_message (
    campaign_id, 
    recipient_phone, 
    device_id
);
```

## Monitoring

### Check Worker ID Usage
```sql
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total,
    COUNT(processing_worker_id) as with_worker_id,
    ROUND(COUNT(processing_worker_id) / COUNT(*) * 100, 1) as percentage
FROM broadcast_messages 
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

### Check for Duplicates
```sql
-- Sequence duplicates
SELECT sequence_stepid, recipient_phone, device_id, COUNT(*) as count
FROM broadcast_messages 
WHERE sequence_stepid IS NOT NULL
GROUP BY sequence_stepid, recipient_phone, device_id
HAVING COUNT(*) > 1;

-- Campaign duplicates
SELECT campaign_id, recipient_phone, device_id, COUNT(*) as count
FROM broadcast_messages 
WHERE campaign_id IS NOT NULL
GROUP BY campaign_id, recipient_phone, device_id
HAVING COUNT(*) > 1;
```

### Check Stuck Messages
```sql
SELECT COUNT(*) as stuck_count
FROM broadcast_messages 
WHERE status = 'processing'
AND processing_started_at < DATE_SUB(NOW(), INTERVAL 5 MINUTE);
```

## Building

```bash
# Build without CGO
cd src
set CGO_ENABLED=0
go build -o ../whatsapp.exe

# Or use the build script
build_nocgo.bat
```

## Deployment

1. Ensure database migrations are applied
2. Run `add_unique_constraints.sql` for duplicate prevention
3. Deploy the binary
4. Monitor logs for duplicate prevention messages
5. Verify `processing_worker_id` is being populated

## Support

For issues or questions, please check:
- Application logs for "Skipping duplicate" messages
- Database for constraint violations
- Worker ID population in broadcast_messages table

---

*Last Updated: August 2025*
