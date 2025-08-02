import pymysql
import sys

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

print("=== DETAILED DUPLICATE ANALYSIS ===\n")

# Check specific duplicates
dup_detail_query = """
SELECT 
    recipient_phone,
    sequence_id,
    sequence_stepid,
    device_id,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(id) as message_ids,
    GROUP_CONCAT(status) as statuses,
    GROUP_CONCAT(DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s')) as created_times
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND status IN ('pending', 'sent')
GROUP BY recipient_phone, sequence_id, sequence_stepid, device_id
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC
"""

cursor.execute(dup_detail_query)
duplicates = cursor.fetchall()

print(f"Found {len(duplicates)} duplicate groups\n")

for i, dup in enumerate(duplicates):
    print(f"Duplicate Group {i+1}:")
    print(f"  Phone: {dup['recipient_phone']}")
    print(f"  Sequence: {dup['sequence_id']}")
    print(f"  Step: {dup['sequence_stepid']}")
    print(f"  Device: {dup['device_id']}")
    print(f"  Count: {dup['duplicate_count']}")
    print(f"  Statuses: {dup['statuses']}")
    print(f"  Created: {dup['created_times']}")
    print(f"  Message IDs: {dup['message_ids'][:50]}...")
    print()

# Check if these are old duplicates or new ones
print("\n=== CHECKING DUPLICATE AGES ===")
recent_dup_query = """
SELECT 
    COUNT(*) as recent_duplicates
FROM (
    SELECT 
        recipient_phone,
        sequence_id,
        sequence_stepid,
        device_id,
        COUNT(*) as duplicate_count,
        MAX(created_at) as latest_created
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    AND status IN ('pending', 'sent')
    GROUP BY recipient_phone, sequence_id, sequence_stepid, device_id
    HAVING COUNT(*) > 1
    AND MAX(created_at) > DATE_SUB(NOW(), INTERVAL 1 HOUR)
) as recent
"""

cursor.execute(recent_dup_query)
recent = cursor.fetchone()
print(f"Recent duplicates (last hour): {recent['recent_duplicates']}")

# Clean up old duplicates
print("\n=== CLEANING OLD DUPLICATES ===")
cleanup_query = """
DELETE bm1 FROM broadcast_messages bm1
INNER JOIN broadcast_messages bm2 
WHERE bm1.recipient_phone = bm2.recipient_phone
AND bm1.sequence_id = bm2.sequence_id  
AND bm1.sequence_stepid = bm2.sequence_stepid
AND bm1.device_id = bm2.device_id
AND bm1.status = 'pending'
AND bm2.status = 'pending'
AND bm1.created_at > bm2.created_at
"""

print("Cleaning duplicates...")
cursor.execute(cleanup_query)
deleted = cursor.rowcount
connection.commit()
print(f"Deleted {deleted} duplicate messages")

cursor.close()
connection.close()
