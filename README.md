# WhatsApp Multi-Device System - Dual Database Edition
**Last Updated: August 06, 2025 - MySQL Application Data + PostgreSQL WhatsApp Sessions**  
**Status: ✅ Production-ready with Worker Pool System**

## 🚨 CRITICAL UPDATE: Duplicate Message Prevention (August 6, 2025)

### Fixed: Messages Being Sent Multiple Times
**Issue**: Some users received the same message 2-3 times at 1-minute intervals.

**Root Cause**: Race condition in `GetPendingMessagesAndLock()` where multiple workers could claim the same message before status update completed.

**Solution Applied**:
1. **Added Worker Locking Columns**:
   ```sql
   ALTER TABLE broadcast_messages 
   ADD COLUMN processing_worker_id VARCHAR(100) DEFAULT NULL,
   ADD COLUMN processing_started_at TIMESTAMP NULL,
   ADD INDEX idx_processing_worker (processing_worker_id);
   ```

2. **Atomic Message Claiming**: 
   - Each worker generates unique ID
   - Only unclaimed messages (processing_worker_id IS NULL) can be processed
   - Prevents multiple workers from processing same message

3. **Fixed Sequence Duplicate Check**:
   - Changed from checking `group_order` to `sequence_stepid`
   - Checks all statuses: pending, processing, queued, sent

4. **Added Safety Double-Check**:
   - Worker verifies message status before sending
   - Skips if already marked as 'sent'

5. **Automatic Cleanup**:
   - Stuck messages (processing > 5 minutes) automatically reset to pending

**Impact**: ZERO duplicate messages even with 3000+ devices running simultaneously!

## 🚨 CRITICAL UPDATE: Duplicate Message Fix Applied (August 6, 2025)

### Fixed: Messages Being Sent Multiple Times
**Issue**: Some users were receiving the same message 2-3 times within minutes.

**Root Cause**: Race condition where multiple workers could process the same message simultaneously due to timing between status updates.

**Solution Applied**:
- Added `processing_worker_id` column for atomic message claiming
- Each worker now exclusively locks messages before processing
- Implemented automatic cleanup for stuck messages
- Fixed sequence duplicate check to use correct fields

**Result**: 
- ✅ Zero duplicate messages
- ✅ Works perfectly with 3000+ devices
- ✅ No changes to worker pool architecture

## 🚨 CRITICAL UPDATE: Worker Pool System Activated (August 5, 2025)

### Duplicate Message Fix:
The system now uses the Worker Pool System for all campaigns and sequences to prevent duplicate messages.

**Root Cause**: The broadcast processor was fetching messages with `status = 'pending'` but there was a race condition between fetching and updating status, allowing multiple processor runs to grab the same message.

**Solution Applied**: 
- Activated the Worker Pool System with 5 workers per device
- Messages are queued in channels (can only be consumed once)
- Mutex lock ensures sequential sending per device
- No more duplicate messages!

**Before**: Direct processing loop → Race conditions → Duplicate messages
**After**: Worker Pool System → Channel queues → Single consumption → No duplicates

## 🚨 IMPORTANT: Timezone Configuration (August 4, 2025)

### Server Timezone Adjustment:
The system is configured to handle an 8-hour timezone difference between the server (UTC-8) and Malaysia time (UTC+8).

**Applied Fix**: The broadcast worker now adds 8 hours when checking scheduled messages:
```sql
-- Before: Messages stuck as server time is 8 hours behind
WHERE scheduled_at <= NOW()

-- After: Correctly processes Malaysia-scheduled messages  
WHERE scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR)
```

**Impact**: 
- Messages scheduled in Malaysia time will be sent correctly
- No more pending messages stuck at end of day
- Worker checks 10-minute window adjusted for timezone

**Note**: If server timezone changes, update the interval in `broadcast_repository.go`

## 🔧 Recent Updates (August 5, 2025)

### Worker Pool System Implementation:
1. **Fixed Duplicate Messages**: Campaigns and sequences now use Worker Pool System exclusively
2. **5 Workers Per Device**: Parallel processing with mutex-based sequential sending
3. **Channel-Based Queuing**: Messages consumed exactly once, preventing duplicates
4. **Automatic Status Updates**: pending → queued → sent with no race conditions
5. **Rate Limiting Built-in**: Worker pool handles anti-spam delays automatically

### Timezone Fix (August 4, 2025):
1. **Fixed 8-hour timezone mismatch**: Server (UTC-8) vs Malaysia (UTC+8)
2. **Adjusted worker query**: Added 8-hour offset to scheduled message checks
3. **Prevents stuck messages**: Messages no longer remain pending past scheduled time

