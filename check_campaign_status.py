import mysql.connector
from datetime import datetime

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== CHECKING CAMPAIGN STATUS ===\n")

# Get DB time
cursor.execute("SELECT NOW() as db_time")
result = cursor.fetchone()
print(f"Database Time: {result['db_time']}")

# Check pending campaigns
cursor.execute("""
    SELECT 
        id,
        title,
        campaign_date,
        time_schedule,
        scheduled_at,
        STR_TO_DATE(CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') as parsed_time,
        NOW() as now_time,
        CASE 
            WHEN STR_TO_DATE(CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW() 
            THEN 'YES - Should Trigger!' 
            ELSE 'NO - Future Time' 
        END as trigger_status,
        TIMESTAMPDIFF(MINUTE, NOW(), STR_TO_DATE(CONCAT(campaign_date, ' ', COALESCE(time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s')) as minutes_diff
    FROM campaigns
    WHERE status = 'pending'
    AND campaign_date >= '2025-08-01'
    ORDER BY campaign_date DESC, time_schedule DESC
    LIMIT 5
""")

campaigns = cursor.fetchall()
print(f"\nFound {len(campaigns)} pending campaigns:\n")

for camp in campaigns:
    print(f"Campaign ID {camp['id']}: {camp['title']}")
    print(f"  Date: {camp['campaign_date']}")
    print(f"  Time: {camp['time_schedule']}")
    print(f"  Scheduled At: {camp['scheduled_at']}")
    print(f"  Parsed Time: {camp['parsed_time']}")
    print(f"  Trigger Status: {camp['trigger_status']}")
    
    if camp['minutes_diff']:
        if camp['minutes_diff'] > 0:
            hours = camp['minutes_diff'] // 60
            minutes = camp['minutes_diff'] % 60
            print(f"  Will trigger in: {hours}h {minutes}m")
        else:
            print(f"  Should have triggered {abs(camp['minutes_diff'])} minutes ago!")
    print()

# Test the exact query from the code
print("\n=== TESTING EXACT QUERY FROM CODE ===")
query = """
SELECT c.id, c.user_id, c.title, c.message, c.niche, 
    COALESCE(c.target_status, 'all') AS target_status, 
    COALESCE(c.image_url, '') AS image_url, c.min_delay_seconds, c.max_delay_seconds,
    c.campaign_date, c.time_schedule
FROM campaigns c
WHERE c.status = 'pending'
AND (
    (c.scheduled_at IS NOT NULL AND c.scheduled_at <= NOW())
    OR
    (c.scheduled_at IS NULL AND 
     STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
)
"""

cursor.execute(query)
ready = cursor.fetchall()
print(f"\nCampaigns ready to trigger: {len(ready)}")
for camp in ready:
    print(f"  - {camp['title']} ({camp['campaign_date']} {camp['time_schedule']})")

conn.close()
