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

print("=== SEQUENCE TRIGGER DEBUGGING ===")
print(f"Current time: {datetime.now()}")

# 1. Check the lead
print("\n1. LEAD STATUS:")
cursor.execute("""
    SELECT id, phone, `trigger`, device_id, user_id 
    FROM leads WHERE phone = '60108924904'
""")
lead = cursor.fetchone()
if lead:
    print(f"  Phone: {lead['phone']}")
    print(f"  Trigger: '{lead['trigger']}'")
    print(f"  Device ID: {lead['device_id']}")
    print(f"  User ID: {lead['user_id']}")

# 2. Check active sequences with entry points
print("\n2. ACTIVE SEQUENCES WITH MATCHING TRIGGERS:")
cursor.execute("""
    SELECT DISTINCT 
        s.id, s.name, s.`trigger` as seq_trigger, s.is_active,
        ss.`trigger` as step_trigger, ss.is_entry_point
    FROM sequences s
    INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
    WHERE s.is_active = 1 
    AND ss.is_entry_point = 1
""")
sequences = cursor.fetchall()
for seq in sequences:
    print(f"  Sequence: {seq['name']} (ID: {seq['id']})")
    print(f"    Sequence trigger: '{seq['seq_trigger']}'")
    print(f"    Step trigger: '{seq['step_trigger']}'")
    print(f"    Is active: {seq['is_active']}")

# 3. Check if the query would find matches
print("\n3. TESTING ENROLLMENT QUERY:")
cursor.execute("""
    SELECT DISTINCT 
        l.id, l.phone, l.name, l.device_id, l.user_id, 
        s.id AS sequence_id, ss.`trigger` AS entry_trigger
    FROM leads l
    CROSS JOIN sequences s
    INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
    WHERE s.is_active = 1 
        AND ss.is_entry_point = 1
        AND l.`trigger` IS NOT NULL 
        AND l.`trigger` != ''
        AND l.device_id IS NOT NULL 
        AND l.user_id IS NOT NULL
        AND position(ss.`trigger` in l.`trigger`) > 0
        AND l.phone = '60108924904'
""")
matches = cursor.fetchall()
print(f"  Found {len(matches)} matches for enrollment")
for match in matches:
    print(f"    Lead: {match['phone']} -> Sequence: {match['sequence_id']}")

# 4. Check for existing messages
print("\n4. EXISTING MESSAGES CHECK:")
cursor.execute("""
    SELECT bm.*, s.name as sequence_name
    FROM broadcast_messages bm
    LEFT JOIN sequences s ON bm.sequence_id = s.id
    WHERE bm.recipient_phone = '60108924904'
    AND bm.created_at > DATE_SUB(NOW(), INTERVAL 1 DAY)
""")
messages = cursor.fetchall()
print(f"  Found {len(messages)} messages in last 24 hours")

# 5. Check the position function
print("\n5. TESTING POSITION FUNCTION:")
if lead and sequences:
    seq = sequences[0]
    cursor.execute("""
        SELECT 
            %s as lead_trigger,
            %s as step_trigger,
            position(%s in %s) as position_result
    """, (lead['trigger'], seq['step_trigger'], seq['step_trigger'], lead['trigger']))
    result = cursor.fetchone()
    print(f"  Lead trigger: '{result['lead_trigger']}'")
    print(f"  Step trigger: '{result['step_trigger']}'")
    print(f"  Position result: {result['position_result']}")
    print(f"  Should match: {result['position_result'] > 0}")

# 6. Check if device is online
print("\n6. DEVICE STATUS:")
if lead:
    cursor.execute("""
        SELECT id, device_name, status, platform
        FROM user_devices WHERE id = %s
    """, (lead['device_id'],))
    device = cursor.fetchone()
    if device:
        print(f"  Device: {device['device_name']}")
        print(f"  Status: {device['status']}")
        print(f"  Platform: {device['platform']}")

conn.close()
print("\n=== ANALYSIS COMPLETE ===")