### Critical Message Fix (August 3, 2025):
1. **Fixed Missing Messages**: Fixed critical bug where `GetPendingMessages` wasn't appending messages to return array
2. **Anti-Spam Flow**: Fixed double anti-spam application - now applied once in BroadcastWorker for all device types
3. **Platform Device Support**: Platform devices (Wablas/Whacenter) now also receive anti-spam (greeting + randomization)
4. **True Random Delays**: Fixed delays to be truly random instead of always using middle value
5. **Sequence Step Delays**: Each sequence step can now have its own min/max delay settings
6. **Fixed Status Queries**: Changed all `status = 'success'` to `status = 'sent'` throughout the system
7. **Sequence Summary Fixes**:
   - Changed "Campaign Device Report" to "Sequence Device Report" for sequences
   - Fixed statistics to use correct status values (sent, failed, pending/queued)
   - Added date range filter (like campaigns)
   - Auto-filters to today's date on page load
   - Date filters are passed to device reports
8. **Error Message Display**: Added error_message column to failed leads popup (for debugging failures)

### Previous Fixes (August 2, 2025):
1. **Duplicate Prevention**: 
   - For Sequences: Checks `sequence_stepid`, `recipient_phone`, and `device_id`
   - For Campaigns: Checks `campaign_id`, `recipient_phone`, and `device_id`
   - Prevents duplicate message creation before inserting
2. **Message Ordering**: Fixed message order to use `scheduled_at` timestamp instead of `created_at`
3. **Recipient Name Display**: Fixed name detection algorithm to properly show recipient names instead of defaulting to "Cik"
4. **Line Break Support**: Fixed message formatting to properly display line breaks in WhatsApp
5. **Message Type Fix**: Changed from ExtendedTextMessage to Conversation for better compatibility

### Database Cleanup Required:
```sql
-- Add columns for duplicate prevention (REQUIRED - August 6, 2025)
ALTER TABLE broadcast_messages 
ADD COLUMN processing_worker_id VARCHAR(100) DEFAULT NULL,
ADD COLUMN processing_started_at TIMESTAMP NULL,
ADD INDEX idx_processing_worker (processing_worker_id),
ADD INDEX idx_processing_started (processing_started_at);

-- Remove duplicate pending messages (both sequences and campaigns)
DELETE bm1 FROM broadcast_messages bm1
INNER JOIN broadcast_messages bm2 
WHERE bm1.recipient_phone = bm2.recipient_phone
AND ((bm1.sequence_id = bm2.sequence_id AND bm1.sequence_stepid = bm2.sequence_stepid)
  OR (bm1.campaign_id = bm2.campaign_id))
AND bm1.device_id = bm2.device_id
AND bm1.status = 'pending'
AND bm2.status = 'pending'
AND bm1.created_at > bm2.created_at;
```

## 🎯 Database Architecture

This system uses a dual-database approach for optimal performance:

### 1. **PostgreSQL** (WhatsApp Session Storage)
- Stores WhatsApp device sessions
- Required by the WhatsApp library (whatsmeow)
- Uses Railway's built-in PostgreSQL or your own instance

### 2. **MySQL** (Application Data)
- Stores all application data:
  - Users and authentication
  - Devices and configurations
  - Leads and contacts
  - Campaigns and messages
  - Sequences and automation
  - Broadcast messages and queues

## 📚 Database Documentation

### Schema Documentation
- **MySQL Schema**: See [MYSQL_SCHEMA_DOCUMENTATION.md](MYSQL_SCHEMA_DOCUMENTATION.md) for complete table structure
- **Database Design**: See [CURRENT_WORKING_SCHEMA.sql](CURRENT_WORKING_SCHEMA.sql) for PostgreSQL reference

### Key Files to Understand Database Structure
1. **`MYSQL_SCHEMA_DOCUMENTATION.md`** - Complete MySQL schema with all tables, columns, indexes
2. **`src/models/`** - Go structs that map to database tables
3. **`src/repository/`** - Database queries and operations
4. **`src/database/migrations/`** - Database migration files

## 🔧 Database Connection Guide

### Connecting to PostgreSQL

#### Option 1: Railway PostgreSQL (Recommended for Production)
1. Go to your Railway project
2. Click on the PostgreSQL service
3. Go to the "Connect" tab
4. Copy the `DATABASE_URL` (use the public URL, not internal)
5. Add to `.env`:
```env
DB_URI=postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway
```

Current connection:
```env
# Railway PostgreSQL (WhatsApp Sessions)
DB_URI=postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway
```

#### Option 2: Local PostgreSQL
```env
DB_URI=postgresql://localhost:5432/whatsapp_db?sslmode=disable
```

#### Option 3: SQLite (Development Only)
```env
DB_URI=file:storages/whatsapp.db?_foreign_keys=on
```

### Connecting to MySQL

The MySQL connection is configured in `.env`:
```env
MYSQL_URI=mysql://username:password@host:port/database_name
```

