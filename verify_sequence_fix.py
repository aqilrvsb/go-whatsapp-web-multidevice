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

print("=== FINAL VERIFICATION ===\n")

# 1. Check lead
print("1. LEAD STATUS:")
cursor.execute("""
    SELECT id, name, phone, `trigger`, device_id, user_id
    FROM leads WHERE phone = '60108924904'
""")
lead = cursor.fetchone()
print(f"  Name: {lead['name']}")
print(f"  Phone: {lead['phone']}")
print(f"  Trigger: '{lead['trigger']}'")
print(f"  Device ID: {lead['device_id']}")
print(f"  User ID: {lead['user_id']}")

# 2. Check sequence
print("\n2. SEQUENCE STATUS:")
cursor.execute("""
    SELECT id, name, `trigger`, status, is_active, device_id, 
           min_delay_seconds, max_delay_seconds
    FROM sequences WHERE name = 'meow'
""")
seq = cursor.fetchone()
print(f"  Name: {seq['name']}")
print(f"  Trigger: '{seq['trigger']}'")
print(f"  Status: {seq['status']}")
print(f"  Is Active: {seq['is_active']}")
print(f"  Device ID: {seq['device_id']}")
print(f"  Delays: {seq['min_delay_seconds']}-{seq['max_delay_seconds']}s")

# 3. Check sequence steps
print("\n3. SEQUENCE STEPS:")
cursor.execute("""
    SELECT id, day_number, `trigger`, is_entry_point, content
    FROM sequence_steps WHERE sequence_id = %s
""", (seq['id'],))
steps = cursor.fetchall()
for step in steps:
    print(f"  Step {step['day_number']}: trigger='{step['trigger']}', entry_point={step['is_entry_point']}")

# 4. Check if device is online
print("\n4. DEVICE STATUS:")
cursor.execute("""
    SELECT id, device_name, status, platform
    FROM user_devices WHERE id = %s
""", (seq['device_id'],))
device = cursor.fetchone()
if device:
    print(f"  Device: {device['device_name']}")
    print(f"  Status: {device['status']}")
    print(f"  Platform: {device['platform']}")

# 5. Check for existing messages
print("\n5. CHECKING FOR MESSAGES:")
cursor.execute("""
    SELECT COUNT(*) as count
    FROM broadcast_messages
    WHERE recipient_phone = '60108924904' 
    AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)
""")
count = cursor.fetchone()['count']
print(f"  Messages created in last hour: {count}")

print("\n=== SUMMARY ===")
if lead['trigger'] == 'meow' and seq['trigger'] == 'meow':
    print("✓ Lead and sequence triggers match!")
    print("✓ Database is properly configured")
    print("\nNext steps:")
    print("1. Run the build script: build_sequence_fix.bat")
    print("2. Run the fixed executable: whatsapp_sequence_fixed.exe rest")
    print("3. Wait 15-30 seconds for the sequence trigger to run")
    print("4. Check broadcast_messages table for new entries")
else:
    print("✗ Triggers don't match - please check the configuration")

conn.close()
