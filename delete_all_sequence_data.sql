-- Delete all sequence-related data
-- WARNING: This will permanently delete all sequence data!

BEGIN;

-- Delete all sequence contacts (enrollments)
DELETE FROM sequence_contacts;

-- Delete all sequence steps
DELETE FROM sequence_steps;

-- Delete all sequences
DELETE FROM sequences;

-- Delete any broadcast messages that came from sequences
DELETE FROM broadcast_messages WHERE sequence_id IS NOT NULL;

-- Delete any message analytics from sequences
DELETE FROM message_analytics WHERE sequence_id IS NOT NULL;

-- Reset any sequences if there are any auto-increment columns
-- (PostgreSQL doesn't need this for UUID columns)

COMMIT;

-- Show counts to confirm deletion
SELECT 'Sequences remaining:' as table_name, COUNT(*) as count FROM sequences
UNION ALL
SELECT 'Sequence steps remaining:', COUNT(*) FROM sequence_steps
UNION ALL
SELECT 'Sequence contacts remaining:', COUNT(*) FROM sequence_contacts
UNION ALL
SELECT 'Sequence broadcast messages remaining:', COUNT(*) FROM broadcast_messages WHERE sequence_id IS NOT NULL
UNION ALL
SELECT 'Sequence analytics remaining:', COUNT(*) FROM message_analytics WHERE sequence_id IS NOT NULL;
