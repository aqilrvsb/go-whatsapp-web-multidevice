import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== INVESTIGATING WHY CAMPAIGN MESSAGE IS STUCK ===\n")

# Check the message and device
cursor.execute("""
    SELECT 
        bm.id,
        bm.campaign_id,
        bm.device_id,
        bm.status,
        bm.scheduled_at,
        bm.created_at,
        ud.status as device_status,
        ud.platform,
        NOW() as server_now,
        CASE 
            WHEN bm.scheduled_at <= NOW() THEN 'SHOULD BE SENT'
            ELSE 'NOT YET'
        END as readiness
    FROM broadcast_messages bm
    LEFT JOIN user_devices ud ON bm.device_id = ud.id
    WHERE bm.id = '61774152-90da-4e9d-a2bc-e71a3ef47c20'
""")

msg = cursor.fetchone()
if msg:
    print(f"Message ID: {msg['id']}")
    print(f"Campaign ID: {msg['campaign_id']}")
    print(f"Status: {msg['status']}")
    print(f"Scheduled: {msg['scheduled_at']}")
    print(f"Server Now: {msg['server_now']}")
    print(f"Readiness: {msg['readiness']}")
    print(f"\nDevice ID: {msg['device_id']}")
    print(f"Device Status: {msg['device_status']}")
    print(f"Device Platform: {msg['platform']}")

# Check if there are ANY campaign messages being processed
print("\n=== CHECKING CAMPAIGN MESSAGE PROCESSING ===")
cursor.execute("""
    SELECT 
        COUNT(*) as total,
        SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
        SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END) as processing,
        SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent
    FROM broadcast_messages
    WHERE campaign_id IS NOT NULL
    AND DATE(scheduled_at) = CURDATE()
""")
stats = cursor.fetchone()
print(f"\nToday's campaign messages:")
print(f"  Total: {stats['total']}")
print(f"  Pending: {stats['pending']}")
print(f"  Processing: {stats['processing']}")
print(f"  Sent: {stats['sent']}")

# Check if broadcast processor is running
print("\n=== POSSIBLE ISSUES ===")
print("1. Device offline - messages won't send if device status is 'offline'")
print("2. Broadcast processor not running or crashed")
print("3. GetPendingMessages query issue with timezone")

# Let's check the exact query that would pick this up
print("\n=== TESTING GetPendingMessages QUERY ===")
cursor.execute("""
    SELECT COUNT(*) as should_be_processed
    FROM broadcast_messages bm
    WHERE bm.device_id = '36c25d7b-3db5-4ffa-b767-aaacd654bc4e'
    AND bm.status = 'pending'
    AND bm.scheduled_at IS NOT NULL
    AND bm.scheduled_at <= NOW()
""")
result = cursor.fetchone()
print(f"Messages ready for this device: {result['should_be_processed']}")

conn.close()
