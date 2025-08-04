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

cursor = conn.cursor()

# Get today's date
today = datetime.now().strftime('%Y-%m-%d')
print(f"Investigating COLD Sequence data for: {today}\n")

# First, find the COLD Sequence ID
cursor.execute("SELECT id, name FROM sequences WHERE name LIKE '%COLD%' AND name LIKE '%EXSTART%'")
sequences = cursor.fetchall()
print("Found COLD Sequences:")
for seq_id, name in sequences:
    print(f"  ID: {seq_id}, Name: {name}")

# Assuming we want "COLD Sequence" with EXSTART
cold_sequence_id = None
for seq_id, name in sequences:
    if "COLD Sequence" in name or (name == "COLD Sequence"):
        cold_sequence_id = seq_id
        break

if not cold_sequence_id:
    print("\nCOLD Sequence not found!")
    exit()

print(f"\nUsing sequence ID: {cold_sequence_id}")

# 1. Check Summary Page Query (how sequence summary calculates)
print("\n1. SUMMARY PAGE CALCULATION (for sequence list):")
summary_query = """
SELECT 
    COUNT(DISTINCT recipient_phone) AS total_distinct,
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(summary_query, (cold_sequence_id, today))
result = cursor.fetchone()
if result:
    total, done, failed, remaining = result
    should_send = done + failed + remaining
    print(f"  Total Distinct: {total}")
    print(f"  Done Send: {done}")
    print(f"  Failed Send: {failed}")
    print(f"  Remaining Send: {remaining}")
    print(f"  Should Send (calculated): {should_send}")

# 2. Check Device Report Query (overall stats)
print("\n2. DEVICE REPORT OVERALL CALCULATION:")
device_query = """
SELECT 
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(device_query, (cold_sequence_id, today))
result = cursor.fetchone()
if result:
    done, failed, remaining = result
    should_send = done + failed + remaining
    print(f"  Done Send: {done}")
    print(f"  Failed Send: {failed}")
    print(f"  Remaining Send: {remaining}")
    print(f"  Should Send (calculated): {should_send}")

# 3. Check if there's a user_id filter missing
print("\n3. CHECKING WITH USER_ID FILTER:")
# First get a user_id from the sequence
cursor.execute("SELECT user_id FROM sequences WHERE id = %s", (cold_sequence_id,))
user_id = cursor.fetchone()[0] if cursor.fetchone() else None

if user_id:
    user_query = """
    SELECT 
        COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
        COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
        COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
    FROM broadcast_messages
    WHERE sequence_id = %s
    AND user_id = %s
    AND DATE(scheduled_at) = %s
    """
    
    cursor.execute(user_query, (cold_sequence_id, user_id, today))
    result = cursor.fetchone()
    if result:
        done, failed, remaining = result
        should_send = done + failed + remaining
        print(f"  With user_id filter:")
        print(f"  Done Send: {done}")
        print(f"  Failed Send: {failed}")
        print(f"  Remaining Send: {remaining}")
        print(f"  Should Send (calculated): {should_send}")

# 4. Check raw data to understand discrepancy
print("\n4. RAW DATA ANALYSIS:")
print("Checking all statuses for this sequence today:")
status_query = """
SELECT status, COUNT(DISTINCT recipient_phone) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
GROUP BY status
"""

cursor.execute(status_query, (cold_sequence_id, today))
statuses = cursor.fetchall()