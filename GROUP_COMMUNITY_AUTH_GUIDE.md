# Group & Community API Authentication Guide

This guide explains how authentication works for the Group and Community management APIs in the WhatsApp Multi-Device system.

## Overview

All Group and Community endpoints now require authentication to ensure:
1. **User Authentication**: Only logged-in users can access these endpoints
2. **Device Ownership**: Users can only perform actions through devices they own
3. **Device Status**: The device must be online (connected to WhatsApp)

## Authentication Methods

The system supports three authentication methods:

### 1. Cookie-based Authentication (Web UI)
```javascript
fetch('/group', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',  // Includes session_token cookie
    body: JSON.stringify({
        device_id: 'your-device-uuid',
        title: 'New Group',
        participants: ['+1234567890']
    })
})
```

### 2. Bearer Token (API)
```bash
curl -X POST http://localhost:3000/group \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "your-device-uuid",
    "title": "New Group",
    "participants": ["+1234567890"]
  }'
```

### 3. X-Auth-Token Header
```bash
curl -X POST http://localhost:3000/group \
  -H "X-Auth-Token: your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "your-device-uuid",
    "title": "New Group",
    "participants": ["+1234567890"]
  }'
```

## Getting a Session Token

First, authenticate to get a session token:

```bash
# Login
curl -X POST http://localhost:3000/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your-email@example.com",
    "password": "your-password"
  }'

# Response
{
  "status": 200,
  "code": "SUCCESS",
  "message": "Login successful",
  "results": {
    "token": "your-session-token-here",
    "user": { ... }
  }
}
```

## Group Management Examples

### Create a Group
```bash
curl -X POST http://localhost:3000/group \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "title": "Dev Team",
    "participants": ["+1234567890", "+0987654321"]
  }'
```

### Add Participants
```bash
curl -X POST http://localhost:3000/group/participants \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "group_id": "123456789@g.us",
    "participants": ["+1112223333"]
  }'
```

### Leave Group
```bash
curl -X POST http://localhost:3000/group/leave \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "group_id": "123456789@g.us"
  }'
```

## Community Management Examples

### Create a Community
```bash
curl -X POST http://localhost:3000/community \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "name": "Tech Community",
    "description": "A community for tech enthusiasts"
  }'
```

### Add Participants to Community
```bash
curl -X POST http://localhost:3000/community/participants \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "community_id": "123456789@g.us",
    "participants": ["+1234567890", "+0987654321"]
  }'
```

### Link Group to Community
```bash
curl -X POST http://localhost:3000/community/link-group \
  -H "Authorization: Bearer your-session-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "community_id": "123456789@g.us",
    "group_id": "987654321@g.us"
  }'
```

## Error Responses

### Authentication Errors

**No Token Provided:**
```json
{
  "status": 401,
  "code": "UNAUTHORIZED",
  "message": "Authentication required - no token provided"
}
```

**Invalid Token:**
```json
{
  "status": 401,
  "code": "UNAUTHORIZED",
  "message": "Invalid session - token not found or expired"
}
```

### Authorization Errors

**Device Not Found:**
```json
{
  "status": 404,
  "code": "DEVICE_NOT_FOUND",
  "message": "Device not found"
}
```

**Device Not Owned:**
```json
{
  "status": 403,
  "code": "FORBIDDEN",
  "message": "You don't have permission to use this device"
}
```

**Device Offline:**
```json
{
  "status": 400,
  "code": "DEVICE_OFFLINE",
  "message": "Device is not connected. Please connect the device first."
}
```

## JavaScript/Frontend Integration

For web applications, include credentials in fetch requests:

```javascript
// Group Management Example
async function createGroup(deviceId, title, participants) {
    try {
        const response = await fetch('/group', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',  // Important: includes cookies
            body: JSON.stringify({
                device_id: deviceId,
                title: title,
                participants: participants
            })
        });
        
        const data = await response.json();
        
        if (data.code === 'SUCCESS') {
            console.log('Group created:', data.results.group_id);
        } else {
            console.error('Error:', data.message);
        }
    } catch (error) {
        console.error('Request failed:', error);
    }
}

// Community Management Example
async function createCommunity(deviceId, name, description) {
    try {
        const response = await fetch('/community', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({
                device_id: deviceId,
                name: name,
                description: description
            })
        });
        
        const data = await response.json();
        
        if (data.code === 'SUCCESS') {
            console.log('Community created:', data.results.community_id);
        } else {
            console.error('Error:', data.message);
        }
    } catch (error) {
        console.error('Request failed:', error);
    }
}
```

## Important Notes

1. **Device ID Required**: All endpoints now require a `device_id` parameter
2. **Device Ownership**: The authenticated user must own the device
3. **Device Status**: The device must be online (connected to WhatsApp)
4. **Phone Format**: Phone numbers should include country code (e.g., +1234567890)
5. **Group ID Format**: Group IDs use the format `123456789@g.us`

## Security Best Practices

1. **Store tokens securely**: Never expose session tokens in client-side code
2. **Use HTTPS**: Always use HTTPS in production
3. **Token expiration**: Tokens expire - handle 401 errors by re-authenticating
4. **Validate inputs**: Always validate user inputs before sending requests
5. **Rate limiting**: Be aware of WhatsApp's rate limits to avoid bans

## Migration Guide

If you're upgrading from the previous version without authentication:

1. Add `device_id` to all request payloads
2. Include authentication headers/cookies in all requests
3. Handle 401/403 error responses appropriately
4. Update frontend code to include `credentials: 'include'`

## Troubleshooting

**"Authentication required" error:**
- Ensure you're including the session token
- Check if the token has expired
- Verify the token format

**"Device not found" error:**
- Verify the device_id is correct
- Check if the device exists in the system

**"Permission denied" error:**
- Ensure the device belongs to the authenticated user
- Check if the user has the necessary permissions

**"Device offline" error:**
- Connect the device through the dashboard
- Wait for the device status to become "online"
