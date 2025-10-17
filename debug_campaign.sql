-- 1. Check current time and timezone
SELECT NOW() as server_time, 
       CONVERT_TZ(NOW(), @@session.time_zone, 'Asia/Kuala_Lumpur') as malaysia_time,
       @@session.time_zone as session_timezone;

-- 2. Check pending campaigns
SELECT 
    id,
    title,
    status,
    campaign_date,
    time_schedule,
    scheduled_at,
    CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')) as combined_datetime,
    STR_TO_DATE(CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') as parsed_datetime,
    CASE 
        WHEN status != 'pending' THEN 'Not pending status'
        WHEN scheduled_at IS NOT NULL AND scheduled_at > NOW() THEN 'Scheduled for future (scheduled_at)'
        WHEN STR_TO_DATE(CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') > NOW() THEN 'Scheduled for future (campaign_date)'
        ELSE 'Should trigger NOW'
    END as trigger_status
FROM campaigns
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
ORDER BY id DESC
LIMIT 10;

-- 3. Check if campaigns processor is finding them
SELECT 
    c.id,
    c.title,
    c.niche,
    c.target_status,
    c.user_id,
    COUNT(DISTINCT l.id) as matching_leads_count
FROM campaigns c
LEFT JOIN leads l ON l.niche = c.niche 
    AND l.target_status = COALESCE(c.target_status, 'prospect')
    AND l.user_id = c.user_id
WHERE c.status = 'pending'
    AND (
        (c.scheduled_at IS NOT NULL AND c.scheduled_at <= CURRENT_TIMESTAMP)
        OR
        (c.scheduled_at IS NULL AND 
         STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
    )
GROUP BY c.id;

-- 4. Check devices status for campaigns
SELECT 
    c.id as campaign_id,
    c.title,
    ud.id as device_id,
    ud.device_name,
    ud.status as device_status,
    COUNT(l.id) as leads_on_device
FROM campaigns c
JOIN user_devices ud ON ud.user_id = c.user_id
LEFT JOIN leads l ON l.device_id = ud.id 
    AND l.niche = c.niche 
    AND l.target_status = COALESCE(c.target_status, 'prospect')
WHERE c.status = 'pending'
GROUP BY c.id, ud.id
ORDER BY c.id, ud.device_name;

-- 5. Check broadcast_messages for recent campaigns
SELECT 
    bm.campaign_id,
    c.title,
    COUNT(*) as message_count,
    MIN(bm.created_at) as first_message,
    MAX(bm.created_at) as last_message,
    SUM(CASE WHEN bm.status = 'sent' THEN 1 ELSE 0 END) as sent_count,
    SUM(CASE WHEN bm.status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM broadcast_messages bm
JOIN campaigns c ON c.id = bm.campaign_id
WHERE bm.campaign_id IS NOT NULL
    AND bm.created_at >= DATE_SUB(NOW(), INTERVAL 1 DAY)
GROUP BY bm.campaign_id, c.title;