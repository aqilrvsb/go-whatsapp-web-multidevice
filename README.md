# WhatsApp Multi-Device System - Dual Database Edition
**Last Updated: July 30, 2025 - MySQL Application Data + PostgreSQL WhatsApp Sessions**  
**Status: ✅ Production-ready with Dual Database Support**

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

## 🔧 Environment Configuration

### Railway Deployment
Set these environment variables in Railway:

```env
# PostgreSQL for WhatsApp Sessions (Railway provides this automatically)
DB_URI=postgres://user:password@host:port/database?sslmode=require

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
2. Import into MySQL using the schema in `emergency_db_fix/mysql_migration/`
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
