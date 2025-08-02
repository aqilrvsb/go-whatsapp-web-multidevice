import pymysql
from datetime import datetime

# Connect to MySQL
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)
cursor = conn.cursor()

print("=== FORCING IMMEDIATE SEND ===\n")

# Update the SCTTN-S03 message to send immediately
cursor.execute("""
    UPDATE broadcast_messages 
    SET scheduled_at = DATE_SUB(NOW(), INTERVAL 5 MINUTE)
    WHERE id = '2b611067-f5e5-4b6f-adcb-56f9cb3e03e8'
    AND status = 'pending'
""")
updated = cursor.rowcount
conn.commit()

print(f"Updated {updated} message(s) to send immediately")

# Also check if there's a worker health issue
print("\n=== CHECKING WORKER HEALTH ===")
cursor.execute("""
    SELECT * FROM worker_health 
    ORDER BY last_heartbeat DESC 
    LIMIT 1
""")
try:
    health = cursor.fetchone()
    if health:
        print(f"Last worker heartbeat: {health[1]}")
except:
    print("No worker_health table found")

# Check recent sent messages to see if worker is active
print("\n=== RECENT SENT MESSAGES ===")
cursor.execute("""
    SELECT recipient_phone, sent_at, device_id
    FROM broadcast_messages 
    WHERE status = 'sent'
    AND sent_at > DATE_SUB(NOW(), INTERVAL 10 MINUTE)
    ORDER BY sent_at DESC
    LIMIT 5
""")
recent = cursor.fetchall()
if recent:
    print(f"Found {len(recent)} messages sent in last 10 minutes")
    for msg in recent:
        print(f"  Sent to {msg[0]} at {msg[1]}")
else:
    print("❌ No messages sent in last 10 minutes - broadcast worker might be down!")

print("\n=== SOLUTION ===")
print("The broadcast worker seems to not be processing messages.")
print("Check Railway logs for 'broadcast worker' activity.")
print("The worker should run every 5 seconds and process pending messages.")

conn.close()
