# Sequence Message Greeting Fix

## Problem
Sequence messages were not showing the Malaysian greeting (Hi/Hello/Salam + name) and were not properly formatted with line breaks.

## Root Cause
The `GetPendingMessages` function in `broadcast_repository.go` was retrieving the message content from the database into the `Content` field, but the greeting processor in `whatsapp_message_sender.go` was looking for the `Message` field.

## Solution Applied
Updated two functions in `src/repository/broadcast_repository.go`:

1. **GetPendingMessages** - Added after scanning the row:
```go
// CRITICAL FIX: Ensure Message field is populated for the greeting processor
msg.Message = msg.Content
msg.ImageURL = msg.MediaURL // Also ensure ImageURL alias is set
```

2. **GetAllPendingMessages** - Added after scanning the row:
```go
// CRITICAL FIX: Ensure Message field is populated for the greeting processor
msg.Message = msg.Content
msg.ImageURL = msg.MediaURL // Also ensure ImageURL alias is set
msg.RecipientName = msg.RecipientPhone // Set recipient name to phone if missing
```

## Expected Result
After this fix, sequence messages should now:
1. Include Malaysian greetings (Hi/Hello/Salam + recipient name)
2. Have proper line breaks between greeting and message content
3. Show recipient name (or "Cik" if name is missing/phone number)

## Files Modified
- `src/repository/broadcast_repository.go`

## Next Steps
1. Rebuild the application
2. Deploy the changes
3. Test with a sequence message to verify the greeting appears correctly
