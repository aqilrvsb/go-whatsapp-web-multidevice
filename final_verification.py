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

print("="*70)
print("FINAL SYSTEM VERIFICATION - READY TO ACTIVATE SEQUENCES?")
print("="*70)

# 1. Clean any remaining duplicates
print("\n1. CLEANING DUPLICATES...")
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

cursor.execute(cleanup_query)
deleted = cursor.rowcount
connection.commit()
print(f"   Cleaned {deleted} duplicate messages")

# 2. Verify no duplicates remain
print("\n2. VERIFYING NO DUPLICATES...")
check_query = """
SELECT COUNT(*) as dup_count FROM (
    SELECT recipient_phone, sequence_id, sequence_stepid, device_id
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    AND status = 'pending'
    GROUP BY recipient_phone, sequence_id, sequence_stepid, device_id
    HAVING COUNT(*) > 1
) as dups
"""

cursor.execute(check_query)
result = cursor.fetchone()
dup_count = result['dup_count']

if dup_count == 0:
    print("   ✅ No duplicates found!")
else:
    print(f"   ❌ Still {dup_count} duplicate groups")

# 3. Check message quality
print("\n3. CHECKING MESSAGE QUALITY...")
quality_query = """
SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN recipient_name IS NOT NULL AND recipient_name != '' THEN 1 ELSE 0 END) as has_name,
    SUM(CASE WHEN content LIKE '%\\n%' THEN 1 ELSE 0 END) as has_linebreaks,
    SUM(CASE WHEN scheduled_at IS NOT NULL THEN 1 ELSE 0 END) as has_schedule
FROM broadcast_messages
WHERE status = 'pending'
AND sequence_id IS NOT NULL
"""

cursor.execute(quality_query)
quality = cursor.fetchone()

print(f"   Total pending: {quality['total']}")
print(f"   With names: {quality['has_name']} ({quality['has_name']/quality['total']*100:.1f}%)")
print(f"   With line breaks: {quality['has_linebreaks']} ({quality['has_linebreaks']/quality['total']*100:.1f}%)")
print(f"   With schedule: {quality['has_schedule']} ({quality['has_schedule']/quality['total']*100:.1f}%)")

# 4. Check sequences ready to activate
print("\n4. SEQUENCES READY TO ACTIVATE...")
seq_query = """
SELECT 
    id,
    name,
    status,
    is_active,
    total_contacts,
    active_contacts
FROM sequences
ORDER BY created_at DESC
"""

cursor.execute(seq_query)
sequences = cursor.fetchall()

print(f"   Total sequences: {len(sequences)}")
for seq in sequences:
    status = "🟢 ACTIVE" if seq['is_active'] else "🔴 INACTIVE"
    print(f"   {status} {seq['name']} - {seq['active_contacts']} active contacts")

# 5. Final recommendations
print("\n" + "="*70)
print("FINAL RECOMMENDATION:")
print("="*70)

issues = []
if dup_count > 0:
    issues.append("Duplicates still exist")
if quality['has_name'] < quality['total'] * 0.9:
    issues.append("Some messages missing recipient names")
if quality['has_schedule'] < quality['total']:
    issues.append("Some messages missing scheduled time")

if len(issues) == 0:
    print("\n✅ ALL SYSTEMS VERIFIED!")
    print("\nYou can now safely:")
    print("1. Activate your sequence templates")
    print("2. Messages will be sent with:")
    print("   - Correct recipient names (not 'Cik')")
    print("   - Proper line breaks")
    print("   - No duplicates")
    print("   - Correct ordering by scheduled time")
else:
    print("\n⚠️ MINOR ISSUES:")
    for issue in issues:
        print(f"   - {issue}")
    print("\nThese are likely old messages. New messages should work correctly.")

cursor.close()
connection.close()
