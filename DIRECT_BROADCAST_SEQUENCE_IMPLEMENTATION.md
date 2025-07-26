# Direct Broadcast Sequence Implementation

## Summary of Changes (January 2025)

### Overview
Modified the sequence processing system to skip `sequence_contacts` table entirely and create messages directly in `broadcast_messages` table with scheduled timing.

### Key Changes:

1. **Enrollment Query Updated**
   - Changed duplicate check from `sequence_contacts` to `broadcast_messages`
   - Checks for pending messages: `WHERE bm.sequence_id = s.id AND bm.recipient_phone = l.phone AND bm.status = 'pending'`

2. **New Direct Enrollment Function**
   - `enrollContactInSequenceDirectBroadcast()` replaces the old enrollment logic
   - Creates ALL messages upfront when a lead enrolls
   - Calculates `scheduled_at` for each message:
     - First message: NOW() + 5 minutes
     - Subsequent messages: previous scheduled_at + trigger_delay_hours
   - Follows sequence links automatically (e.g., COLD → WARM → HOT)

3. **Benefits**
   - Simpler architecture - no intermediate `sequence_contacts` table
   - Better visibility - can see all scheduled messages upfront
   - Unified processing - sequences and campaigns both use broadcast_messages
   - Easier cancellation - just update status to cancel future messages

### Technical Details:

#### Message Creation Flow:
```
1. Lead has trigger "COLDEXAMA"
2. System finds COLD sequence with entry point "COLDEXAMA"
3. Creates 5 messages for COLD steps with calculated scheduled_at times
4. Finds COLD step 5 has next_trigger = "WARMEXAMA"
5. Looks for sequence with trigger = "WARMEXAMA" (finds WARM sequence)
6. Creates 4 messages for WARM steps
7. Finds WARM step 4 has next_trigger = "HOTEXAMA"
8. Creates 2 messages for HOT steps
9. Total: 11 messages created in one transaction
```

#### Timing Example:
```
Enrollment at 10:00 AM:
- COLD Step 1: 10:05 AM (NOW + 5 min)
- COLD Step 2: 11:05 AM (Step 1 + 1 hour)
- COLD Step 3: 12:05 PM (Step 2 + 1 hour)
- COLD Step 4: 1:05 PM (Step 3 + 1 hour)
- COLD Step 5: 3:05 PM (Step 4 + 2 hours)
- WARM Step 1: 3:05 PM (immediately after COLD)
- WARM Step 2: 4:05 PM (Step 1 + 1 hour)
- WARM Step 3: 5:05 PM (Step 2 + 1 hour)
- WARM Step 4: 8:05 PM (Step 3 + 3 hours)
- HOT Step 1: 8:05 PM (immediately after WARM)
- HOT Step 2: 9:05 PM (Step 1 + 1 hour)
```

### Database Changes:
- No schema changes required
- `broadcast_messages` already has all necessary columns:
  - `scheduled_at` - when to send the message
  - `sequence_id` - which sequence
  - `sequence_stepid` - which step
  - `status` - pending/sent/failed

### Processing Changes:
- Sequence processor now only handles enrollment (every 15 seconds)
- No longer processes `sequence_contacts` 
- Unified broadcast processor handles all messages based on `scheduled_at`

### Files Modified:
1. `src/usecase/sequence_trigger_processor.go` - Complete rewrite for direct broadcast
2. Removed old backup files to prevent compilation conflicts

### How to Use:
1. Create sequences with steps as before
2. Set `trigger_delay_hours` on each step for timing
3. Link sequences using `next_trigger` field in steps
4. Add triggers to leads
5. System automatically enrolls and schedules all messages

### Rollback Instructions:
If needed, the old implementation is backed up as:
- `sequence_trigger_processor_old.go`

To rollback:
1. Delete current `sequence_trigger_processor.go`
2. Rename `sequence_trigger_processor_old.go` to `sequence_trigger_processor.go`
3. Rebuild the application

### Testing:
1. Create a test lead with trigger
2. Check `broadcast_messages` table for scheduled messages
3. Verify `scheduled_at` times are correct
4. Monitor logs for enrollment confirmation

---
**Implementation Date**: January 2025
**Developer Notes**: This implementation simplifies the sequence system significantly while maintaining all functionality.
