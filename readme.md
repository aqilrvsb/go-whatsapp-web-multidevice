# WhatsApp Analytics Multi-Device Dashboard

**Last Updated: June 24, 2025 - 3:00 PM**  
**Latest Feature: Complete Authentication & Dashboard Fix**

## 🚀 Latest Updates:

### Complete System Fix (June 24, 2025)
- **Fixed Authentication**: Cookie-based sessions with `credentials: 'include'`
- **Fixed Device Filter**: Dropdown now shows "All Devices" option
- **Fixed JavaScript Errors**: Proper function spacing and definitions
- **Added Phone Linking**: New endpoint to link phone numbers to devices
- **Railway Deployment**: Successfully forced rebuild to deploy embedded views

### Authentication System (June 24, 2025)
- **Base64 Password Storage**: Replaced bcrypt with base64 encoding for easy password viewing
- **Cookie-Based Sessions**: Simple session management using HTTP cookies
- **No Token Management**: Login once, stay logged in until logout
- **Fixed Build Errors**: Resolved unused import issues
- **Working Login System**: Authentication now works properly

### PostgreSQL Database Integration (June 23, 2025)
- **Full PostgreSQL Support**: Migrated from file-based storage to PostgreSQL
- **Railway Integration**: Configured for Railway PostgreSQL deployment
- **Database Schema**: Includes tables for users, devices, sessions, and message analytics
- **Connection Pooling**: Optimized for 200+ concurrent users
- **Auto-deployment**: Push to main branch triggers Railway deployment

## 🎯 Key Features:

### 1. Multi-User System (200+ Users Support)
- **User Management**: Proper user registration and authentication
- **Session Management**: Secure token-based sessions
- **User Isolation**: Each user sees only their own data
- **Scalable Architecture**: PostgreSQL-backed for high concurrency
- **Persistent Storage**: All data stored in PostgreSQL database

### 2. Real-Time Dashboard Auto-Refresh
- **10-Second Auto-Refresh**: Dashboard updates every 10 seconds
- **Toggle Control**: Enable/disable auto-refresh
- **Manual Refresh**: Button for instant refresh
- **Smart Updates**: Silent refresh without UI flicker
- **Visibility API**: Pauses refresh when tab is hidden

### 3. Lead Analytics Dashboard
- **Email-based Analytics**: Each user gets their own analytics
- **Real WhatsApp Data**: Tracks actual message status
- **Lead Metrics**:
  - Active/Inactive Devices per user
  - Leads Sent, Received (with %)
  - Leads Not Received (with %)
  - Leads Read/Not Read (with %)
  - Leads Replied (with %)
- **Device Filter**: Filter analytics by device
- **Time Ranges**: Today, 7, 30, 90 days, or custom

### 4. Default Admin Account
- Email: `admin@whatsapp.com`
- Password: `changeme123`

### 5. Performance Optimizations
- Efficient data structure for 200+ users
- Optimized refresh mechanism
- Minimal server load with smart caching

## 📊 Database Schema

### PostgreSQL Tables:
1. **users** - User accounts and authentication (passwords stored as base64)
2. **user_devices** - WhatsApp devices per user
3. **user_sessions** - Active user sessions
4. **message_analytics** - Message tracking and analytics

### Connection Details:
- Database is hosted on Railway PostgreSQL
- Connection pooling configured for high performance
- Automatic cleanup of expired sessions

## 🔐 Authentication System

### Password Storage:
- Passwords are stored as base64 encoded strings (not hashed)
- This allows admins to view user passwords if needed
- Perfect for personal/internal systems where security isn't critical
- Example: password `aqil@gmail.com` → stored as `YXFpbEBnbWFpbC5jb20=`

### Session Management:
- Cookie-based authentication (no complex tokens)
- Login once, stay logged in until logout
- Sessions persist across browser refreshes
- Logout endpoint: `/logout`

## 📈 Current Status (June 24, 2025 - 3:00 PM)

### ✅ Working Features:
- User registration with base64 passwords
- User login with cookie sessions
- Dashboard loads successfully
- PostgreSQL database integration
- Railway deployment (after forcing rebuild)
- QR code display for WhatsApp connection
- Device filter dropdown with "All Devices" option

### ⚠️ In Progress:
- Linking phone numbers to devices
- Connecting actual WhatsApp accounts
- Real-time message analytics

### 🔧 Recent Fixes Applied:
1. **JavaScript Errors Fixed**:
   - Added proper spacing between functions
   - Fixed "updateDeviceFilter is not defined" error
   - Fixed "startAutoRefresh is not defined" error

2. **Authentication Fixed**:
   - Added `credentials: 'include'` to all fetch calls
   - Cookie-based sessions working properly
   - No more 401 errors on API calls

3. **Device Filter Fixed**:
   ```javascript
   deviceFilter.innerHTML = '<option value="all">All Devices</option>';
   ```

4. **Railway Deployment Fixed**:
   - Discovered HTML views are embedded in Go binary
   - Forced complete rebuild to update embedded files
   - Deployment now works correctly

### 🔄 Latest Code Changes:
- Added `/app/link-device` endpoint for linking phone numbers
- Updated all API calls to include credentials
- Fixed function definitions in dashboard.html
- Added console logging for debugging

## 🚀 Deployment

### Railway Auto-Deployment:
1. Push to `main` branch at `https://github.com/aqilrvsb/Was-MCP.git`
2. Railway automatically builds and deploys
3. Uses PostgreSQL database from Railway
4. Environment variables configured in Railway dashboard

