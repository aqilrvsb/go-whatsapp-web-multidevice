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

print("=== CHECKING MESSAGE AND GREETING ISSUES ===")

# 1. Check the broadcast message that was sent
print("\n1. CHECKING RECENT MESSAGES:")
cursor.execute("""
    SELECT id, recipient_phone, message_type, content, status, 
           created_at, sent_at, device_id
    FROM broadcast_messages 
    WHERE recipient_phone = '60108924904'
    ORDER BY created_at DESC
    LIMIT 5
""")
messages = cursor.fetchall()
for msg in messages:
    print(f"\n  Message ID: {msg['id']}")
    print(f"  Type: {msg['message_type']}")
    print(f"  Status: {msg['status']}")
    print(f"  Content: '{msg['content']}'")
    print(f"  Created: {msg['created_at']}")
    print(f"  Sent: {msg['sent_at']}")

# 2. Check the sequence step content
print("\n2. CHECKING SEQUENCE STEP CONTENT:")
cursor.execute("""
    SELECT ss.*, s.name as sequence_name
    FROM sequence_steps ss
    JOIN sequences s ON ss.sequence_id = s.id
    WHERE s.name = 'meow'
""")
steps = cursor.fetchall()
for step in steps:
    print(f"\n  Sequence: {step['sequence_name']}")
    print(f"  Step {step['day_number']}:")
    print(f"  Message Type: {step['message_type']}")
    print(f"  Content: '{step['content']}'")
    print(f"  Media URL: {step['media_url']}")

# 3. Check device settings
print("\n3. CHECKING DEVICE PLATFORM:")
cursor.execute("""
    SELECT ud.*, u.email
    FROM user_devices ud
    JOIN users u ON ud.user_id = u.id
    WHERE ud.phone = '60146674397'
    OR ud.id IN (
        SELECT device_id FROM broadcast_messages 
        WHERE recipient_phone = '60108924904'
        LIMIT 1
    )
""")
devices = cursor.fetchall()
for device in devices:
    print(f"\n  Device: {device['device_name']}")
    print(f"  Platform: {device['platform']}")
    print(f"  Status: {device['status']}")

conn.close()

print("\n=== ANALYSIS ===")
print("\nThe issue is that the line break between greeting and content is not showing.")
print("This could be because:")
print("1. WhatsApp image captions might handle line breaks differently")
print("2. The platform (Wablas/Whacenter) might be stripping line breaks")
print("3. The greeting processor might need adjustment for image captions")
