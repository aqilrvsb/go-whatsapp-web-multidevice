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

print("=== MONITORING MESSAGE SEND ===")

# Check message status every 2 seconds for 20 seconds
for i in range(10):
    cursor.execute("""
        SELECT id, recipient_name, status, scheduled_at, sent_at, error_message
        FROM broadcast_messages 
        WHERE recipient_phone = '60108924904'
        ORDER BY created_at DESC
        LIMIT 1
    """)
    msg = cursor.fetchone()
    
    print(f"\n[Check {i+1}] Message Status:")
    print(f"  ID: {msg['id']}")
    print(f"  Recipient Name: '{msg['recipient_name']}'")
    print(f"  Status: {msg['status']}")
    print(f"  Scheduled: {msg['scheduled_at']}")
    print(f"  Sent at: {msg['sent_at']}")
    if msg['error_message']:
        print(f"  ERROR: {msg['error_message']}")
    
    if msg['status'] == 'sent':
        print("\n✅ MESSAGE SENT! Check your WhatsApp now!")
        print("It should show the greeting with proper format:")
        print("  Hello Aqil 1,")
        print("  ")
        print("  asdsad")
        break
    elif msg['status'] == 'failed':
        print("\n❌ MESSAGE FAILED!")
        if msg['error_message']:
            print(f"Error: {msg['error_message']}")
        break
    
    if i < 9:
        print("  Waiting 2 seconds...")
        time.sleep(2)

conn.close()

if msg['status'] == 'pending':
    print("\n⚠️ Message still pending after 20 seconds")
    print("Possible issues:")
    print("1. Broadcast worker might not be running")
    print("2. Device might be offline")
    print("3. Check Railway logs for errors")
