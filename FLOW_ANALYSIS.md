# COMPLETE FLOW ANALYSIS

## SEQUENCE FLOW (Working):

1. **Trigger Processing** (every 5 minutes):
   - `StartSequenceTriggerProcessor()` → `ProcessSequenceTriggers()`
   - Finds leads with triggers matching sequence entry points
   - Query: `leads` JOIN `sequences` JOIN `sequence_steps` WHERE trigger matches

2. **Direct Enrollment**:
   - For each matching lead, enrolls them in the sequence
   - Creates ALL messages upfront in `broadcast_messages` table
   - Sets `scheduled_at` for each message (5 min, then +24h for each step)
   - Updates lead trigger to NULL after enrollment

3. **Message Processing** (every 5 seconds):
   - `OptimizedBroadcastProcessor` runs
   - Gets devices with pending messages
   - For each device:
     - Checks if online
     - Gets pending messages with `GetPendingMessagesAndLock()`
     - Updates status to 'processing' with worker_id
     - Sends to broadcast manager
     - Updates status to 'queued'
   - Broadcast manager sends via WhatsApp
   - Updates status to 'sent' or 'failed'

## CAMPAIGN FLOW (Current):

1. **Campaign Processing** (every 1 minute):
   - `ProcessCampaigns()` checks campaigns table
   - Query: campaigns WHERE status='pending' AND scheduled_at <= NOW()
   
2. **Campaign Execution**:
   - Gets leads matching niche AND target_status
   - For each lead, creates ONE message in broadcast_messages
   - Sets campaign_id (not sequence_id)
   - Updates campaign status to 'triggered'

3. **Same Message Processing**:
   - Uses same OptimizedBroadcastProcessor
   - Messages are processed exactly like sequence messages

## KEY DIFFERENCES:

1. **Sequences**: 
   - Multiple messages per lead (entire journey)
   - Messages scheduled over days/weeks
   - Tracks progress per contact
   
2. **Campaigns**: 
   - One message per lead
   - All sent immediately (or at scheduled time)
   - Simple broadcast to matching leads

## ISSUES TO FIX:

1. Campaign query is checking for sequence_steps (WRONG)
2. Campaign scheduled_at might have timezone issues
3. Need to ensure campaign messages have proper delays