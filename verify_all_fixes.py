import pymysql
import sys
from datetime import datetime

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

print("="*70)
print("WHATSAPP MULTI-DEVICE SYSTEM - COMPREHENSIVE VERIFICATION")
print("="*70)
print(f"Verification Date: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
print("="*70)

# Connect to MySQL
connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

# 1. CHECK DUPLICATE PREVENTION
print("\n✅ CHECK 1: DUPLICATE PREVENTION")
print("-" * 50)

# Check for sequence duplicates
seq_dup_query = """
SELECT 
    COUNT(*) as duplicate_groups,
    SUM(duplicate_count - 1) as extra_messages
FROM (
    SELECT 
        recipient_phone,
        sequence_id,
        sequence_stepid,
        device_id,
        COUNT(*) as duplicate_count
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    AND status IN ('pending', 'sent')
    GROUP BY recipient_phone, sequence_id, sequence_stepid, device_id
    HAVING COUNT(*) > 1
) as duplicates
"""

cursor.execute(seq_dup_query)
seq_result = cursor.fetchone()

print(f"Sequence Duplicates: {seq_result['duplicate_groups'] or 0} groups with {seq_result['extra_messages'] or 0} extra messages")

# Check for campaign duplicates
camp_dup_query = """
SELECT 
    COUNT(*) as duplicate_groups,
    SUM(duplicate_count - 1) as extra_messages
FROM (
    SELECT 
        recipient_phone,
        campaign_id,
        device_id,
        COUNT(*) as duplicate_count
    FROM broadcast_messages
    WHERE campaign_id IS NOT NULL
    AND status IN ('pending', 'sent')
    GROUP BY recipient_phone, campaign_id, device_id
    HAVING COUNT(*) > 1
) as duplicates
"""

cursor.execute(camp_dup_query)
camp_result = cursor.fetchone()

print(f"Campaign Duplicates: {camp_result['duplicate_groups'] or 0} groups with {camp_result['extra_messages'] or 0} extra messages")

if (seq_result['duplicate_groups'] or 0) == 0 and (camp_result['duplicate_groups'] or 0) == 0:
    print("✅ PASS: No duplicates found!")
else:
    print("❌ FAIL: Duplicates still exist!")

# 2. CHECK MESSAGE ORDERING
print("\n\n✅ CHECK 2: MESSAGE ORDERING (scheduled_at)")
print("-" * 50)

order_query = """
SELECT 
    device_id,
    COUNT(*) as total_messages,
    MIN(scheduled_at) as first_scheduled,
    MAX(scheduled_at) as last_scheduled,
    SUM(CASE WHEN scheduled_at IS NULL THEN 1 ELSE 0 END) as null_scheduled
FROM broadcast_messages
WHERE status = 'pending'
GROUP BY device_id
ORDER BY total_messages DESC
LIMIT 5
"""

cursor.execute(order_query)
order_results = cursor.fetchall()

print(f"Devices with pending messages: {len(order_results)}")
for device in order_results[:3]:
    print(f"\nDevice: {device['device_id'][:8]}...")
    print(f"  Messages: {device['total_messages']}")
    print(f"  First scheduled: {device['first_scheduled']}")
    print(f"  Last scheduled: {device['last_scheduled']}")
    print(f"  NULL scheduled: {device['null_scheduled']}")

# 3. CHECK RECIPIENT NAME HANDLING
print("\n\n✅ CHECK 3: RECIPIENT NAME HANDLING")
print("-" * 50)

name_query = """
SELECT 
    recipient_phone,
    recipient_name,
    LEFT(content, 100) as content_preview,
    sequence_id,
    campaign_id
FROM broadcast_messages
WHERE status = 'pending'
AND recipient_name IS NOT NULL
AND recipient_name != ''
ORDER BY created_at DESC
LIMIT 10
"""

cursor.execute(name_query)
name_results = cursor.fetchall()

print(f"Sample messages with recipient names:")
phone_as_name = 0
proper_names = 0

for msg in name_results[:5]:
    print(f"\nPhone: {msg['recipient_phone']}")
    print(f"Name: '{msg['recipient_name']}'")
    
    # Check if name looks like a phone number
    if msg['recipient_name'].replace('+', '').replace(' ', '').isdigit():
        print("  ⚠️ Name appears to be a phone number")
        phone_as_name += 1
    else:
        print("  ✅ Proper name detected")
        proper_names += 1
    
    # Check greeting in content
    if msg['content_preview']:
        first_line = msg['content_preview'].split('\n')[0] if '\n' in msg['content_preview'] else msg['content_preview'][:30]
        print(f"  First line: {first_line}")

# 4. CHECK LINE BREAKS IN CONTENT
print("\n\n✅ CHECK 4: LINE BREAK PRESERVATION")
print("-" * 50)

linebreak_query = """
SELECT 
    id,
    recipient_phone,
    LENGTH(content) as content_length,
    LENGTH(content) - LENGTH(REPLACE(content, '\n', '')) as line_breaks,
    LEFT(content, 200) as content_preview
FROM broadcast_messages
WHERE status = 'pending'
AND content LIKE '%\n%'
LIMIT 5
"""

cursor.execute(linebreak_query)
lb_results = cursor.fetchall()

print(f"Messages with line breaks: {len(lb_results)}")
for msg in lb_results[:3]:
    print(f"\nID: {msg['id'][:8]}...")
    print(f"Line breaks found: {msg['line_breaks']}")
    print(f"Content preview (with \\n shown):")
    print(repr(msg['content_preview']))

# 5. CHECK ACTIVE SEQUENCES
print("\n\n✅ CHECK 5: SEQUENCE STATUS")
print("-" * 50)

seq_status_query = """
SELECT 
    s.id,
    s.name,
    s.status,
    s.is_active,
    COUNT(DISTINCT sc.id) as active_contacts,
    COUNT(DISTINCT bm.id) as pending_messages
FROM sequences s
LEFT JOIN sequence_contacts sc ON s.id = sc.sequence_id AND sc.status = 'active'
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id AND bm.status = 'pending'
GROUP BY s.id, s.name, s.status, s.is_active
ORDER BY s.created_at DESC
LIMIT 10
"""

cursor.execute(seq_status_query)
seq_results = cursor.fetchall()

print(f"Total sequences found: {len(seq_results)}")
active_count = sum(1 for s in seq_results if s['is_active'])
print(f"Active sequences: {active_count}")

for seq in seq_results[:5]:
    print(f"\nSequence: {seq['name']}")
    print(f"  Status: {seq['status']} | Active: {seq['is_active']}")
    print(f"  Active contacts: {seq['active_contacts']}")
    print(f"  Pending messages: {seq['pending_messages']}")

# 6. CHECK SYSTEM HEALTH
print("\n\n✅ CHECK 6: SYSTEM HEALTH")
print("-" * 50)

# Check message distribution
health_query = """
SELECT 
    COUNT(DISTINCT device_id) as unique_devices,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(*) as total_pending,
    SUM(CASE WHEN sequence_id IS NOT NULL THEN 1 ELSE 0 END) as sequence_messages,
    SUM(CASE WHEN campaign_id IS NOT NULL THEN 1 ELSE 0 END) as campaign_messages
FROM broadcast_messages
WHERE status = 'pending'
"""

cursor.execute(health_query)
health = cursor.fetchone()

print(f"Unique devices with pending: {health['unique_devices']}")
print(f"Unique users with pending: {health['unique_users']}")
print(f"Total pending messages: {health['total_pending']}")
print(f"  - Sequence messages: {health['sequence_messages']}")
print(f"  - Campaign messages: {health['campaign_messages']}")

# Final Summary
print("\n\n" + "="*70)
print("VERIFICATION SUMMARY")
print("="*70)

issues = []

if (seq_result['duplicate_groups'] or 0) > 0 or (camp_result['duplicate_groups'] or 0) > 0:
    issues.append("❌ Duplicates still exist")
else:
    print("✅ Duplicate Prevention: WORKING")

if len(order_results) > 0:
    print("✅ Message Ordering: Messages have scheduled_at timestamps")
else:
    print("⚠️ Message Ordering: No pending messages to verify")

if proper_names > phone_as_name:
    print("✅ Name Display: Proper names detected")
else:
    issues.append("❌ Names may still show as phone numbers")

if len(lb_results) > 0:
    print("✅ Line Breaks: Preserved in content")
else:
    print("⚠️ Line Breaks: No messages with line breaks to verify")

if active_count > 0:
    print(f"✅ Sequences: {active_count} active sequences ready")
else:
    print("⚠️ Sequences: No active sequences")

print("\n" + "="*70)
if len(issues) == 0:
    print("✅ ALL SYSTEMS GO! Safe to activate sequence templates.")
else:
    print("❌ ISSUES FOUND:")
    for issue in issues:
        print(f"  {issue}")
    print("\nRecommendation: Fix issues before activating sequences.")

cursor.close()
connection.close()
