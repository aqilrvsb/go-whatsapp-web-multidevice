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

# Check the pending campaign
cursor.execute("""
    SELECT *
    FROM campaigns
    WHERE status = 'pending'
    LIMIT 1
""")

camp = cursor.fetchone()
if camp:
    print(f"\n=== Campaign: {camp['title']} ===")
    print(f"ID: {camp['id']}")
    print(f"Date: {camp['campaign_date']}")
    print(f"Time: {camp['time_schedule']}")
    print(f"Niche: {camp['niche']}")
    print(f"Target Status: {camp['target_status']}")
    print(f"User ID: {camp['user_id']}")
    
    # Check if time has passed
    scheduled_datetime = f"{camp['campaign_date']} {camp['time_schedule']}"
    print(f"\nScheduled for: {scheduled_datetime}")
    print(f"Current time: {result['db_time']}")
    
    # The issue is clear - campaign is scheduled for 15:36 but current time is 04:31
    # That's 11 hours in the future!
    
    # Test the exact query
    print("\n=== Testing Campaign Query ===")
    cursor.execute("""
        SELECT 
            c.id,
            STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') as scheduled_time,
            NOW() as current_time,
            CASE 
                WHEN STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW() 
                THEN 'YES - Should trigger'
                ELSE 'NO - Future time'
            END as trigger_status
        FROM campaigns c
        WHERE c.id = %s
    """, (camp['id'],))
    
    result = cursor.fetchone()
    print(f"Scheduled Time: {result['scheduled_time']}")
    print(f"Current Time: {result['current_time']}")
    print(f"Will Trigger? {result['trigger_status']}")
    
    # Calculate time difference
    cursor.execute("""
        SELECT 
            TIMESTAMPDIFF(MINUTE, 
                NOW(), 
                STR_TO_DATE(CONCAT(%s, ' ', %s), '%Y-%m-%d %H:%i:%s')
            ) as minutes_until_trigger
    """, (camp['campaign_date'], camp['time_schedule']))
    
    diff = cursor.fetchone()
    if diff['minutes_until_trigger'] and diff['minutes_until_trigger'] > 0:
        hours = diff['minutes_until_trigger'] // 60
        minutes = diff['minutes_until_trigger'] % 60
        print(f"\nCampaign will trigger in: {hours} hours and {minutes} minutes")
    elif diff['minutes_until_trigger'] and diff['minutes_until_trigger'] < 0:
        print(f"\nCampaign should have triggered {abs(diff['minutes_until_trigger'])} minutes ago!")
        print("But it's still pending - let's check why...")
        
        # Check for leads
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM leads l
            INNER JOIN user_devices ud ON l.device_id = ud.id
            WHERE ud.user_id = %s
            AND l.niche LIKE CONCAT('%%', %s, '%%')
        """, (camp['user_id'], camp['niche']))
        
        leads = cursor.fetchone()
        print(f"\nTotal leads with niche '{camp['niche']}': {leads['count']}")
        
        # Check connected devices
        cursor.execute("""
            SELECT id, device_name, status, platform
            FROM user_devices
            WHERE user_id = %s
            AND (status = 'connected' OR status = 'online' OR platform IS NOT NULL)
        """, (camp['user_id'],))
        
        devices = cursor.fetchall()
        print(f"\nConnected devices: {len(devices)}")
        for dev in devices:
            print(f"  - {dev['device_name']} (Status: {dev['status']}, Platform: {dev['platform']})")

conn.close()
