# WhatsApp Analytics Multi-Device Dashboard

**Last Updated: June 23, 2025**  
**Latest Fix: Implemented email-based login system with registration support**

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