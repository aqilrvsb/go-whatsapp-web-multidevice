-- Check actual PostgreSQL table structure for broadcast_messages
-- This will show all columns including sequence_stepid if it exists

\d broadcast_messages

-- Alternative query to check columns
SELECT 
    column_name, 
    data_type, 
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_name = 'broadcast_messages'
ORDER BY ordinal_position;

-- Check if sequence_stepid column exists
SELECT EXISTS (
    SELECT 1 
    FROM information_schema.columns 
    WHERE table_name = 'broadcast_messages' 
    AND column_name = 'sequence_stepid'
) as has_sequence_stepid;

-- Check sequence_steps table structure for delay columns
SELECT 
    column_name, 
    data_type
FROM information_schema.columns 
WHERE table_name = 'sequence_steps'
AND column_name IN ('min_delay_seconds', 'max_delay_seconds')
ORDER BY ordinal_position;
