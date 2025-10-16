import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== CHECKING HOW MESSAGES ARE PROCESSED ===\n")

# Check pending messages
cursor.execute("""
    SELECT 
        bm.id,
        bm.campaign_id,
        bm.sequence_id,
        bm.status,
        bm.scheduled_at,
        bm.device_id,
        ud.status as device_status,
        NOW() as current_time,
        CASE 
            WHEN bm.scheduled_at <= NOW() THEN 'READY'
            ELSE CONCAT('WAIT ', TIMESTAMPDIFF(MINUTE, NOW(), bm.scheduled_at), ' min')
        END as readiness
    FROM broadcast_messages bm
    LEFT JOIN user_devices ud ON bm.device_id = ud.id
    WHERE bm.status = 'pending'
    ORDER BY bm.scheduled_at
    LIMIT 10
""")

messages = cursor.fetchall()
print(f"Found {len(messages)} pending messages\n")

for msg in messages:
    msg_type = "Campaign" if msg['campaign_id'] else "Sequence"
    print(f"{msg_type} Message:")
    print(f"  ID: {msg['id']}")
    print(f"  Scheduled: {msg['scheduled_at']}")
    print(f"  Current Time: {msg['current_time']}")
    print(f"  Status: {msg['readiness']}")
    print(f"  Device Status: {msg['device_status']}")
    print()

print("\n=== KEY FINDING ===")
print("Both campaigns and sequences use the EXACT SAME processing:")
print("1. GetPendingMessages checks: status='pending' AND scheduled_at <= NOW()")
print("2. Same broadcast processor handles both")
print("3. Same pool manager queues messages")
print("4. Only difference is campaign_id vs sequence_id field")

conn.close()
