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

# The date from the UI
ui_date = '2025-08-04'  # August 4, 2025
print(f"Investigating COLD Sequence data for date: {ui_date}\n")

# First, find the COLD Sequence ID
cursor.execute("SELECT id, name, user_id FROM sequences WHERE name LIKE '%COLD Sequence%' AND niche = 'EXSTART'")
result = cursor.fetchone()
if not result:
    print("COLD Sequence not found!")
    exit()

cold_sequence_id, sequence_name, user_id = result
print(f"Found sequence: {sequence_name}")
print(f"Sequence ID: {cold_sequence_id}")
print(f"User ID: {user_id}")

# 1. EXACT Summary Page Query (with user_id filter)
print("\n1. SUMMARY PAGE QUERY (matches GetSequenceSummary):")
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

cursor.execute(summary_query, (cold_sequence_id, ui_date))
result = cursor.fetchone()
if result:
    total, done, failed, remaining = result
    should_send = done + failed + remaining
    print(f"  Total (distinct): {total}")
    print(f"  Done Send: {done}")
    print(f"  Failed Send: {failed}")
    print(f"  Remaining Send: {remaining}")
    print(f"  Should Send (UI calc): {should_send}")
    print(f"  Expected from UI: 427")

# 2. Device Report Query (with user_id - matches device report)
print("\n2. DEVICE REPORT QUERY (with user_id filter):")
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

cursor.execute(device_query, (cold_sequence_id, user_id, ui_date))
result = cursor.fetchone()
if result:
    done, failed, remaining = result
    should_send = done + failed + remaining
    print(f"  Done Send: {done}")
    print(f"  Failed Send: {failed}")
    print(f"  Remaining Send: {remaining}")
    print(f"  Should Send (calculated): {should_send}")
    print(f"  Device Report shows: 507")

# 3. Check if there are multiple dates in the data
print("\n3. CHECKING DATE DISTRIBUTION:")
date_query = """
SELECT DATE(scheduled_at) as sched_date, COUNT(DISTINCT recipient_phone) as contacts
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
GROUP BY DATE(scheduled_at)
ORDER BY sched_date DESC
LIMIT 10
"""

cursor.execute(date_query, (cold_sequence_id, user_id))
dates = cursor.fetchall()
print("Dates with data for this sequence:")
for date, count in dates:
    print(f"  {date}: {count} contacts")

# 4. Check if scheduled_at has NULL values
print("\n4. CHECKING FOR NULL scheduled_at:")
null_query = """
SELECT COUNT(DISTINCT recipient_phone) as null_scheduled
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND scheduled_at IS NULL
"""

cursor.execute(null_query, (cold_sequence_id, user_id))
null_count = cursor.fetchone()[0]
print(f"  Contacts with NULL scheduled_at: {null_count}")

# 5. Check the step-by-step data
print("\n5. STEP-BY-STEP BREAKDOWN:")
step_query = """
SELECT 
    ss.day_number,
    COUNT(DISTINCT bm.recipient_phone) as total,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) AS done,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) AS remaining
FROM broadcast_messages bm
JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
WHERE bm.sequence_id = %s
AND bm.user_id = %s
AND DATE(bm.scheduled_at) = %s
GROUP BY ss.day_number
ORDER BY ss.day_number
"""

cursor.execute(step_query, (cold_sequence_id, user_id, ui_date))
steps = cursor.fetchall()
print("Per step breakdown:")
total_all_steps = {'done': 0, 'failed': 0, 'remaining': 0}
for day, total, done, failed, remaining in steps:
    should = done + failed + remaining
    print(f"  Step {day}: Total={total}, Done={done}, Failed={failed}, Remaining={remaining}, Should={should}")
    total_all_steps['done'] += done
    total_all_steps['failed'] += failed
    total_all_steps['remaining'] += remaining

print(f"\n  Sum across all steps:")
print(f"  Done: {total_all_steps['done']}")
print(f"  Failed: {total_all_steps['failed']}")
print(f"  Remaining: {total_all_steps['remaining']}")
print(f"  Total Should: {sum(total_all_steps.values())}")

# 6. Check for created_at vs scheduled_at
print("\n6. CREATED_AT vs SCHEDULED_AT comparison:")
created_query = """
SELECT 
    'scheduled_at' as date_type,
    COUNT(DISTINCT recipient_phone) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(scheduled_at) = %s

UNION ALL

SELECT 
    'created_at' as date_type,
    COUNT(DISTINCT recipient_phone) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(created_at) = %s
"""

cursor.execute(created_query, (cold_sequence_id, user_id, ui_date, cold_sequence_id, user_id, ui_date))
date_results = cursor.fetchall()
for date_type, count in date_results:
    print(f"  Using {date_type}: {count} contacts")

cursor.close()
conn.close()
