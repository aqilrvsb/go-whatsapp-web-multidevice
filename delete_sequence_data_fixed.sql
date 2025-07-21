-- Delete all sequence-related data
-- WARNING: This will permanently delete all sequence data!
-- Fixed order to respect foreign key constraints

BEGIN;

-- 1. First delete all broadcast messages that reference sequences or sequence steps
DELETE FROM broadcast_messages WHERE sequence_id IS NOT NULL OR sequence_stepid IS NOT NULL;

-- 2. Delete message analytics if they exist
DELETE FROM message_analytics WHERE sequence_id IS NOT NULL;

-- 3. Delete all sequence contacts (enrollments)
DELETE FROM sequence_contacts;

-- 4. Delete all sequence steps
DELETE FROM sequence_steps;

-- 5. Finally delete all sequences
DELETE FROM sequences;

COMMIT;

-- Show counts to confirm deletion
SELECT 'Sequences remaining:' as table_name, COUNT(*) as count FROM sequences
UNION ALL
SELECT 'Sequence steps remaining:', COUNT(*) FROM sequence_steps
UNION ALL
SELECT 'Sequence contacts remaining:', COUNT(*) FROM sequence_contacts
UNION ALL
SELECT 'Sequence broadcast messages remaining:', COUNT(*) FROM broadcast_messages WHERE sequence_id IS NOT NULL OR sequence_stepid IS NOT NULL;