-- Check campaigns table structure
DESCRIBE campaigns;

-- Check sequences table structure  
DESCRIBE sequences;

-- Check sequence_steps table structure
DESCRIBE sequence_steps;

-- Check broadcast_messages table structure
DESCRIBE broadcast_messages;

-- Show sample campaign data
SELECT * FROM campaigns LIMIT 5;

-- Show sample sequence data
SELECT * FROM sequences LIMIT 5;

-- Show how sequences work with broadcast_messages
SELECT bm.*, s.name as sequence_name
FROM broadcast_messages bm
JOIN sequences s ON s.id = bm.sequence_id
WHERE bm.sequence_id IS NOT NULL
LIMIT 5;

-- Show how campaigns work with broadcast_messages
SELECT bm.*, c.title as campaign_title
FROM broadcast_messages bm
JOIN campaigns c ON c.id = bm.campaign_id  
WHERE bm.campaign_id IS NOT NULL
LIMIT 5;