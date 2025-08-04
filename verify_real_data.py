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

# COLD Sequence details
cold_sequence_id = '0be82745-8f68-4352-abd0-0b405b43a905'
user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
target_date = '2025-08-04'

print("=" * 80)
print(f"VERIFYING REAL DATA FOR COLD SEQUENCE ON {target_date}")
print("=" * 80)

# 1. Get the TRUE count from database
print("\n1. ACTUAL DATABASE COUNTS (THE TRUTH):")
truth_query = """
SELECT 
    COUNT(DISTINCT recipient_phone) as unique_contacts,
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed_send,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining_send
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(truth_query, (cold_sequence_id, target_date))
unique_contacts, done, failed, remaining = cursor.fetchone()
should_send = done + failed + remaining

print(f"  Unique Contacts: {unique_contacts}")
print(f"  Done Send: {done}")
print(f"  Failed Send: {failed}")
print(f"  Remaining Send: {remaining}")
print(f"  Should Send (done+failed+remaining): {should_send}")

# 2. Compare with what UI shows
print("\n2. UI DISPLAY COMPARISON:")
print("  SUMMARY PAGE shows:")
print("    - Total Should Send: 427")
print("    - Done: 158")
print("    - Failed: 19")
print("    - Remaining: 250")
print("    - Calculation: 158 + 19 + 250 = 427 [/]")

print("\n  DEVICE REPORT shows:")
print("    - Total Should Send: 507")
print("    - Done: 207") 
print("    - Failed: 50")
print("    - Remaining: 250")
print("    - Calculation: 207 + 50 + 250 = 507 ✓")

print("\n  ACTUAL DATABASE shows:")
print(f"    - Total Should Send: {should_send}")
print(f"    - Done: {done}")
print(f"    - Failed: {failed}")
print(f"    - Remaining: {remaining}")
print(f"    - Calculation: {done} + {failed} + {remaining} = {should_send} ✓")

# 3. Check why numbers might be different
print("\n3. INVESTIGATING DISCREPANCIES:")

# Check if device report is including other dates
print("\n  a) Checking if Device Report includes multiple dates:")
multi_date_query = """
SELECT 
    DATE(scheduled_at) as date,
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
FROM broadcast_messages
WHERE sequence_id = %s
GROUP BY DATE(scheduled_at)
ORDER BY date DESC
LIMIT 5
"""

cursor.execute(multi_date_query, (cold_sequence_id,))
dates = cursor.fetchall()
print("    Recent dates with data:")
cumulative_done = 0
cumulative_failed = 0
for date, d, f, r in dates:
    cumulative_done += d
    cumulative_failed += f
    print(f"    {date}: Done={d}, Failed={f}, Remaining={r}")
    if str(date) <= target_date:
        print(f"    Cumulative up to {date}: Done={cumulative_done}, Failed={cumulative_failed}")

# Check specific statuses
print("\n  b) Checking all status values in database:")
status_query = """
SELECT status, COUNT(DISTINCT recipient_phone) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
GROUP BY status
"""

cursor.execute(status_query, (cold_sequence_id, target_date))
statuses = cursor.fetchall()
print(f"    Status breakdown for {target_date}:")
for status, count in statuses:
    print(f"    - {status}: {count} contacts")

# 4. Check for data inconsistency
print("\n4. CHECKING FOR DATA ISSUES:")

# Check if some messages have weird status or error_message combinations
weird_query = """
SELECT 
    status,
    CASE WHEN error_message IS NULL THEN 'NULL' 
         WHEN error_message = '' THEN 'EMPTY' 
         ELSE 'HAS_ERROR' END as error_state,
    COUNT(DISTINCT recipient_phone) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
GROUP BY status, error_state
"""

cursor.execute(weird_query, (cold_sequence_id, target_date))
weird = cursor.fetchall()
print("  Status and error_message combinations:")
for status, error_state, count in weird:
    print(f"    {status} with error_message {error_state}: {count} contacts")

# 5. Final verdict
print("\n" + "=" * 80)
print("VERDICT: WHICH DATA IS CORRECT?")
print("=" * 80)
print(f"\nThe REAL data from database for {target_date}:")
print(f"  ✓ Done Send: {done}")
print(f"  ✓ Failed Send: {failed}")
print(f"  ✓ Remaining Send: {remaining}")
print(f"  ✓ Total Should Send: {should_send}")

print("\nCONCLUSION:")
if should_send == 427:
    print("  → SUMMARY PAGE is showing CORRECT data")
    print("  → DEVICE REPORT is showing INCORRECT data (possibly multiple dates)")
elif should_send == 507:
    print("  → DEVICE REPORT is showing CORRECT data")
    print("  → SUMMARY PAGE is showing OUTDATED/CACHED data")
else:
    print(f"  → BOTH UI pages are WRONG!")
    print(f"  → The correct total should be: {should_send}")
    print("  → Summary page shows: 427 (WRONG)")
    print("  → Device report shows: 507 (WRONG)")

cursor.close()
conn.close()
