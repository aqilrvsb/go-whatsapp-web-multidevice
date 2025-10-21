import mysql.connector
from datetime import datetime

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)
print("Connected to MySQL successfully!")

# Check current time
cursor.execute("SELECT NOW() as db_time")
result = cursor.fetchone()
print(f"\nCurrent Database Time: {result['db_time']}")

# Check all pending campaigns
print("\n=== CHECKING PENDING CAMPAIGNS ===")
cursor.execute("""
    SELECT 
        id,
        title,
        campaign_date,
        time_schedule,
        scheduled_at,
        status,
        niche,
        target_status,
        user_id
    FROM campaigns
    WHERE status = 'pending'
    ORDER BY campaign_date DESC, time_schedule DESC
    LIMIT 5
""")

campaigns = cursor.fetchall()
print(f"\nFound {len(campaigns)} pending campaigns:")

for camp in campaigns:
    print(f"\n=== Campaign ID {camp['id']}: {camp['title']} ===")
    print(f"  Date: {camp['campaign_date']}, Time: {camp['time_schedule']}")
    print(f"  Scheduled At: {camp['scheduled_at']}")
    print(f"  Status: {camp['status']}")
    print(f"  Niche: {camp['niche']}")
    print(f"  Target Status: {camp['target_status']}")
    print(f"  User ID: {camp['user_id']}")
    
    # Check if it should trigger
    if camp['time_schedule']:
        check_query = f"""
            SELECT 
                STR_TO_DATE(CONCAT('{camp['campaign_date']}', ' ', '{camp['time_schedule']}'), '%Y-%m-%d %H:%i:%s') as parsed_time,
                NOW() as current_time,
                CASE 
                    WHEN STR_TO_DATE(CONCAT('{camp['campaign_date']}', ' ', '{camp['time_schedule']}'), '%Y-%m-%d %H:%i:%s') <= NOW() 
                    THEN 'YES' 
                    ELSE 'NO' 
                END as should_trigger
        """
        cursor.execute(check_query)
        result = cursor.fetchone()
        print(f"  Parsed Time: {result['parsed_time']}")
        print(f"  Current Time: {result['current_time']}")
        print(f"  Should Trigger? {result['should_trigger']}")
    
    # Check for matching leads
    cursor.execute("""
        SELECT COUNT(DISTINCT l.id) as lead_count
        FROM leads l
        INNER JOIN user_devices ud ON l.device_id = ud.id
        WHERE ud.user_id = %s
        AND (ud.status = 'connected' OR ud.status = 'online' OR ud.platform IS NOT NULL)
        AND l.niche LIKE CONCAT('%%', %s, '%%')
        AND (%s = 'all' OR l.target_status = %s)
    """, (camp['user_id'], camp['niche'], camp['target_status'], camp['target_status']))
    
    lead_result = cursor.fetchone()
    print(f"  Matching Leads Available: {lead_result['lead_count']}")
    
    # Check connected devices
    cursor.execute("""
        SELECT COUNT(*) as device_count
        FROM user_devices
        WHERE user_id = %s
        AND (status = 'connected' OR status = 'online' OR platform IS NOT NULL)
    """, (camp['user_id'],))
    
    device_result = cursor.fetchone()
    print(f"  Connected Devices: {device_result['device_count']}")

# Test the exact query from ProcessCampaigns
print("\n\n=== TESTING EXACT PROCESSCAMPAIGNS QUERY ===")
cursor.execute("""
    SELECT c.id, c.user_id, c.title, c.message, c.niche, 
        COALESCE(c.target_status, 'all') AS target_status, 
        COALESCE(c.image_url, '') AS image_url, 
        c.min_delay_seconds, c.max_delay_seconds
    FROM campaigns c
    WHERE c.status = 'pending'
    AND (
        (c.scheduled_at IS NOT NULL AND c.scheduled_at <= NOW())
        OR
        (c.scheduled_at IS NULL AND 
         STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
    )
    LIMIT 10
""")

ready_campaigns = cursor.fetchall()
print(f"\nCampaigns that should be processed: {len(ready_campaigns)}")
for camp in ready_campaigns:
    print(f"  - {camp['title']} (ID: {camp['id']}, Niche: {camp['niche']})")

# Check recent broadcast messages
print("\n\n=== CHECKING RECENT BROADCAST MESSAGES ===")
cursor.execute("""
    SELECT 
        bm.id,
        bm.campaign_id,
        bm.sequence_id,
        bm.status,
        bm.created_at,
        c.title as campaign_title
    FROM broadcast_messages bm
    LEFT JOIN campaigns c ON bm.campaign_id = c.id
    WHERE bm.created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
    ORDER BY bm.created_at DESC
    LIMIT 10
""")

recent_messages = cursor.fetchall()
print(f"\nRecent broadcast messages (last hour): {len(recent_messages)}")
for msg in recent_messages:
    if msg['campaign_id']:
        print(f"  - Campaign: {msg['campaign_title']} (ID: {msg['campaign_id']}) - Status: {msg['status']} - Created: {msg['created_at']}")
    else:
        print(f"  - Sequence ID: {msg['sequence_id']} - Status: {msg['status']} - Created: {msg['created_at']}")

conn.close()
print("\n\nDone checking!")
