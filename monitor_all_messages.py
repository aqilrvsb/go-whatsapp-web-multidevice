import pymysql
import time

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

print("=== MONITORING MESSAGE SEND WITH SCHQ-S09 ===\n")

# The device SCHQ-S09 is ONLINE, so let's monitor the latest messages
for i in range(10):
    cursor.execute("""
        SELECT bm.id, bm.recipient_name, bm.status, bm.error_message, 
               bm.sent_at, bm.device_id, ud.device_name, ud.status as device_status
        FROM broadcast_messages bm
        JOIN user_devices ud ON bm.device_id = ud.id
        WHERE bm.recipient_phone = '60108924904'
        ORDER BY bm.created_at DESC
        LIMIT 2
    """)
    messages = cursor.fetchall()
    
    print(f"[Check {i+1}] Latest messages:")
    for msg in messages:
        print(f"\n  Message ID: {msg['id'][:8]}...")
        print(f"  Device: {msg['device_name']} (Status: {msg['device_status']})")
        print(f"  Recipient Name: '{msg['recipient_name']}'")
        print(f"  Message Status: {msg['status']}")
        if msg['error_message']:
            print(f"  Error: {msg['error_message']}")
        if msg['sent_at']:
            print(f"  Sent at: {msg['sent_at']}")
        
        if msg['status'] == 'sent':
            print("\n✅ MESSAGE SENT! Check your WhatsApp!")
            conn.close()
            exit()
    
    if i < 9:
        print("\nWaiting 2 seconds...")
        time.sleep(2)
        print("-" * 50)

conn.close()
print("\n⚠️ No messages sent yet. Check Railway logs for broadcast worker activity.")
