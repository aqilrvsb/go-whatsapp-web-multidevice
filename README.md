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
- ✅ **Message Sequences** - Automated drip campaigns with niche targeting
- ✅ **Broadcast Manager** - Optimized for 3,000+ devices
- ✅ **Device Rate Limiting** - Custom min/max delay per device
- ✅ **Campaign Triggers** - Auto-send based on date and niche matching
- ✅ **Worker Pool System** - Simultaneous message processing

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

## 📧 Message Sequences Feature

### What are Sequences?
Automated drip campaigns that send messages over multiple days. Each contact progresses through their own timeline.

### How it Works
1. **Create Sequence**: Define messages for Day 1, Day 2, etc.
2. **Set Send Times**: Each day can have specific send time
3. **Niche Targeting**: Auto-enroll leads based on their niche
4. **Individual Progress**: New contacts always start from Day 1

### Example Sequence
```
Day 1 (10:00 AM): Welcome message + introduction
Day 2 (2:00 PM): Product features + benefits  
Day 3 (11:00 AM): Customer testimonials
Day 4 (3:00 PM): Special offer
Day 5 (10:00 AM): Final reminder
```

## 🚀 Broadcast Optimization

### Device Workers
- Each device runs its own worker thread
- Custom delay settings (min/max seconds)
- Queue-based message processing
- Automatic retry on failure

### Performance Features
- **Worker Pool**: Up to 100 concurrent workers
- **Message Queue**: 1000 messages buffer per device
- **Rate Limiting**: Random delay between min/max
- **Health Monitoring**: Auto-restart unhealthy workers

### Configuration
```env
# Device delay settings (per device)
MIN_DELAY_SECONDS=5
MAX_DELAY_SECONDS=15
```

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
