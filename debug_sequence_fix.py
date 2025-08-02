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

print("=== SEQUENCE ISSUE ANALYSIS ===\n")

# 1. The main issue with your sequence
print("1. ISSUE WITH YOUR 'meow' SEQUENCE:")
cursor.execute("""
    SELECT * FROM sequences WHERE name = 'meow'
""")
seq = cursor.fetchone()
print(f"  Sequence Name: {seq['name']}")
print(f"  Sequence Trigger: '{seq['trigger']}' (EMPTY!)")
print(f"  Status: {seq['status']}")
print(f"  Target Status: {seq['target_status']}")
print(f"  Device ID: {seq['device_id']} (NULL!)")
print(f"  Min/Max Delay: {seq['min_delay_seconds']}/{seq['max_delay_seconds']} (BOTH 0!)")

print("\n  [X] PROBLEMS FOUND:")
print("     - Sequence trigger is EMPTY (should be 'meow')")
print("     - Device ID is NULL (sequences need a device)")
print("     - Min/Max delays are 0 (should be like 10/30)")

# 2. Check the step trigger
print("\n2. SEQUENCE STEP DETAILS:")
cursor.execute("""
    SELECT * FROM sequence_steps WHERE sequence_id = %s
""", (seq['id'],))
step = cursor.fetchone()
print(f"  Step Trigger: '{step['trigger']}' (This is correct)")
print(f"  Entry Point: {step['is_entry_point']}")

# 3. Show correct lead with meow trigger
print("\n3. LEAD WITH 'meow' TRIGGER:")
cursor.execute("""
    SELECT l.*, ud.status as device_status
    FROM leads l
    LEFT JOIN user_devices ud ON l.device_id = ud.id
    WHERE l.`trigger` = 'meow'
""")
lead = cursor.fetchone()
if lead:
    print(f"  Name: {lead['name']}")
    print(f"  Phone: {lead['phone']}")
    print(f"  Device: {lead['device_id']}")
    print(f"  Device Status: {lead['device_status']}")

# 4. Fix suggestions
print("\n=== HOW TO FIX ===")
print("\n1. UPDATE SEQUENCE TRIGGER:")
print(f"UPDATE sequences SET `trigger` = 'meow' WHERE id = '{seq['id']}';")

print("\n2. SET A DEVICE ID (use an online device):")
cursor.execute("""
    SELECT id, device_name FROM user_devices 
    WHERE status = 'online' AND platform IS NOT NULL
    LIMIT 1
""")
online_device = cursor.fetchone()
if online_device:
    print(f"UPDATE sequences SET device_id = '{online_device['id']}' WHERE id = '{seq['id']}';")
    print(f"   (This will use device: {online_device['device_name']})")

print("\n3. SET PROPER DELAYS:")
print(f"UPDATE sequences SET min_delay_seconds = 10, max_delay_seconds = 30 WHERE id = '{seq['id']}';")

print("\n4. ALSO CHECK WHY NO MESSAGES ARE BEING CREATED:")
print("   The sequence trigger processor looks for:")
print("   - Sequences where trigger matches lead trigger")
print("   - But your sequence trigger is EMPTY!")
print("   - That's why it found 3502 leads (all with empty triggers)")

# 5. Show the fix SQL all together
print("\n=== RUN THIS SQL TO FIX: ===")
if online_device:
    print(f"""
UPDATE sequences 
SET `trigger` = 'meow',
    device_id = '{online_device['id']}',
    min_delay_seconds = 10,
    max_delay_seconds = 30
WHERE id = '{seq['id']}';
""")

conn.close()
