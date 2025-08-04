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

print("Connected to MySQL successfully\n")
cursor = conn.cursor()

# Get today's date
today = datetime.now().strftime('%Y-%m-%d')
print(f"Investigating discrepancy for date: {today}\n")

# 1. Check unique contacts across ALL sequences for today
print("1. Checking DISTINCT recipient_phone across ALL sequences:")
query1 = """
SELECT COUNT(DISTINCT recipient_phone) as unique_contacts_overall
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND DATE(scheduled_at) = %s
"""
cursor.execute(query1, (today,))
unique_overall = cursor.fetchone()[0]
print(f"   Unique contacts across all sequences: {unique_overall}")

# 2. Check total rows (not distinct) 
print("\n2. Checking total broadcast_messages rows (not distinct):")
query2 = """
SELECT COUNT(*) as total_rows
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND DATE(scheduled_at) = %s
"""
cursor.execute(query2, (today,))
total_rows = cursor.fetchone()[0]
print(f"   Total rows: {total_rows}")

# 3. Check if same contact appears in multiple sequences
print("\n3. Checking contacts that appear in multiple sequences:")
query3 = """
SELECT recipient_phone, COUNT(DISTINCT sequence_id) as sequence_count
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND DATE(scheduled_at) = %s
GROUP BY recipient_phone
HAVING COUNT(DISTINCT sequence_id) > 1
ORDER BY sequence_count DESC
LIMIT 10
"""
cursor.execute(query3, (today,))
duplicates = cursor.fetchall()
print(f"   Found {len(duplicates)} contacts in multiple sequences")
if duplicates:
    print("   Sample contacts in multiple sequences:")
    for phone, count in duplicates[:5]:
        print(f"   - {phone}: appears in {count} sequences")

# 4. Check the status breakdown with duplicates consideration
print("\n4. Status breakdown (DISTINCT per status, may have overlaps):")
statuses = ['sent', 'failed', 'pending', 'queued']
for status in statuses:
    if status in ['pending', 'queued']:
        query = """
        SELECT COUNT(DISTINCT recipient_phone) 
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
        AND DATE(scheduled_at) = %s
        AND status IN ('pending', 'queued')
        """
        cursor.execute(query, (today,))
    else:
        query = """
        SELECT COUNT(DISTINCT recipient_phone) 
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
        AND DATE(scheduled_at) = %s
        AND status = %s
        """
        cursor.execute(query, (today, status))
    count = cursor.fetchone()[0]
    if status in ['pending', 'queued'] and count > 0:
        print(f"   {status}/queued: {count}")
        break
    elif status not in ['pending', 'queued']:
        print(f"   {status}: {count}")

# 5. Check what the UI shows vs actual calculation method
print("\n5. Understanding the calculation difference:")
print("   UI shows:")
print("   - Total Should Send: 2322 (sum of individual sequences)")
print("   - This is counting the same contact multiple times if in different sequences")
print("\n   Database shows:")
print(f"   - Unique contacts: {unique_overall} (DISTINCT across all sequences)")
print("   - This counts each contact only once")

# 6. Verify the sum matches what UI shows
print("\n6. Calculating sum as UI does (per sequence, allowing duplicates):")
query6 = """
SELECT 
    s.name,
    COUNT(DISTINCT bm.recipient_phone) as contacts_in_sequence
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id 
    AND DATE(bm.scheduled_at) = %s
WHERE bm.recipient_phone IS NOT NULL
GROUP BY s.id, s.name
"""
cursor.execute(query6, (today,))
sequences = cursor.fetchall()
total_sum = 0
for name, count in sequences:
    print(f"   {name}: {count} contacts")
    total_sum += count
print(f"\n   Total (allowing duplicates): {total_sum}")

cursor.close()
conn.close()