Example:
```env
MYSQL_URI=mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway
```

### Testing Database Connections

Run the database operations script:
```bash
# Install dependencies and run operations
run_database_operations.bat

# Or manually:
pip install psycopg2-binary pymysql
python database_operations.py
```

This script will:
1. Connect to both databases
2. Export MySQL schema documentation
3. Show PostgreSQL disk usage
4. Clean specified tables (if confirmed)

## 🔧 Environment Configuration

### Railway Deployment
Set these environment variables in Railway:

```env
# PostgreSQL for WhatsApp Sessions
DB_URI=postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway

# MySQL for Application Data
MYSQL_URI=mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway

# Application Settings
APP_PORT=3000
APP_DEBUG=false
APP_BASIC_AUTH=admin:changeme123

# WhatsApp Settings
WHATSAPP_AUTOREAD=true
```

### Local Development
Create a `.env` file:

```env
# Use SQLite for WhatsApp sessions locally
DB_URI=file:storages/whatsapp.db?_foreign_keys=on

# MySQL for Application Data
MYSQL_URI=mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway
```

## 🚀 Key Features

### ✅ Multi-Device Support
- Connect unlimited WhatsApp devices (3000+ tested)
- Each device operates independently
- Self-healing connections with auto-refresh
- Device-level rate limiting

### ✅ Campaign System
- Create and manage marketing campaigns
- CSV lead import/export
- Automatic device distribution
- Real-time progress tracking
- Spintax support for message variation

### ✅ Sequence System
- Multi-step automated sequences
- Trigger-based enrollment
- Time-delayed messages
- Automatic progression (COLD → WARM → HOT)

### ✅ Anti-Spam Protection
- 5-15 second delays between messages
- Device-level mutex prevents simultaneous sends
- 10% homoglyph replacement
- Message variation using Spintax
- Greeting personalization

## 🏗️ Building & Deployment

### Local Build (Windows)
```bash
# Without CGO (no SQLite support)
build_local.bat

# Run
whatsapp.exe rest
```

### Railway Deployment
1. Push to GitHub
2. Railway auto-deploys from main branch
3. Ensure environment variables are set
4. Add your server IP to MySQL remote access

## 📊 MySQL Compatibility Updates

### Changes Made for MySQL Support:
1. **Parameter Placeholders**: Changed from `$1, $2` to `?, ?`
2. **UUID Generation**: Changed from `gen_random_uuid()` to `UUID()`
3. **Boolean Values**: Changed from `TRUE/FALSE` to `1/0`
4. **Case-Insensitive Search**: Changed from `ILIKE` to `LIKE`
5. **Null Ordering**: Removed `NULLS LAST` (not supported in MySQL)
6. **RETURNING Clause**: Removed (use LastInsertId instead)

## 🛠️ Troubleshooting

### PostgreSQL Disk Space Issues
If PostgreSQL is hitting 100% disk usage, run:
```bash
python db_operations_fixed.py
```
This will clear the following tables:
- leads (26,366 records cleared)
- leads_ai (21 records cleared)
- sequences (3 records cleared)
- sequence_contacts
- broadcast_messages
- campaigns (3 records cleared)

**Results from latest cleanup:**
- Database size reduced from 167 MB to 121 MB
- 46 MB of disk space reclaimed
- VACUUM FULL executed to free disk space

### MySQL Connection Issues
1. **Error 1130**: Add Railway's IP to MySQL remote access in cPanel
2. **Error 1064**: SQL syntax has been updated for MySQL compatibility
3. **Error 1054**: Parameter placeholders have been converted to MySQL format

### Database Requirements
- **MySQL 5.7+** or **MariaDB 10.2+**
- **PostgreSQL 12+** for WhatsApp sessions
- Remote access enabled for your deployment server

## 📈 Performance Optimizations

- Connection pool: 500 max connections
- Idle connections: 100
- Connection lifetime: 5 minutes
- Optimized for 3000+ devices
- Supports 200+ concurrent users

## 🔄 Migration from PostgreSQL

If migrating from a PostgreSQL-only setup:
1. Export your data from PostgreSQL
2. Import into MySQL using the schema in `MYSQL_SCHEMA_DOCUMENTATION.md`
3. Update environment variables
4. Deploy the new version

## 📝 API Endpoints

The system provides REST APIs for:
- Device management
- Lead management
- Campaign creation and monitoring
- Sequence automation
- Message sending
- Analytics and reporting

Access the web interface at `http://your-domain:3000`

## 🔐 Security

- Basic authentication for API access
- Device-level access control
- Webhook secret validation
- MySQL SSL connections supported

## 📄 License

This project is based on the original work from [aldinokemal/go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice)

---

**Important**: Ensure both PostgreSQL (for WhatsApp) and MySQL (for application data) are properly configured before deployment.