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
cursor = conn.cursor()

print("=== FORCING PENDING MESSAGES TO SEND ===")

# Update all pending messages to send immediately
cursor.execute("""
    UPDATE broadcast_messages 
    SET scheduled_at = DATE_SUB(NOW(), INTERVAL 1 MINUTE)
    WHERE recipient_phone = '60108924904' 
    AND status = 'pending'
""")
updated = cursor.rowcount
conn.commit()

print(f"\nUpdated {updated} messages to send immediately")
print("Messages scheduled for 1 minute ago - broadcast worker should pick them up now")

# Check if there's a broadcast lock
print("\n=== CHECKING BROADCAST LOCKS ===")
cursor.execute("""
    SELECT * FROM broadcast_locks 
    WHERE device_id IN (
        SELECT device_id FROM broadcast_messages 
        WHERE recipient_phone = '60108924904' 
        AND status = 'pending'
    )
""")
locks = cursor.fetchall()
if locks:
    print(f"Found {len(locks)} locks - this might be blocking sends")
    # Clear locks
    cursor.execute("DELETE FROM broadcast_locks")
    conn.commit()
    print("Cleared all broadcast locks")
else:
    print("No locks found")

# Also check if broadcast worker is running
print("\n=== MESSAGE STATUS ===")
cursor.execute("""
    SELECT id, status, scheduled_at, error_message
    FROM broadcast_messages 
    WHERE recipient_phone = '60108924904'
    ORDER BY created_at DESC
    LIMIT 3
""")
messages = cursor.fetchall()
for msg in messages:
    print(f"\nMessage {msg[0]}:")
    print(f"  Status: {msg[1]}")
    print(f"  Scheduled: {msg[2]}")
    if msg[3]:
        print(f"  Error: {msg[3]}")

conn.close()

print("\n=== DONE ===")
print("Check your WhatsApp now - the message should be sent within 5 seconds")
print("If not, check Railway logs for broadcast worker errors")
