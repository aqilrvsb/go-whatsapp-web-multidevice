-- FIX SEQUENCE DUPLICATE ISSUE
-- The problem: System is creating multiple sequence_contact records per contact/sequence
-- Instead of updating a single record as the contact progresses through steps

-- 1. First, let's see the extent of the problem
SELECT 
    'BEFORE FIX: Duplicate sequence_contacts' as status,
    COUNT(*) as total_duplicates
FROM (
    SELECT contact_phone, sequence_id, COUNT(*) as cnt
    FROM sequence_contacts
    GROUP BY contact_phone, sequence_id
    HAVING COUNT(*) > 1
) duplicates;

-- 2. Show some examples
SELECT 
    'Example duplicates (first 5):' as info,
    contact_phone,
    sequence_id,
    COUNT(*) as duplicate_count,
    STRING_AGG(current_step::text, ', ' ORDER BY current_step) as steps
FROM sequence_contacts
GROUP BY contact_phone, sequence_id
HAVING COUNT(*) > 1
ORDER BY COUNT(*) DESC
LIMIT 5;

-- 3. Create a backup table before fixing
CREATE TABLE IF NOT EXISTS sequence_contacts_backup_20250123 AS 
SELECT * FROM sequence_contacts;

-- 4. Delete duplicate records, keeping only the one with the highest step number
-- This assumes the contact should be at their furthest progress point
WITH ranked_contacts AS (
    SELECT 
        id,
        contact_phone,
        sequence_id,
        current_step,
        ROW_NUMBER() OVER (
            PARTITION BY contact_phone, sequence_id 
            ORDER BY current_step DESC, created_at DESC
        ) as rn
    FROM sequence_contacts
)
DELETE FROM sequence_contacts
WHERE id IN (
    SELECT id FROM ranked_contacts WHERE rn > 1
);

-- 5. Verify the fix
SELECT 
    'AFTER FIX: Remaining duplicates' as status,
    COUNT(*) as should_be_zero
FROM (
    SELECT contact_phone, sequence_id, COUNT(*) as cnt
    FROM sequence_contacts
    GROUP BY contact_phone, sequence_id
    HAVING COUNT(*) > 1
) duplicates;

-- 6. Show current state of sequence_contacts
SELECT 
    'Current sequence_contacts summary:' as info,
    COUNT(DISTINCT contact_phone) as unique_contacts,
    COUNT(*) as total_records,
    COUNT(DISTINCT sequence_id) as unique_sequences;

-- 7. Reset active sequences if needed (optional - uncomment to use)
-- This will restart sequences from step 1 for testing
/*
UPDATE sequence_contacts
SET 
    current_step = 1,
    status = 'active',
    next_trigger_time = NOW() + INTERVAL '5 minutes'
WHERE sequence_id IN (
    SELECT id FROM sequences WHERE is_active = true
)
AND status = 'completed';
*/

-- 8. Show the distribution of contacts by step
SELECT 
    current_step,
    status,
    COUNT(*) as contact_count
FROM sequence_contacts
GROUP BY current_step, status
ORDER BY current_step, status;