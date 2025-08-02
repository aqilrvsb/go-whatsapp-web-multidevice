import pymysql

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

print("=== CHECKING DEVICE SCHQ-S09 ===\n")

# Check the device status
cursor.execute("""
    SELECT id, device_name, status, platform, phone, jid
    FROM user_devices 
    WHERE device_name = 'SCHQ-S09'
""")
device = cursor.fetchone()

if device:
    print(f"Device Name: {device['device_name']}")
    print(f"Device ID: {device['id']}")
    print(f"Status: {device['status']}")
    print(f"Platform: {device['platform']}")
    print(f"Phone: {device['phone']}")
    print(f"JID/Token: {device['jid'][:20]}..." if device['jid'] else "JID/Token: None")
    
    # Update device to online
    if device['status'] != 'online':
        print(f"\n⚠️ Device is {device['status']}, updating to online...")
        cursor.execute("""
            UPDATE user_devices 
            SET status = 'online'
            WHERE id = %s
        """, (device['id'],))
        conn.commit()
        print("✅ Device updated to online")
    
    # Check messages using this device
    print("\n=== MESSAGES USING THIS DEVICE ===")
    cursor.execute("""
        SELECT id, recipient_phone, recipient_name, status, error_message, created_at
        FROM broadcast_messages 
        WHERE device_id = %s
        ORDER BY created_at DESC
        LIMIT 5
    """)
    messages = cursor.fetchall()
    
    for msg in messages:
        print(f"\nMessage to {msg['recipient_phone']} ({msg['recipient_name']})")
        print(f"  Status: {msg['status']}")
        if msg['error_message']:
            print(f"  Error: {msg['error_message']}")
        print(f"  Created: {msg['created_at']}")

# Now update the pending message to retry
print("\n=== RETRYING PENDING MESSAGE ===")
cursor.execute("""
    UPDATE broadcast_messages 
    SET status = 'pending',
        scheduled_at = DATE_ADD(NOW(), INTERVAL 10 SECOND),
        error_message = NULL
    WHERE recipient_phone = '60108924904'
    AND status = 'skipped'
    ORDER BY created_at DESC
    LIMIT 1
""")
updated = cursor.rowcount
conn.commit()

if updated > 0:
    print(f"✅ Updated message to retry in 10 seconds")
    print("The broadcast worker should pick it up now that device is online")
else:
    print("No skipped messages found to retry")

conn.close()
