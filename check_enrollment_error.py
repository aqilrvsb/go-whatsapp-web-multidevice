import pymysql
from datetime import datetime
import sys

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

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

print("=== CHECKING SEQUENCE ENROLLMENT ERROR ===\n")

# 1. Show the lead that's trying to enroll
print("1. LEAD TRYING TO ENROLL:")
cursor.execute("""
    SELECT * FROM leads WHERE phone = '60108924904'
""")
lead = cursor.fetchone()
print(f"  Name: {lead['name']}")
print(f"  Phone: {lead['phone']}")
print(f"  Trigger: {lead['trigger']}")
print(f"  Status: {lead['status']}")

# 2. Show the sequence it's trying to enroll into
print("\n2. SEQUENCE DETAILS:")
cursor.execute("""
    SELECT * FROM sequences WHERE `trigger` = 'meow'
""")
seq = cursor.fetchone()
if seq:
    print(f"  Name: {seq['name']}")
    print(f"  ID: {seq['id']}")
    print(f"  Trigger: {seq['trigger']}")
    print(f"  Status: {seq['status']}")

# 3. The problematic query (what the Go code is trying to run)
print("\n3. THE FAILING QUERY (from error log):")
print("  The Go code is trying to run a query like:")
print("  SELECT id, sequence_id, day_number, trigger, next_trigger, ...")
print("  WITHOUT backticks around 'trigger'")

# 4. Show the correct query
print("\n4. CORRECT QUERY (with backticks):")
if seq:
    cursor.execute("""
        SELECT id, sequence_id, day_number, `trigger`, next_trigger, trigger_delay_hours,
               message_type, content, media_url, is_entry_point,
               min_delay_seconds, max_delay_seconds
        FROM sequence_steps 
        WHERE sequence_id = %s AND is_entry_point = 1
    """, (seq['id'],))
    steps = cursor.fetchall()
    print(f"  Found {len(steps)} entry point steps")
    for step in steps:
        print(f"    - Day {step['day_number']}: {step['trigger']} -> {step['next_trigger']}")

# 5. Check if any messages were created despite the error
print("\n5. CHECKING BROADCAST_MESSAGES:")
cursor.execute("""
    SELECT COUNT(*) as count 
    FROM broadcast_messages 
    WHERE recipient_phone = '60108924904'
    AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)
""")
count = cursor.fetchone()['count']
print(f"  Messages created in last hour: {count}")

print("\n=== THE ISSUE ===")
print("The Go code in the sequence enrollment has a SQL query that doesn't")
print("escape the 'trigger' keyword with backticks. This is a code bug.")
print("\nThe error happens in the sequence enrollment process when it tries")
print("to get the sequence steps.")

print("\n=== TEMPORARY WORKAROUND ===")
print("You can manually insert a message to test:")
if seq:
    print(f"""
INSERT INTO broadcast_messages (
    id, user_id, device_id, sequence_id,
    recipient_phone, recipient_name, message_type,
    content, status, created_at
) VALUES (
    UUID(),
    '{seq['user_id']}',
    '{lead['device_id']}',
    '{seq['id']}',
    '60108924904',
    'Aqil 1',
    'text',
    'asdsad',
    'pending',
    NOW()
);
""")

print("\n=== PERMANENT FIX ===")
print("The Go source code needs to be updated to add backticks around 'trigger'")
print("in the sequence enrollment queries.")

conn.close()
