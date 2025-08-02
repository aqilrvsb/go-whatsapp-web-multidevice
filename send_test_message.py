import pymysql
from datetime import datetime, timedelta

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

print("=== PREPARING TO SEND TEST MESSAGE ===\n")

# 1. Check the lead info
print("1. CHECKING LEAD INFO:")
cursor.execute("""
    SELECT * FROM leads WHERE phone = '60108924904'
""")
lead = cursor.fetchone()
if lead:
    print(f"  Name: {lead['name']}")
    print(f"  Phone: {lead['phone']}")
    print(f"  Trigger: {lead['trigger']}")
    print(f"  Device ID: {lead['device_id']}")

# 2. Check last message sent
print("\n2. LAST MESSAGE SENT:")
cursor.execute("""
    SELECT * FROM broadcast_messages 
    WHERE recipient_phone = '60108924904'
    ORDER BY created_at DESC
    LIMIT 1
""")
last_msg = cursor.fetchone()
if last_msg:
    print(f"  Recipient Name: '{last_msg['recipient_name']}'")
    print(f"  Content: {last_msg['content']}")
    print(f"  Status: {last_msg['status']}")

# 3. Get sequence info
print("\n3. CREATING NEW TEST MESSAGE:")
cursor.execute("""
    SELECT s.*, ss.* 
    FROM sequences s
    JOIN sequence_steps ss ON s.id = ss.sequence_id
    WHERE s.name = 'meow'
    LIMIT 1
""")
seq_info = cursor.fetchone()

if seq_info and lead:
    # Generate UUID
    import uuid
    msg_id = str(uuid.uuid4())
    
    # Create a new broadcast message
    print("\nInserting new test message...")
    cursor.execute("""
        INSERT INTO broadcast_messages (
            id, user_id, device_id, sequence_id,
            recipient_phone, recipient_name, message_type,
            content, media_url, status, created_at, scheduled_at
        ) VALUES (
            %s, %s, %s, %s, %s, %s, %s, %s, %s, 'pending', NOW(), %s
        )
    """, (
        msg_id,
        lead['user_id'],
        lead['device_id'],
        seq_info['sequence_id'],
        '60108924904',
        'Aqil 1',  # Using actual name instead of phone number
        seq_info['message_type'],
        seq_info['content'],
        seq_info['media_url'],
        datetime.now() + timedelta(seconds=30)  # Schedule for 30 seconds from now
    ))
    
    conn.commit()
    print(f"✅ Message created with ID: {msg_id}")
    print(f"   Recipient Name: 'Aqil 1' (not phone number)")
    print(f"   Status: pending")
    print(f"   Scheduled for: 30 seconds from now")
    print("\nThe broadcast worker should pick this up within 5 seconds")
    print("and send it with the greeting: 'Hello Aqil 1,' or similar")
else:
    print("❌ Could not create message - missing data")

conn.close()
