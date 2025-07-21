# WhatsApp Group and Community Management API

This document describes the new Group and Community management endpoints added to the go-whatsapp-web-multidevice project.

## Group Management Endpoints

### 1. Create Group with Participants
Creates a new group and optionally adds participants in one operation.

**Endpoint:** `POST /group`

**Request Body:**
```json
{
  "title": "My New Group",
  "participants": [
    "+1234567890",
    "+0987654321"
  ]
}
```

**Response:**
```json
{
  "status": 200,
  "code": "SUCCESS",
  "message": "Success created group with id 123456789@g.us",
  "results": {
    "group_id": "123456789@g.us",
    "participants_status": [
      {
        "participant": "+1234567890",
        "status": "success",
        "message": "Added to group successfully"
      },
      {
        "participant": "+0987654321",
        "status": "success",
        "message": "Added to group successfully"
      }
    ]
  }
}
```

### 2. Add Participants to Existing Group
Adds one or more participants to an existing group.

**Endpoint:** `POST /group/participants`

**Request Body:**
```json
{
  "group_id": "123456789@g.us",
  "participants": [
    "+1234567890",
    "+0987654321"
  ]
}
```

### 3. Get Group Invite Link
Gets the invite link for a group.

**Endpoint:** `GET /group/invite-link?group_id=123456789@g.us`

### 4. Revoke Group Invite Link
Revokes the current invite link and generates a new one.

**Endpoint:** `POST /group/invite-link/revoke`

**Request Body:**
```json
{
  "group_id": "123456789@g.us"
}
```

## Community Management Endpoints

### 1. Create Community
Creates a new WhatsApp community.

**Endpoint:** `POST /community`

**Request Body:**
```json
{
  "name": "My Community",
  "description": "This is my awesome community"
}
```

**Response:**
```json
{
  "status": 200,
  "code": "SUCCESS",
  "message": "Successfully created community with ID 123456789@g.us",
  "results": {
    "community_id": "123456789@g.us"
  }
}
```

### 2. Add Participants to Community
Adds participants to a community (they will be added to the announcement group).

**Endpoint:** `POST /community/participants`

**Request Body:**
```json
{
  "community_id": "123456789@g.us",
  "participants": [
    "+1234567890",
    "+0987654321"
  ]
}
```

**Response:**
```json
{
  "status": 200,
  "code": "SUCCESS",
  "message": "Participants processed",
  "results": [
    {
      "participant": "+1234567890",
      "status": "success",
      "message": "Added to community successfully"
    },
    {
      "participant": "+0987654321",
      "status": "failed",
      "message": "User not on WhatsApp"
    }
  ]
}
```

### 3. Link Group to Community
Links an existing group to a community.

**Endpoint:** `POST /community/link-group`

**Request Body:**
```json
{
  "community_id": "123456789@g.us",
  "group_id": "987654321@g.us"
}
```

### 4. Unlink Group from Community
Unlinks a group from a community.

**Endpoint:** `POST /community/unlink-group`

**Request Body:**
```json
{
  "group_id": "987654321@g.us"
}
```

### 5. Get Community Info
Retrieves information about a community.

**Endpoint:** `GET /community?community_id=123456789@g.us`

## Important Notes

1. **Phone Number Format**: All phone numbers should include the country code (e.g., +1234567890).

2. **JID Format**: Group and Community IDs are in JID format (e.g., 123456789@g.us).

3. **Permissions**: The WhatsApp account connected must have admin permissions to:
   - Create groups/communities
   - Add participants
   - Link/unlink groups

4. **Rate Limits**: WhatsApp has rate limits on these operations. Avoid making too many requests in a short time.

5. **Community Limitations**:
   - A community can have up to 50 groups (plus the announcement group)
   - Each group can have up to 1024 participants
   - Only admins can add participants to communities

## Error Handling

All endpoints return standardized error responses:

```json
{
  "status": 400,
  "code": "VALIDATION_ERROR",
  "message": "Invalid phone number format"
}
```

Common error codes:
- `VALIDATION_ERROR`: Invalid request data
- `NOT_AUTHORIZED`: No permission to perform action
- `NOT_FOUND`: Group/Community not found
- `RATE_LIMITED`: Too many requests
