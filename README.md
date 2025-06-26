# WhatsApp Multi-Device System - FINAL WORKING VERSION
**Last Updated: June 27, 2025**  
**Status: ✅ All features working on Railway**

## 🚀 Quick Deploy to Railway

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template?template=https%3A%2F%2Fgithub.com%2Faqilrvsb%2FWas-MCP&plugins=postgresql)

## 🎯 System Overview

A powerful WhatsApp Multi-Device system designed for:
- **200+ users** with **15 devices each** (3,000+ connections)
- **Broadcast messaging** to thousands of contacts
- **Real-time analytics** and tracking
- **Chat history storage**
- **Campaign management**

## ✅ Current Status (All Working)

### Core Features
- ✅ **Multi-user authentication** - Cookie-based sessions
- ✅ **Multi-device support** - Unlimited devices per user
- ✅ **WhatsApp Web integration** - Read-only chat viewer
- ✅ **Analytics dashboard** - Real-time metrics
- ✅ **Campaign calendar** - Schedule broadcasts
- ✅ **Chat storage** - Save all messages
- ✅ **Auto-reply** - Automatic responses
- ✅ **Webhooks** - Real-time notifications

### New Features (June 2025)
- ✅ **Message Sequences** - Automated drip campaigns with individual progress tracking
- ✅ **Broadcast Manager** - Optimized for 3,000+ devices with worker pools
- ✅ **Device Rate Limiting** - Custom min/max delay per device
- ✅ **Campaign Triggers** - Auto-send based on date and niche matching
- ✅ **Worker Pool System** - Parallel message processing with health monitoring
- ✅ **Sequence UI** - Full-featured interface for creating and managing sequences
- ✅ **Auto-enrollment** - Automatically add leads to sequences based on niche
- ✅ **Progress Tracking** - Each contact maintains their own timeline

### Fixed Issues (June 27, 2025)
- ✅ Build errors - Go 1.23, correct paths
- ✅ 502 errors - REST mode enabled
- ✅ Database connection - DB_URI mapping
- ✅ Authentication - Cookie sessions
- ✅ Campaign creation - Schema updated
- ✅ Device deletion - NULL handling
- ✅ JavaScript errors - Syntax fixes
- ✅ WhatsApp message storage - Fixed to capture both sent and received messages
- ✅ Chat sync functionality - Added manual and auto-sync features
- ✅ Foreign key constraint - Fixed campaign_id type mismatch
- ✅ Sequence initialization - Added missing usecase initialization
- ✅ Simplified sequences - Removed device requirement, simplified to match campaigns

## 📋 Environment Variables (Railway)

```env
# Database (Auto-set by Railway)
DB_URI=${{DATABASE_URL}}

# Application
APP_PORT=3000
APP_DEBUG=false
APP_OS=WhatsApp Business System
APP_BASIC_AUTH=admin:changeme123

# WhatsApp Features
WHATSAPP_CHAT_STORAGE=true
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_AUTO_REPLY=Thank you for contacting us!

# Optional Webhooks
WHATSAPP_WEBHOOK=https://your-webhook.com
WHATSAPP_WEBHOOK_SECRET=your-secret
```

## 🔧 Installation & Deployment

### Option 1: One-Click Railway Deploy
1. Click the Deploy button above
2. Railway will automatically:
   - Create PostgreSQL database
   - Set environment variables
   - Build and deploy the app

### Option 2: Manual Setup
```bash
# Clone repository
git clone https://github.com/aqilrvsb/Was-MCP.git
cd Was-MCP

# Deploy to Railway
railway login
railway new
railway add postgresql
railway up

# Set environment variables
railway variables set DB_URI='${{DATABASE_URL}}'
railway variables set WHATSAPP_CHAT_STORAGE=true
```

## 💻 Usage Guide

### 1. Access Dashboard
- URL: `https://your-app.up.railway.app`
- Login: `admin@whatsapp.com` / `changeme123`

### 2. Add WhatsApp Device
1. Go to **Devices** tab
2. Click **Add Device**
3. Scan QR code with WhatsApp
4. Device will show as "online"

### 3. View WhatsApp Chats
1. Click **WhatsApp Web** button on device
2. See all your chats in read-only mode
3. Messages are stored in database

### 4. Create Campaigns
1. Go to **Campaign** tab
2. Click any date on calendar
3. Fill in:
   - Title
   - Niche/Category
   - Message
   - Image (optional)
   - Time

### 5. Send Messages (Device Actions)
1. Click device name
2. Go to **Actions**
3. Test messaging features:
   - Send text
   - Send images
   - Check number status
   - Broadcast messages

## 🗄️ Database Schema

### Tables Created Automatically:
- `users` - User accounts
- `user_devices` - WhatsApp devices
- `campaigns` - Marketing campaigns
- `whatsapp_chats` - Chat metadata
- `whatsapp_messages` - Message history
- `message_analytics` - Tracking data

