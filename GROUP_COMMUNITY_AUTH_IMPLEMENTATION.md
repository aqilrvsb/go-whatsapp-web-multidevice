# Authentication Implementation Summary for Group & Community APIs

## Changes Made

### 1. Updated REST Handlers with Authentication

#### `src/ui/rest/group.go`
- Added `middleware.CustomAuth()` to all group routes
- Implemented user authentication checks in each handler
- Added device ownership validation
- Ensures device is online before processing requests

#### `src/ui/rest/community.go`
- Added `middleware.CustomAuth()` to all community routes
- Implemented user authentication checks in each handler
- Added device ownership validation
- Ensures device is online before processing requests

### 2. Created Authentication Helper

#### `src/ui/rest/auth_helpers.go`
- Created `validateDeviceOwnership()` function
- Validates:
  - User owns the specified device
  - Device exists in the system
  - Device is online (connected to WhatsApp)
- Returns appropriate error responses

### 3. Updated Domain Models

#### `src/domains/group/group.go`
- Added `DeviceID` field to all request structs:
  - `JoinGroupWithLinkRequest`
  - `LeaveGroupRequest`
  - `GetGroupRequestParticipantsRequest`
  - `GroupRequestParticipantsRequest`

#### `src/domains/community/community.go`
- Added `DeviceID` field to all request structs:
  - `GetCommunityInfoRequest`
  - `LinkGroupRequest`
  - `UnlinkGroupRequest`

### 4. Documentation

#### `GROUP_COMMUNITY_AUTH_GUIDE.md`
- Comprehensive authentication guide
- Examples for all endpoints
- Error response documentation
- Migration guide for existing implementations

## How It Works

1. **Authentication Flow**:
   - Client sends request with authentication (cookie/header)
   - `CustomAuth()` middleware validates the token
   - User context is stored in the request

2. **Authorization Flow**:
   - Handler extracts user ID from context
   - Validates device ownership using `validateDeviceOwnership()`
   - Checks device is online
   - Processes the request if all checks pass

3. **Error Handling**:
   - 401 Unauthorized: No/invalid authentication
   - 403 Forbidden: User doesn't own the device
   - 400 Bad Request: Device offline or invalid IDs
   - 404 Not Found: Device doesn't exist

## API Changes

All group and community endpoints now require:

1. **Authentication**: Via cookie or header
2. **Device ID**: In request body/query params
3. **Device Ownership**: User must own the device
4. **Device Status**: Device must be online

## Example Usage

```bash
# With Bearer Token
curl -X POST http://localhost:3000/group \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "title": "New Group",
    "participants": ["+1234567890"]
  }'

# With Cookie (Web UI)
fetch('/group', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        device_id: 'device-uuid',
        title: 'New Group',
        participants: ['+1234567890']
    })
})
```

## Next Steps

1. **Test all endpoints** with authentication
2. **Update frontend code** to include device_id
3. **Handle authentication errors** in UI
4. **Update API documentation** if needed

## Benefits

1. **Security**: Only authenticated users can manage groups/communities
2. **Multi-tenancy**: Users can only use their own devices
3. **Reliability**: Ensures device is connected before operations
4. **Consistency**: Same auth pattern as other endpoints (send message, etc.)
