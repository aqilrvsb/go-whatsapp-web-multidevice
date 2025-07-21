-- Fix for sequence delays: Add sequence_stepid to broadcast_messages table
-- This allows broadcast processor to get delays from sequence_steps table

-- 1. Add sequence_stepid column if it doesn't exist
ALTER TABLE broadcast_messages 
ADD COLUMN IF NOT EXISTS sequence_stepid UUID REFERENCES sequence_steps(id) ON DELETE SET NULL;

-- 2. Add recipient_name column if it doesn't exist (for greeting processing)
ALTER TABLE broadcast_messages 
ADD COLUMN IF NOT EXISTS recipient_name VARCHAR(255);

-- 3. Create index for better performance
CREATE INDEX IF NOT EXISTS idx_broadcast_messages_sequence_stepid 
ON broadcast_messages(sequence_stepid) 
WHERE sequence_stepid IS NOT NULL;

-- 4. Update any existing sequence messages to have sequence_stepid
-- This query links existing messages to their sequence steps
UPDATE broadcast_messages bm
SET sequence_stepid = sc.sequence_stepid
FROM sequence_contacts sc
WHERE bm.sequence_id = sc.sequence_id
  AND bm.recipient_phone = sc.contact_phone
  AND bm.sequence_stepid IS NULL
  AND sc.sequence_stepid IS NOT NULL;

-- 5. Verify the fix
SELECT 
    COUNT(*) as total_sequence_messages,
    COUNT(sequence_stepid) as messages_with_stepid,
    COUNT(*) - COUNT(sequence_stepid) as messages_missing_stepid
FROM broadcast_messages 
WHERE sequence_id IS NOT NULL;
