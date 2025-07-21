# WhatsApp Multi-Device System - ULTIMATE BROADCAST EDITION
**Last Updated: January 24, 2025 - Complete System with Group & Community Management**  
**Status: ✅ Production-ready with 3000+ device support**
**Architecture: ✅ Redis-based queuing + Worker pools + Per-step delays**
**Deploy**: ✅ Auto-deployment via Railway with Redis

## 🚀 LATEST UPDATES (January 24, 2025)

### ✅ NEW: Group & Community Management:
1. **Group Operations** - Create groups, manage participants, admin controls
2. **Community Features** - Create communities, add members, link groups
3. **Complete API** - REST endpoints for all group/community operations
4. **Based on whatsmeow** - Using native WhatsApp Web Multi-Device protocol

### ✅ Complete Working System:
1. **Redis is MANDATORY** - No fallback, optimized for 3000+ devices
2. **Zombie Pool Bug Prevention** - Pools properly cleaned from registry
3. **Per-Step Delays Fixed** - Each sequence step uses its own delays
4. **Unified Processing** - Same Redis system for campaigns AND sequences

### ✅ How Delays Work:
- **Campaigns**: Use `min_delay_seconds` and `max_delay_seconds` from `campaigns` table
- **Sequences**: Use `min_delay_seconds` and `max_delay_seconds` from `sequence_steps` table
- Each sequence step can have different delays (Step 1: 5-10s, Step 2: 20-30s, etc.)
- NO rate limiting - only delays between messages

### ✅ System Architecture (100% Working):
```
CAMPAIGNS                           SEQUENCES
    ↓                                   ↓
Create messages                    Time-based processor (15 sec)
    ↓                                   ↓
    └────→ broadcast_messages table ←───┘
                    ↓
         Unified Processor (2 sec)
                    ↓
            Redis Queues (MANDATORY)
                    ↓
            Worker Pools (per broadcast)
                    ↓
            Device Workers (per device)
                    ↓
              WhatsApp API
```

### ✅ Key Features:
- **3000+ devices** supported simultaneously
- **No conflicts** - One worker per device
- **Auto-cleanup** - Pools removed after 5 min idle (no zombies)
- **Per-step delays** - Each sequence step has custom delays
- **Platform support** - Works with Wablas/Whacenter APIs
- **100% unified** - Same flow for campaigns and sequences
- **Group Management** - Create, manage groups and participants
- **Community Support** - Create and manage WhatsApp Communities

## 🚀 Quick Start

### Prerequisites:
- Redis (MANDATORY) - System won't start without it
- PostgreSQL database
- Go 1.19+ for building
- Railway or similar platform

### Environment Variables:
```bash
REDIS_URL=redis://user:password@host:port/db  # REQUIRED!
DATABASE_URL=postgresql://user:pass@host/db   # REQUIRED!
APP_PORT=3000
```

### Local Development:
```bash
# Clone repository
git clone https://github.com/aqilrvsb/go-whatsapp-web-multidevice.git
cd go-whatsapp-web-multidevice

# Set environment
set REDIS_URL=redis://localhost:6379
set DATABASE_URL=postgresql://user:pass@localhost/whatsapp

# Build and run
build_local.bat
```

### Railway Deployment:
1. Add PostgreSQL service
2. Add Redis service  
3. Push to GitHub
4. Railway auto-deploys with all services connected

## 📊 How It Works

### Campaign Flow:
1. Create campaign with min/max delays
2. Messages created in `broadcast_messages`
3. Processor picks up every 2 seconds
4. Queued to Redis by campaign ID
5. Workers send with campaign delays

### Sequence Flow:
1. Lead gets trigger → enrolled
2. ALL steps created as "pending" 
3. Time-based check every 15 seconds
4. When time arrives → message to `broadcast_messages`
5. Same Redis queue system as campaigns
6. Workers send with per-step delays

### Message Processing:
- Batch size: 5000 messages
- Processing interval: 2 seconds
- Worker pools: Auto-created per broadcast
- Cleanup: After 5 minutes idle
- Delays: Random between min/max

## 🆕 Group & Community Management (NEW - January 2025)

### ✅ Group Management Features:
- **Create Groups** with participants in one operation
- **Add/Remove Participants** to existing groups
- **Promote/Demote** participants (admin rights)
- **Get/Revoke Invite Links** for groups
- **Manage Group Settings** (icon, description)
- **Join Groups** via invite links

### ✅ Community Management Features:
- **Create Communities** (WhatsApp Communities)
- **Add Members** to communities (via announcement group)
- **Link/Unlink Groups** to/from communities
- **Get Community Info** and member lists

### 📌 API Endpoints:

#### Groups:
- `POST /group` - Create group with participants
- `POST /group/participants` - Add participants to group
- `POST /group/participants/remove` - Remove participants
- `POST /group/participants/promote` - Make admin
- `POST /group/participants/demote` - Remove admin
- `GET /group/participant-requests` - List join requests
- `POST /group/participant-requests/approve` - Approve requests
- `POST /group/participant-requests/reject` - Reject requests

#### Communities:
- `POST /community` - Create community
- `GET /community` - Get community info
- `POST /community/participants` - Add members
- `POST /community/link-group` - Link group to community
- `POST /community/unlink-group` - Unlink group

### 📖 Example Usage:
```bash
# Create a group with participants
curl -X POST http://localhost:3000/group \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Dev Team",
    "participants": ["+1234567890", "+0987654321"]
  }'

# Create a community
curl -X POST http://localhost:3000/community \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tech Community",
    "description": "A community for tech enthusiasts"
  }'
```

For complete API documentation, see [GROUP_COMMUNITY_API_DOCS.md](GROUP_COMMUNITY_API_DOCS.md)

## 🛠️ Configuration

### Campaign Delays:
```sql
-- In campaigns table
min_delay_seconds: 5   -- Minimum delay between messages
max_delay_seconds: 15  -- Maximum delay between messages
```

### Sequence Step Delays:
```sql
-- In sequence_steps table (per step!)
min_delay_seconds: 10  -- Step 1 might have 10-20 seconds
max_delay_seconds: 20
```

## 📈 Performance

- **Capacity**: 3000+ simultaneous devices
- **Throughput**: Limited only by delays
- **Memory**: ~1MB per worker
- **Redis**: ~500MB for queues
- **Processing**: 5000 messages per batch

## ⚠️ Important Notes

1. **Redis Required** - No Redis = No Start
2. **No Rate Limiting** - Only delays between messages
3. **Per-Step Delays** - Each sequence step uses its own settings
4. **Zombie Prevention** - Pools cleaned from registry properly

## 🐛 Troubleshooting

**System won't start**
- Check REDIS_URL is set
- Verify Redis is accessible

**Messages not sending**
- Check device online status
- Verify Redis has queued messages
- Check worker logs

**Wrong delays**
- Campaigns: Check `campaigns.min_delay_seconds`
- Sequences: Check `sequence_steps.min_delay_seconds`

## 📄 License

MIT License - see LICENSE file

---
**Working at scale with 3000+ devices!** 🚀