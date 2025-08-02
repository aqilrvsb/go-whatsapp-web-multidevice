import pymysql
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

print("=== SEQUENCE BUG ANALYSIS ===\n")

# 1. Check the exact lead data
print("1. CHECKING LEAD DATA:")
cursor.execute("""
    SELECT id, name, phone, `trigger`, niche, target_status, device_id
    FROM leads 
    WHERE phone = '60108924904'
""")
lead = cursor.fetchone()
if lead:
    print(f"  ID: {lead['id']}")
    print(f"  Name: {lead['name']}")
    print(f"  Phone: {lead['phone']}")
    print(f"  Trigger: '{lead['trigger']}' (type: {type(lead['trigger'])})")
    print(f"  Niche: '{lead['niche']}'")
    print(f"  Target Status: {lead['target_status']}")
    print(f"  Device ID: {lead['device_id']}")

# 2. Check if sequence was updated
print("\n2. CHECKING SEQUENCE DATA:")
cursor.execute("""
    SELECT id, name, `trigger`, status, device_id, min_delay_seconds, max_delay_seconds
    FROM sequences 
    WHERE name = 'meow'
""")
seq = cursor.fetchone()
if seq:
    print(f"  ID: {seq['id']}")
    print(f"  Name: {seq['name']}")
    print(f"  Trigger: '{seq['trigger']}'")
    print(f"  Status: {seq['status']}")
    print(f"  Device ID: {seq['device_id']}")
    print(f"  Delays: {seq['min_delay_seconds']}-{seq['max_delay_seconds']}s")

# 3. Check sequence steps
print("\n3. SEQUENCE STEPS:")
if seq:
    cursor.execute("""
        SELECT id, day_number, `trigger`, next_trigger, is_entry_point, content
        FROM sequence_steps 
        WHERE sequence_id = %s
        ORDER BY day_number
    """, (seq['id'],))
    steps = cursor.fetchall()
    for step in steps:
        print(f"  Step {step['day_number']}:")
        print(f"    Trigger: {step['trigger']}")
        print(f"    Content: {step['content']}")
        print(f"    Entry Point: {step['is_entry_point']}")

# 4. The SQL fix for the Go code
print("\n4. THE GO CODE BUG:")
print("  The error message shows the query is missing backticks around 'trigger'")
print("  File to fix: Look for sequence enrollment code in Go files")

# 5. Let's find similar working queries
print("\n5. LOOKING FOR SEQUENCE TRIGGER MATCHING:")
if seq and lead:
    # Check if trigger matches
    if lead['trigger'] == seq['trigger']:
        print(f"  ✓ Triggers match: '{lead['trigger']}' == '{seq['trigger']}'")
    else:
        print(f"  ✗ Triggers DON'T match: '{lead['trigger']}' != '{seq['trigger']}'")
        
    # Check what the enrollment would do
    print("\n  The enrollment process would:")
    print("  1. Find leads where trigger matches sequence trigger")
    print("  2. Get sequence steps (THIS IS WHERE IT FAILS)")
    print("  3. Create broadcast_message entries")

# Find the problematic Go file
print("\n=== FILES TO CHECK ===")
print("Look for files containing sequence enrollment logic:")
print("  - sequence_trigger.go")
print("  - sequence_repository.go")
print("  - sequence_processor.go")
print("\nSearch for queries that SELECT from sequence_steps")
print("and add backticks around 'trigger'")

conn.close()
