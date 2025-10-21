-- Test campaign query
SELECT 
    c.id, 
    c.title, 
    c.status,
    c.campaign_date,
    c.time_schedule,
    c.scheduled_at,
    STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') as computed_time,
    NOW() as current_time,
    CONVERT_TZ(NOW(), @@session.time_zone, 'Asia/Kuala_Lumpur') as malaysia_time
FROM campaigns c
WHERE c.status = 'pending'
ORDER BY c.id DESC
LIMIT 10;

-- Check timezone
SELECT @@session.time_zone, @@global.time_zone;

-- Check if any campaigns should be triggered
SELECT COUNT(*) as pending_campaigns
FROM campaigns c
WHERE c.status = 'pending'
AND (
    (c.scheduled_at IS NOT NULL AND c.scheduled_at <= CURRENT_TIMESTAMP)
    OR
    (c.scheduled_at IS NULL AND 
     STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
);