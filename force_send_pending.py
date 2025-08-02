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
cursor = conn.cursor(pymysql.cursors.DictCursor)

print("=== CHECKING PENDING MESSAGES ===\n")

# 1. Check pending messages
cursor.execute("""
    SELECT id, recipient_phone, recipient_name, status, 
           scheduled_at, created_at, device_id
    FROM broadcast_messages 
    WHERE recipient_phone = '60108924904' 
    AND status = 'pending'
    ORDER BY created_at DESC
""")
pending = cursor.fetchall()

print(f"Found {len(pending)} pending messages")
for msg in pending:
    print(f"\nMessage ID: {msg['id']}")
    print(f"  Recipient Name: '{msg['recipient_name']}'")
    print(f"  Scheduled at: {msg['scheduled_at']}")
    print(f"  Created at: {msg['created_at']}")
    print(f"  Current time: {datetime.now()}")
    
    # Check if it's time to send
    if msg['scheduled_at'] <= datetime.now():
        print(f"  ⚠️ This message should have been sent already!")

# 2. Update scheduled time to NOW for immediate sending
if pending:
    print("\n=== FORCING IMMEDIATE SEND ===")
    for msg in pending:
        cursor.execute("""
            UPDATE broadcast_messages 
            SET scheduled_at = NOW()
            WHERE id = %s
        """, (msg['id'],))
        print(f"Updated message {msg['id']} to send immediately")
    
    conn.commit()
    print("\n✅ All pending messages updated to send NOW")
    print("The broadcast worker (runs every 5 seconds) should pick them up immediately")

# 3. Check device status
print("\n=== CHECKING DEVICE STATUS ===")
if pending:
    device_id = pending[0]['device_id']
    cursor.execute("""
        SELECT id, device_name, status, platform, phone
        FROM user_devices 
        WHERE id = %s
    """, (device_id,))
    device = cursor.fetchone()
    if device:
        print(f"Device: {device['device_name']}")
        print(f"Status: {device['status']}")
        print(f"Platform: {device['platform']}")
        print(f"Phone: {device['phone']}")

conn.close()
