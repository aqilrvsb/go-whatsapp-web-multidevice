-- Delete all records from sequence_contacts and broadcast_messages tables
-- WARNING: This will permanently delete ALL data from these tables!

BEGIN;

-- Delete all sequence contacts
DELETE FROM sequence_contacts;

-- Delete all broadcast messages
DELETE FROM broadcast_messages;

COMMIT;

-- Show counts to confirm deletion
SELECT 'Sequence contacts remaining:' as table_name, COUNT(*) as count FROM sequence_contacts
UNION ALL
SELECT 'Broadcast messages remaining:', COUNT(*) FROM broadcast_messages;