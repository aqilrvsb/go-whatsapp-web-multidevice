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

print("=== CHECKING WHY SEQUENCES NOT CREATING BROADCAST_MESSAGES ===\n")

# 1. Check the active sequence details
print("1. ACTIVE SEQUENCE DETAILS:")
cursor.execute("""
    SELECT * FROM sequences 
    WHERE is_active = 1 AND status = 'active'
""")
sequence = cursor.fetchone()
if sequence:
    print(f"  Name: {sequence['name']}")
    print(f"  ID: {sequence['id']}")
    print(f"  Trigger: '{sequence['trigger']}'")
    print(f"  Target Status: {sequence['target_status']}")
    print(f"  Device ID: {sequence['device_id']}")
    print(f"  Min/Max Delay: {sequence['min_delay_seconds']}/{sequence['max_delay_seconds']}s")

# 2. Check sequence steps
print("\n2. SEQUENCE STEPS:")
cursor.execute("""
    SELECT * FROM sequence_steps 
    WHERE sequence_id = %s
    ORDER BY day_number
""", (sequence['id'],))
steps = cursor.fetchall()
for step in steps:
    print(f"  Step {step['day_number']}:")
    print(f"    Trigger: {step['trigger']}")
    print(f"    Content: {step['content'][:50]}...")
    print(f"    Entry Point: {step['is_entry_point']}")
    print(f"    Message Type: {step['message_type']}")

# 3. Check leads that should match
print("\n3. LEADS THAT SHOULD TRIGGER:")
# Check for leads with matching trigger
cursor.execute("""
    SELECT l.*, ud.status as device_status
    FROM leads l
    LEFT JOIN user_devices ud ON l.device_id = ud.id
    WHERE l.`trigger` LIKE %s
    AND l.target_status = %s
""", (f'%{sequence["trigger"]}%', sequence['target_status']))
matching_leads = cursor.fetchall()
print(f"  Found {len(matching_leads)} leads with trigger '{sequence['trigger']}' and status '{sequence['target_status']}'")
for lead in matching_leads:
    print(f"    - {lead['name']} ({lead['phone']})")
    print(f"      Device: {lead['device_id']} - Status: {lead['device_status']}")
    print(f"      Triggers: {lead['trigger']}")

# 4. Check if this lead already has messages in broadcast_messages
print("\n4. CHECKING EXISTING MESSAGES FOR THESE LEADS:")
if matching_leads:
    phones = [lead['phone'] for lead in matching_leads]
    placeholders = ','.join(['%s'] * len(phones))
    cursor.execute(f"""
        SELECT recipient_phone, status, created_at, sequence_id
        FROM broadcast_messages
        WHERE recipient_phone IN ({placeholders})
        AND created_at > DATE_SUB(NOW(), INTERVAL 7 DAY)
        ORDER BY created_at DESC
    """, phones)
    existing_msgs = cursor.fetchall()
    print(f"  Found {len(existing_msgs)} recent messages for these leads")
    for msg in existing_msgs:
        print(f"    - {msg['recipient_phone']} - {msg['status']} - {msg['created_at']}")

# 5. Test what the trigger query should find
print("\n5. TESTING SEQUENCE TRIGGER LOGIC:")
print("  The sequence trigger should find leads where:")
print(f"    - trigger contains '{sequence['trigger']}'")
print(f"    - target_status = '{sequence['target_status']}'")
print(f"    - device is online")

# 6. Check if there are any broadcast messages from ANY sequence
print("\n6. ANY SEQUENCE MESSAGES IN LAST 24H:")
cursor.execute("""
    SELECT s.name, COUNT(*) as msg_count
    FROM broadcast_messages bm
    JOIN sequences s ON bm.sequence_id = s.id
    WHERE bm.created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)
    GROUP BY s.name
""")
seq_msgs = cursor.fetchall()
if seq_msgs:
    for sm in seq_msgs:
        print(f"  - {sm['name']}: {sm['msg_count']} messages")
else:
    print("  ❌ NO sequence messages in last 24 hours!")

# 7. Manual insert test
print("\n7. MANUAL BROADCAST MESSAGE INSERT (TEST):")
if matching_leads:
    lead = matching_leads[0]
    step = steps[0] if steps else None
    
    if step:
        print(f"  Would insert message for: {lead['name']} ({lead['phone']})")
        print(f"  Content: {step['content']}")
        print(f"  Device: {lead['device_id']}")
        print("\n  SQL to manually test:")
        print(f"""
INSERT INTO broadcast_messages (
    id, user_id, device_id, sequence_id, 
    recipient_phone, recipient_name, message_type, 
    content, status, created_at
) VALUES (
    UUID(), 
    '{sequence['user_id']}',
    '{lead['device_id']}',
    '{sequence['id']}',
    '{lead['phone']}',
    '{lead['name']}',
    '{step['message_type']}',
    '{step['content'].replace("'", "''")}',
    'pending',
    NOW()
);
""")

print("\n=== POSSIBLE ISSUES ===")
print("1. Sequence trigger service not running")
print("2. No matching leads (check trigger and target_status)")
print("3. Device offline (sequences only send to online devices)")
print("4. Error in sequence trigger processor")

conn.close()
