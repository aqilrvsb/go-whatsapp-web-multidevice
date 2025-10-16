# WhatsApp Sequence Flow Documentation
Date: September 24, 2025

## ✅ CONFIRMED FLOW - How Sequences Work

### 1. TRIGGER SETUP
- Each lead has a `trigger` field in the database (e.g., EXSTART, COLDVITAC, HOTVITAC)
- Triggers are set when leads are imported or updated
- Multiple leads can have the same trigger

### 2. SEQUENCE TEMPLATE CREATION
- Admin creates sequence templates with:
  - Name (e.g., "FOLLOWUP EXSTART")
  - Niche (e.g., EXSTART)
  - Trigger (e.g., EXSTART)
  - Multiple day steps (Day 1, Day 2, Day 3, etc.)
  - Each step has: message content, delay settings, media (optional)

### 3. SEQUENCE ACTIVATION
When a sequence is set to "active":

#### A. ENROLLMENT PROCESS
1. **ProcessDirectEnrollments()** runs every 5 minutes
2. Finds leads where:
   - Lead's trigger matches sequence's entry trigger
   - Lead has valid device_id and user_id
   - Not already enrolled in this sequence
3. Creates ALL messages for the entire sequence journey immediately
4. Schedules messages based on day intervals

#### B. MESSAGE CREATION
For each enrolled lead, the system:
1. Creates broadcast messages for ALL days in the sequence
2. Sets scheduled times (Day 1: +5 minutes, Day 2: +24 hours, etc.)
3. Can chain sequences (COLD → WARM → HOT) if configured
4. All messages are created upfront, not day-by-day

#### C. TRIGGER REMOVAL ✅
**CONFIRMED: After successful enrollment:**
```go
// From direct_broadcast_processor.go line 394
func (p *DirectBroadcastProcessor) removeCompletedTrigger(phone, trigger string) {
    _, err := p.db.Exec("UPDATE leads SET `trigger` = NULL WHERE phone = ?", phone)
    if err != nil {
        logrus.Errorf("Failed to remove trigger for %s: %v", phone, err)
    }
}
```
- The trigger is REMOVED (set to NULL) from the lead
- This prevents duplicate enrollment
- Lead won't be enrolled again even if sequence is deactivated/reactivated

### 4. MESSAGE PROCESSING
- Messages are sent based on their scheduled time
- Broadcast processor picks up pending messages
- Sends via WhatsApp API
- Updates status to 'sent'

## 📋 FLOW SUMMARY

```
1. Lead has trigger "EXSTART"
   ↓
2. Admin creates & activates "FOLLOWUP EXSTART" sequence
   ↓
3. System finds leads with matching trigger
   ↓
4. Creates ALL messages for entire journey
   ↓
5. REMOVES trigger from lead (set to NULL) ✅
   ↓
6. Messages sent according to schedule
```

## ⚠️ IMPORTANT NOTES

1. **Triggers are ONE-TIME USE**
   - Once a lead is enrolled, their trigger is removed
   - They won't be enrolled again unless trigger is re-added

2. **All Messages Created Upfront**
   - Not created day-by-day
   - All scheduled at enrollment time
   - More efficient than daily processing

3. **Sequence Chaining**
   - Can link sequences (COLD → WARM → HOT)
   - Uses next_trigger field to chain sequences
   - Creates messages for entire chain at once

## 🔧 SUGGESTED "FLOW UPDATE" BUTTON

Based on your UI request, a "Flow Update" button could:

### Option 1: Re-trigger Leads
```sql
-- Reset triggers for leads that completed a sequence
UPDATE leads 
SET trigger = 'EXSTART' 
WHERE phone IN (
    SELECT DISTINCT recipient_phone 
    FROM broadcast_messages 
    WHERE sequence_id = ? 
    AND status = 'sent'
)
```

### Option 2: Move to Next Sequence
```sql
-- Update trigger to next stage
UPDATE leads 
SET trigger = 'WARMEXSTART' 
WHERE phone IN (
    SELECT DISTINCT recipient_phone 
    FROM broadcast_messages 
    WHERE sequence_id = ? 
    AND status = 'sent'
    GROUP BY recipient_phone
    HAVING COUNT(*) >= ? -- minimum messages sent
)
```

### Option 3: Restart Failed Sequences
```sql
-- Re-add trigger for failed enrollments
UPDATE leads l
SET trigger = 'EXSTART'
WHERE NOT EXISTS (
    SELECT 1 FROM broadcast_messages bm
    WHERE bm.recipient_phone = l.phone
    AND bm.sequence_id = ?
    AND bm.status IN ('pending', 'sent')
)
AND l.niche LIKE '%EXSTART%'
```

## 📊 DATABASE VERIFICATION QUERIES

### Check Leads with Triggers
```sql
SELECT trigger, COUNT(*) as count 
FROM leads 
WHERE trigger IS NOT NULL 
GROUP BY trigger;
```

### Check Sequence Enrollments
```sql
SELECT 
    s.name as sequence_name,
    COUNT(DISTINCT bm.recipient_phone) as enrolled_leads,
    SUM(CASE WHEN bm.status = 'sent' THEN 1 ELSE 0 END) as sent_messages,
    SUM(CASE WHEN bm.status = 'pending' THEN 1 ELSE 0 END) as pending_messages
FROM sequences s
LEFT JOIN broadcast_messages bm ON bm.sequence_id = s.id
GROUP BY s.id, s.name;
```

### Find Leads Without Triggers
```sql
SELECT COUNT(*) as leads_without_trigger
FROM leads 
WHERE trigger IS NULL OR trigger = '';
```

## 🚀 IMPLEMENTATION FOR "FLOW UPDATE" BUTTON

The button should allow admins to:
1. **View**: Current enrollment status
2. **Reset**: Re-add triggers to completed leads
3. **Progress**: Move leads to next sequence stage
4. **Retry**: Re-enroll failed leads

This would give more control over the sequence flow without waiting for manual trigger updates.