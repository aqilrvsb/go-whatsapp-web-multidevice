import pymysql
import uuid
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

print("=== CREATING MESSAGE WITH ONLINE DEVICE ===\n")

# 1. Find an ONLINE device
cursor.execute("""
    SELECT id, device_name, phone, platform, user_id
    FROM user_devices 
    WHERE status = 'online'
    AND platform IN ('Wablas', 'Whacenter', 'WhatsApp')
    LIMIT 1
""")
online_device = cursor.fetchone()

if not online_device:
    print("❌ No online devices found!")
else:
    print(f"Found online device: {online_device['device_name']} ({online_device['platform']})")
    
    # 2. Get sequence info
    cursor.execute("""
        SELECT s.id as sequence_id, ss.* 
        FROM sequences s
        JOIN sequence_steps ss ON s.id = ss.sequence_id
        WHERE s.name = 'meow'
        LIMIT 1
    """)
    seq_info = cursor.fetchone()
    
    if seq_info:
        # 3. Create new message with online device
        msg_id = str(uuid.uuid4())
        
        print(f"\nCreating message with:")
        print(f"  Device: {online_device['device_name']} (ONLINE)")
        print(f"  Recipient Name: 'Aqil 1' (not Cik)")
        print(f"  Message Type: {seq_info['message_type']}")
        
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
            online_device['user_id'],
            online_device['id'],
            seq_info['sequence_id'],
            '60108924904',
            'Aqil 1',  # Proper name, not phone number
            seq_info['message_type'],
            seq_info['content'],
            seq_info['media_url'],
            datetime.now() + timedelta(seconds=5)  # Send in 5 seconds
        ))
        
        conn.commit()
        
        print(f"\n✅ Message created!")
        print(f"Message ID: {msg_id}")
        print(f"Scheduled to send in 5 seconds")
        print(f"\nExpected greeting format:")
        print(f"  Hello Aqil 1,")
        print(f"  ")
        print(f"  asdsad")
        
        # 4. Mark old message as cancelled
        cursor.execute("""
            UPDATE broadcast_messages 
            SET status = 'cancelled', error_message = 'Replaced with new message'
            WHERE recipient_phone = '60108924904' 
            AND status IN ('pending', 'skipped')
            AND id != %s
        """, (msg_id,))
        conn.commit()

conn.close()

print("\n=== CHECK YOUR WHATSAPP IN 10 SECONDS ===")
