import mysql.connector
from datetime import datetime

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== FIXING EXISTING CAMPAIGN MESSAGES TIMEZONE ===\n")

# Update pending campaign messages to subtract 8 hours
cursor.execute("""
    UPDATE broadcast_messages 
    SET scheduled_at = DATE_SUB(scheduled_at, INTERVAL 8 HOUR)
    WHERE campaign_id IS NOT NULL 
    AND status = 'pending'
    AND scheduled_at > NOW()
""")
affected = cursor.rowcount
conn.commit()

print(f"Updated {affected} campaign messages to UTC timezone")

# Check current campaign messages
cursor.execute("""
    SELECT 
        id,
        recipient_phone,
        scheduled_at,
        NOW() as server_time,
        CASE 
            WHEN scheduled_at <= NOW() THEN 'Ready to send'
            ELSE CONCAT('Wait ', TIMESTAMPDIFF(MINUTE, NOW(), scheduled_at), ' minutes')
        END as status
    FROM broadcast_messages 
    WHERE campaign_id IS NOT NULL 
    AND status = 'pending'
    ORDER BY scheduled_at
    LIMIT 5
""")

messages = cursor.fetchall()
if messages:
    print("\nCampaign messages status:")
    for msg in messages:
        print(f"\nPhone: {msg[1]}")
        print(f"  Scheduled: {msg[2]}")
        print(f"  Status: {msg[4]}")

conn.close()
