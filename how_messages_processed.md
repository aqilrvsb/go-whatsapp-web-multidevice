# HOW SEQUENCES AND CAMPAIGNS PROCESS PENDING MESSAGES

## 1. UNIFIED PROCESSING - BOTH USE SAME SYSTEM

### GetPendingMessages Query (broadcast_repository.go):
```sql
WHERE bm.device_id = ? 
AND bm.status = 'pending'
AND bm.scheduled_at IS NOT NULL
AND bm.scheduled_at <= ?  -- NOW()
```

**KEY POINTS:**
- Both sequences and campaigns are fetched by the SAME query
- No distinction between campaign_id or sequence_id
- Only checks: status='pending' AND scheduled_at <= NOW()

## 2. ULTRA OPTIMIZED BROADCAST PROCESSOR

### Processing Flow:
1. Gets all devices with pending messages
2. For each device, calls GetPendingMessagesAndLock()
3. Checks if device is online (skips if offline for WhatsApp Web)
4. Creates broadcast pools for campaigns/sequences
5. Queues messages to appropriate pool

### Key Code:
```go
// Same processing for both
if msg.CampaignID != nil {
    broadcastType = "campaign"
    broadcastID = fmt.Sprintf("%d", *msg.CampaignID)
} else if msg.SequenceID != nil {
    broadcastType = "sequence"
    broadcastID = *msg.SequenceID
}

// Queue to same system
err = p.manager.QueueMessageToBroadcast(broadcastType, broadcastID, &msg)
```

## 3. DELAY HANDLING - EXACT SAME FOR BOTH

```sql
COALESCE(
    c.min_delay_seconds,     -- Campaign delay
    ss.min_delay_seconds,    -- Sequence step delay
    s.min_delay_seconds,     -- Sequence delay
    10                       -- Default
) AS min_delay
```

## SUMMARY: THEY USE IDENTICAL PROCESSING!
- Same GetPendingMessages query
- Same broadcast processor
- Same pool manager
- Same delay logic
- Only difference: campaign uses campaign_id, sequence uses sequence_id
