import pymysql

conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)

cursor = conn.cursor()

print("VERIFYING COLD Sequence (EXSTART) data for August 4, 2025\n")
print("=" * 80)

# 1. First, let's find the exact sequence
print("1. Finding the COLD Sequence with EXSTART:")
find_sequence = """
SELECT id, name, niche, user_id 
FROM sequences 
WHERE name LIKE '%COLD%' 
AND niche = 'EXSTART'
"""

cursor.execute(find_sequence)
sequences = cursor.fetchall()
for seq_id, name, niche, user_id in sequences:
    print(f"   Found: {name} (Niche: {niche})")
    print(f"   ID: {seq_id}")
    print(f"   User ID: {user_id}")

# Using the found ID
sequence_id = '0be82745-8f68-4352-abd0-0b405b43a905'
user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
target_date = '2025-08-04'

print(f"\n2. Checking broadcast_messages for this sequence on {target_date}:")
print("-" * 80)

# 2. Count total messages first
total_messages_query = """
SELECT COUNT(*) as total_rows
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(total_messages_query, (sequence_id, target_date))
total_rows = cursor.fetchone()[0]
print(f"   Total message rows for this date: {total_rows}")

# 3. The EXACT query used by the backend for Summary page
print(f"\n3. Running EXACT Summary Page Query:")
summary_query = """
SELECT 
    COUNT(DISTINCT recipient_phone) AS total,
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(summary_query, (sequence_id, target_date))
total, done, failed, remaining = cursor.fetchone()

print(f"   Query: WHERE sequence_id = '{sequence_id}'")
print(f"   AND DATE(scheduled_at) = '{target_date}'")
print(f"\n   Results:")
print(f"   - Done (sent with no error): {done}")
print(f"   - Failed: {failed}")
print(f"   - Remaining (pending/queued): {remaining}")
print(f"   - Total distinct contacts: {total}")
print(f"   - Should Send (done+failed+remaining): {done + failed + remaining}")

# 4. Let's verify by checking each status separately
print(f"\n4. Double-checking by counting each status:")
status_check = """
SELECT status, COUNT(*) as msg_count, COUNT(DISTINCT recipient_phone) as unique_count
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
GROUP BY status
ORDER BY status
"""

cursor.execute(status_check, (sequence_id, target_date))
statuses = cursor.fetchall()
for status, msg_count, unique_count in statuses:
    print(f"   Status '{status}': {msg_count} messages, {unique_count} unique phones")

# 5. Check with user_id filter (as Device Report does)
print(f"\n5. With user_id filter (like Device Report):")
device_query = """
SELECT 
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed_send,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining_send
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(device_query, (sequence_id, user_id, target_date))
done2, failed2, remaining2 = cursor.fetchone()
print(f"   Done: {done2}")
print(f"   Failed: {failed2}")
print(f"   Remaining: {remaining2}")
print(f"   Total: {done2 + failed2 + remaining2}")

# 6. Show some sample data to verify
print(f"\n6. Sample of actual data (first 5 records):")
sample_query = """
SELECT recipient_phone, status, error_message, scheduled_at
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
LIMIT 5
"""

cursor.execute(sample_query, (sequence_id, target_date))
samples = cursor.fetchall()
for phone, status, error, scheduled in samples:
    error_str = 'NULL' if error is None else f"'{error}'"
    print(f"   Phone: {phone}, Status: {status}, Error: {error_str}, Scheduled: {scheduled}")

print("\n" + "=" * 80)
print("FINAL ANSWER:")
print("=" * 80)
print(f"For COLD Sequence (EXSTART) on {target_date}:")
print(f"  REAL Done: {done}")
print(f"  REAL Failed: {failed}")
print(f"  REAL Remaining: {remaining}")
print(f"  REAL Total Should Send: {done + failed + remaining}")

cursor.close()
conn.close()