## 🔍 Troubleshooting

### Messages Not Showing in WhatsApp Web?
1. Ensure `WHATSAPP_CHAT_STORAGE=true` is set
2. Check if device is online
3. Send a test message to trigger sync
4. Click the "Sync" button in WhatsApp Web view
5. Check Railway logs for any errors

**Note**: Messages are now properly saved for both sent and received messages in personal chats.

### Campaign Creation Error?
The database schema is automatically updated on startup. If you still get errors:
```sql
-- Run this manually in PostgreSQL
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS niche VARCHAR(255);
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS image_url TEXT;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS scheduled_time TIME;
```

### 502 Errors?
- Application runs in REST mode automatically
- Check Railway logs for startup errors
- Ensure DATABASE_URL is provided by Railway

## 🚀 Performance & Scale

### Optimized for 3,000+ Devices:
- **Sharded architecture** - Distributes load
- **Message buffering** - Batch processing
- **Connection pooling** - Efficient resource use
- **In-memory caching** - Fast response times

### Recommended Railway Plan:
- **Pro plan** for production use
- **2+ GB RAM** for 3,000 devices
- **PostgreSQL** with connection pooling

## 📡 Webhook Integration

When `WHATSAPP_WEBHOOK` is set, you'll receive:
```json
{
  "event": "message",
  "data": {
    "deviceId": "uuid",
    "from": "+1234567890",
    "message": "Hello!",
    "timestamp": "2025-06-27T10:00:00Z"
  }
}
```

## 🛠️ API Endpoints

### Authentication
- `POST /login` - User login
- `POST /register` - Create account
- `POST /logout` - Logout

### Devices
- `GET /api/devices` - List devices
- `POST /api/devices` - Add device
- `DELETE /api/devices/:id` - Delete device

### WhatsApp
- `GET /app/qr` - Get QR code
- `POST /send/message` - Send text
- `POST /send/image` - Send image
- `GET /device/:id/whatsapp` - Web view

### Analytics
- `GET /api/analytics/:days` - Get metrics
- `GET /api/campaigns` - List campaigns
- `POST /api/campaigns` - Create campaign

### Sequences (NEW!)
- `GET /api/sequences` - List sequences
- `POST /api/sequences` - Create sequence
- `GET /api/sequences/:id` - Get sequence details
- `PUT /api/sequences/:id` - Update sequence
- `DELETE /api/sequences/:id` - Delete sequence
- `POST /api/sequences/:id/contacts` - Add contacts
- `POST /api/sequences/:id/start` - Start sequence
- `POST /api/sequences/:id/pause` - Pause sequence

## 📧 Message Sequences Feature (Updated June 27, 2025)

### What are Sequences?
Simplified multi-day campaigns similar to regular campaigns but with automatic progression. Each contact maintains their own timeline - perfect for onboarding and follow-up sequences.

### Simplified Features
1. **Multi-Day Messages**: Create sequences with unlimited days
2. **Individual Progress**: Each contact starts from Day 1 regardless of when added
3. **Simple Content**: Text message + optional image URL (just like campaigns)
4. **Per-Message Delays**: Min/max delay in seconds between messages
5. **Niche Matching**: Auto-enroll contacts based on niche
6. **Status Control**: Active, Paused, or Draft states

### How it Works
1. **Create Sequence**: 
   - Name and description
   - Set niche for auto-enrollment
   - Add days with messages and delays

2. **Add Contacts**:
   - Manual: Add phone numbers
   - Automatic: Contacts with matching niche

3. **Message Delivery**:
   - Each message sent with random delay (min/max seconds)
   - Next day message sent 24 hours after previous
   - Respects per-device rate limits

### Example Sequences

**Sales Sequence** (5 days):
```
Day 1: Welcome! Check out our special offer... [5-15 sec delay]
Day 2: Did you see our products? Here's 10% off... [5-15 sec delay]
Day 3: Customer success story + testimonial... [5-15 sec delay]
Day 4: Limited time - 20% discount code... [5-15 sec delay]
Day 5: Final reminder - offer expires tonight! [5-15 sec delay]
```

**Onboarding Sequence** (3 days):
```
Day 1: Thanks for joining! Here's how to start... [10-20 sec delay]
Day 2: Pro tip: Try this feature... [10-20 sec delay]
Day 3: Need help? Contact our support... [10-20 sec delay]
```

## 🚀 Broadcast System Architecture

### Optimized for 200 Users × 15 Devices = 3,000+ Connections

### Device Workers
- **Individual Workers**: Each device runs its own message worker
- **Parallel Processing**: Up to 100 concurrent workers system-wide
- **Custom Rate Limiting**: Each device has min/max delay settings
- **Queue Management**: 1000 message buffer per device
- **Health Monitoring**: Auto-restart stuck workers