### Required Environment Variables:
```
DB_URI=postgresql://user:pass@host:port/database?sslmode=require
PORT=3000                    # Railway provides this automatically
APP_PORT=3000               # Fallback if PORT not set
APP_BASIC_AUTH=admin:changeme123
APP_DEBUG=false
APP_OS=Chrome
APP_CHAT_FLUSH_INTERVAL=7
WHATSAPP_AUTO_REPLY=
WHATSAPP_WEBHOOK=
WHATSAPP_WEBHOOK_SECRET=
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_CHAT_STORAGE=true
```

### Troubleshooting 502 Error:
If you're seeing a 502 error, check:
1. **Database Connection**: Ensure DB_URI is correct and PostgreSQL is accessible
2. **Environment Variables**: All required variables must be set in Railway
3. **Application Logs**: Check Railway logs for startup errors
4. **Port Configuration**: Railway automatically sets PORT env variable
5. **Database Migration**: The app creates tables on first run - check if it has permissions

### Current Status (June 23, 2025):
- ✅ Code fixed and builds successfully
- ✅ PostgreSQL integration complete
- ✅ Auto-deployment configured
- ⚠️ 502 error indicates runtime issue - check Railway logs

## Login Credentials
- **Admin Account**: 
  - Email: `admin@whatsapp.com`
  - Password: `changeme123` (or whatever you set in APP_BASIC_AUTH environment variable)
- **Registered Users**: Can register via `/register` page and login with their email

[![Patreon](https://img.shields.io/badge/Support%20on-Patreon-orange.svg)](https://www.patreon.com/c/aldinokemal)  
**If you're using this tools to generate income, consider supporting its development by becoming a Patreon member!**  
Your support helps ensure the library stays maintained and receives regular updates!
___

![release version](https://img.shields.io/github/v/release/aldinokemal/go-whatsapp-web-multidevice)

## 🚀 New Features - Analytics Dashboard with Real Data

### Multi-User Analytics Dashboard
- **Real WhatsApp Data**: Analytics pulled from actual message history
- **User Management**: Register and manage multiple users
- **Device Management**: Each user can manage multiple WhatsApp devices
- **Live Analytics**: Real-time message statistics from chat storage
- **Custom Date Ranges**: Analyze any time period with real data

![Build Image](https://github.com/aldinokemal/go-whatsapp-web-multidevice/actions/workflows/build-docker-image.yaml/badge.svg)

![release windows](https://github.com/aldinokemal/go-whatsapp-web-multidevice/actions/workflows/release-windows.yml/badge.svg)
![release linux](https://github.com/aldinokemal/go-whatsapp-web-multidevice/actions/workflows/release-linux.yml/badge.svg)
![release macos](https://github.com/aldinokemal/go-whatsapp-web-multidevice/actions/workflows/release-mac.yml/badge.svg)

## 🚀 New Features - Analytics Dashboard

### Multi-User Analytics Dashboard
- **User Management**: Register and manage multiple users
- **Device Management**: Each user can manage multiple WhatsApp devices
- **Analytics Dashboard**: View message statistics, active chats, and trends
- **Real-time Updates**: Live device status and message tracking

### Dashboard Features
- 📊 **Analytics Overview**: Messages sent/received, active chats, reply rate
- 📱 **Multi-Device Support**: Add, manage, and monitor multiple WhatsApp devices
- 📈 **Time-based Analytics**: 
  - Today view
  - Preset ranges: 7, 30, or 90 days
  - **Custom date range**: Select any start and end date
- 📅 **Date Range Selector**: Pick specific date ranges for detailed analysis
- 🔐 **User Authentication**: Secure login and registration system
- 👥 **User Registration**: New users can create accounts
- 🎨 **Modern UI**: Clean, responsive design with WhatsApp's color scheme
- 📊 **Real-time Updates**: Live statistics and device status monitoring

## Support for `ARM` & `AMD` Architecture along with `MCP` Support

Download:

- [Release](https://github.com/aldinokemal/go-whatsapp-web-multidevice/releases/latest)
- [Docker Image](https://hub.docker.com/r/aldinokemal2104/go-whatsapp-web-multidevice/tags)

## Breaking Changes

- `v6`
  - For REST mode, you need to run `<binary> rest` instead of `<binary>`
    - for example: `./whatsapp rest` instead of ~~./whatsapp~~
  - For MCP mode, you need to run `<binary> mcp`
    - for example: `./whatsapp mcp`

## Feature

- **NEW: Analytics Dashboard with Multi-User & Multi-Device Support**
- Send WhatsApp message via http API, [docs/openapi.yml](./docs/openapi.yaml) for more details
- **MCP (Model Context Protocol) Server Support** - Integrate with AI agents and tools using standardized protocol
- Mention someone
  - `@phoneNumber`
  - example: `Hello @628974812XXXX, @628974812XXXX`
- Post Whatsapp Status
- Compress image before send
- Compress video before send
- Change OS name become your app (it's the device name when connect via mobile)
  - `--os=Chrome` or `--os=MyApplication`
- Basic Auth (able to add multi credentials)
  - `--basic-auth=kemal:secret,toni:password,userName:secretPassword`, or you can simplify
  - `-b=kemal:secret,toni:password,userName:secretPassword`
- Customizable port and debug mode
  - `--port 8000`
  - `--debug true`
- Auto reply message
  - `--autoreply="Don't reply this message"`