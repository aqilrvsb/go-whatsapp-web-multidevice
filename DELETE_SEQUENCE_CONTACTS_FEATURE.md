# Delete Sequence Contacts Feature

## Overview
This feature adds a "Delete Contact Trigger" button to each sequence that allows deleting broadcast messages based on their status (pending/sent/failed).

## Implementation

### 1. Backend Implementation

#### API Endpoints:
```
DELETE /sequence/{sequence_id}/contacts
Body: { "status": "pending|sent|failed" }

DELETE /sequence/{sequence_id}/contacts/all
```

#### Database Query:
```sql
-- Delete by status
DELETE FROM broadcast_messages 
WHERE sequence_id = $1 AND status = $2

-- Delete all
DELETE FROM broadcast_messages 
WHERE sequence_id = $1
```

### 2. Frontend Implementation

The UI adds a "Delete Contacts" button to each sequence card that:
1. Opens a modal with status options
2. Allows selecting: Pending, Sent, Failed, or All
3. Confirms deletion and shows result

### 3. Usage

1. Navigate to Message Sequences page
2. Click "Delete Contacts" button on any sequence
3. Select which status to delete:
   - **Pending**: Messages scheduled but not yet sent
   - **Sent**: Messages already sent successfully
   - **Failed**: Messages that failed to send
   - **All**: All messages regardless of status
4. Click "Delete Contacts" to confirm

### 4. Files Created/Modified

1. `src/usecase/sequence_delete_contacts.go` - Backend logic
2. `src/domains/app/sequence_delete_handler.go` - API handlers
3. `sequence_delete_contacts_ui.js` - Frontend JavaScript

### 5. Integration Steps

To integrate into your existing UI:

1. Add the JavaScript file to your sequences page:
```html
<script src="sequence_delete_contacts_ui.js"></script>
```

2. Or copy the JavaScript code directly into your existing sequences page.

3. Add the API routes to your router:
```go
r.Route("/sequence/{sequence_id}", func(r chi.Router) {
    r.Delete("/contacts", h.DeleteSequenceContacts)
    r.Delete("/contacts/all", h.DeleteAllSequenceContacts)
})
```

### 6. Security Considerations

- Add authentication middleware to protect these endpoints
- Consider adding confirmation dialog for large deletions
- Log all delete operations for audit trail

### 7. Testing

Test scenarios:
1. Delete pending messages only
2. Delete sent messages only
3. Delete failed messages only
4. Delete all messages
5. Delete from empty sequence (should show 0 deleted)

## Benefits

1. **Clean up pending messages** - Remove scheduled messages that haven't been sent
2. **Retry failed messages** - Delete failed messages to re-enroll contacts
3. **Reset sequences** - Clear all messages to start fresh
4. **Status-specific cleanup** - Target specific message states

## Notes

- This permanently deletes messages from broadcast_messages table
- Deleted messages cannot be recovered
- Does not affect sequence_contacts table (which is no longer used with Direct Broadcast)
- Useful for testing and managing sequence enrollments