### Message Flow
1. **Campaign/Sequence Trigger** → Message created in database
2. **Broadcast Manager** → Routes message to appropriate device worker
3. **Device Worker** → Queues message with rate limiting
4. **WhatsApp Send** → Message sent with random delay
5. **Status Update** → Database updated with result

### Performance Features
- **Sharded Architecture**: Distributes load across devices
- **Message Buffering**: Prevents overwhelming WhatsApp
- **Connection Pooling**: Efficient database usage
- **In-Memory Queues**: Fast message processing
- **Automatic Retries**: Failed messages retry with backoff

### Rate Limiting Strategy
```
Per Device Settings:
- Min Delay: 5-10 seconds (configurable)
- Max Delay: 15-30 seconds (configurable)
- Random delay between min/max for each message
- Prevents pattern detection and bans
```

### Scaling Guidelines
- **Small (< 500 devices)**: 2GB RAM, 2 CPU cores
- **Medium (500-2000 devices)**: 4GB RAM, 4 CPU cores
- **Large (2000-5000 devices)**: 8GB RAM, 8 CPU cores
- **Database**: PostgreSQL with connection pooling

### Anti-Ban Best Practices
1. **Variable Delays**: Random delay between messages
2. **Human-like Patterns**: Different delays per device
3. **Message Variety**: Mix text, images, videos
4. **Gradual Ramp-up**: Start slow with new devices
5. **Monitor Failed**: Track and adjust if high failure rate

## 🎉 Summary

This WhatsApp Multi-Device system is production-ready with:
- ✅ Stable connections for 3,000+ devices
- ✅ Real-time message tracking
- ✅ Complete chat history
- ✅ Broadcast capabilities
- ✅ Analytics dashboard
- ✅ Campaign management
- ✅ Message sequences with niche targeting
- ✅ Optimized broadcasting with device workers
- ✅ Automatic triggers for campaigns and sequences

## 🛠️ Implementation Guide

### Setting Up Sequences

1. **Create a Sequence**
   - Navigate to Sequences tab
   - Click "Create New Sequence"
   - Define messages for each day with specific send times
   - Set niche for auto-enrollment

2. **Add Contacts**
   ```bash
   POST /api/sequences/{id}/contacts
   {
     "contacts": ["+1234567890", "+0987654321"]
   }
   ```

3. **Auto-Enrollment by Niche**
   - Leads with matching niche are automatically enrolled
   - Processed every minute by background worker

### Broadcast Optimization

1. **Device Configuration**
   ```sql
   UPDATE user_devices 
   SET min_delay_seconds = 10, max_delay_seconds = 30 
   WHERE id = 'device-id';
   ```

2. **Monitor Workers**
   - Check `/api/broadcast/stats` for worker status
   - Auto-restart on failure
   - Health checks every 30 seconds

### Campaign Automation

1. **Schedule Campaign**
   - Set date and time in campaign calendar
   - Assign device and niche
   - System auto-sends at scheduled time

2. **Track Progress**
   - Real-time status updates
   - Message delivery tracking
   - Failed message retry

## 📈 Scaling Guidelines

### For 3,000+ Devices:
1. **Database**: Use PostgreSQL with connection pooling
2. **Memory**: Minimum 4GB RAM for worker pool
3. **CPU**: 4+ cores recommended
4. **Network**: Stable connection for concurrent messaging

### Performance Tuning:
- Adjust `maxWorkers` in broadcast manager (default: 100)
- Configure message queue buffer size (default: 1000)
- Set appropriate delay ranges per device type
- Monitor and adjust based on WhatsApp response

## 🐛 Troubleshooting

### Common Issues:
1. **Import Cycle Error**: Fixed by moving types to domain layer
2. **Worker Stuck**: Auto-restart after 10 minutes of inactivity
3. **Queue Full**: Increase buffer size or add more workers
4. **High Ban Rate**: Increase delay settings, reduce concurrent messages

**Support**: Create an issue on GitHub for help!
 
## Deployment Trigger - Fri 27/06/2025  1:53:49.15 


## Latest Update - June 27, 2025

### Sequence System Simplified
The sequence system has been simplified to match the campaign structure:
- **No device selection required** - System automatically selects available devices
- **Simple message format** - Text + optional image URL only
- **Per-message delays** - Min/max seconds delay between messages
- **Auto-enrollment** - Based on niche matching
- **No complex scheduling** - Messages sent with delays, next day = 24 hours later

### Key Improvements
1. **Fixed nil pointer error** - Added missing sequence usecase initialization
2. **Fixed foreign key constraint** - Changed campaign_id to INTEGER type
3. **Simplified UI** - Removed unnecessary options
4. **Better performance** - Streamlined message processing

### Usage
1. Go to Dashboard → Sequences tab
2. Create sequence with name, description, and niche
3. Add days with messages and delay settings
4. Contacts are auto-enrolled based on niche
5. Each contact progresses individually through the sequence

---