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
cold_sequence_id = '0be82745-8f68-4352-abd0-0b405b43a905'
user_id = 'de078f16-3266-4ab3-8153-a248b015228f'

print(f"Deep investigation for COLD Sequence on {ui_date}\n")

# 1. Check if the summary page might be caching or showing different calculation
print("1. CHECKING WHY SUMMARY SHOWS 427 vs DATABASE SHOWS 443:")
print("\nPossible reasons:")
print("a) Some contacts might have multiple statuses")
print("b) The UI might be showing cached data")
print("c) There might be duplicates being filtered differently")

# Check for contacts with multiple statuses
multi_status_query = """
SELECT recipient_phone, GROUP_CONCAT(DISTINCT status) as statuses, COUNT(*) as count
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(scheduled_at) = %s
GROUP BY recipient_phone
HAVING COUNT(DISTINCT status) > 1
LIMIT 10
"""

cursor.execute(multi_status_query, (cold_sequence_id, user_id, ui_date))
multi_status = cursor.fetchall()
print(f"\nContacts with multiple statuses: {len(multi_status)}")
for phone, statuses, count in multi_status[:5]:
    print(f"  {phone}: {statuses} ({count} records)")

# 2. Check why step totals (519) don't match overall (443)
print("\n2. WHY STEP TOTALS (519) DON'T MATCH OVERALL (443):")
print("This happens when the same contact appears in multiple steps")

# Check contacts appearing in multiple steps
multi_step_query = """
SELECT recipient_phone, COUNT(DISTINCT sequence_stepid) as step_count
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(scheduled_at) = %s
GROUP BY recipient_phone
HAVING COUNT(DISTINCT sequence_stepid) > 1
ORDER BY step_count DESC
LIMIT 10
"""

cursor.execute(multi_step_query, (cold_sequence_id, user_id, ui_date))
multi_step = cursor.fetchall()
print(f"\nContacts in multiple steps: {len(multi_step)}")
for phone, step_count in multi_step[:5]:
    print(f"  {phone}: appears in {step_count} steps")

# 3. Get the EXACT count that should match UI
print("\n3. CALCULATING EXACT NUMBERS:")

# Method 1: Count unique combinations (what UI might be doing)
method1_query = """
SELECT 
    COUNT(DISTINCT CONCAT(recipient_phone, '_sent')) as sent_unique,
    COUNT(DISTINCT CONCAT(recipient_phone, '_failed')) as failed_unique,
    COUNT(DISTINCT CONCAT(recipient_phone, '_pending')) as pending_unique
FROM (
    SELECT recipient_phone,
           CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN '_sent'
                WHEN status = 'failed' THEN '_failed'
                WHEN status IN ('pending', 'queued') THEN '_pending'
           END as status_group
    FROM broadcast_messages
    WHERE sequence_id = %s
    AND user_id = %s
    AND DATE(scheduled_at) = %s
) as grouped
WHERE status_group IS NOT NULL
"""

# Method 2: Traditional distinct count per status
method2_query = """
SELECT 
    (SELECT COUNT(DISTINCT recipient_phone) FROM broadcast_messages 
     WHERE sequence_id = %s AND user_id = %s AND DATE(scheduled_at) = %s 
     AND status = 'sent' AND (error_message IS NULL OR error_message = '')) as done,
     
    (SELECT COUNT(DISTINCT recipient_phone) FROM broadcast_messages 
     WHERE sequence_id = %s AND user_id = %s AND DATE(scheduled_at) = %s 
     AND status = 'failed') as failed,
     
    (SELECT COUNT(DISTINCT recipient_phone) FROM broadcast_messages 
     WHERE sequence_id = %s AND user_id = %s AND DATE(scheduled_at) = %s 
     AND status IN ('pending', 'queued')) as remaining
"""

cursor.execute(method2_query, (
    cold_sequence_id, user_id, ui_date,
    cold_sequence_id, user_id, ui_date,
    cold_sequence_id, user_id, ui_date
))
done, failed, remaining = cursor.fetchone()
print(f"\nMethod 2 (separate counts):")
print(f"  Done: {done}")
print(f"  Failed: {failed}")
print(f"  Remaining: {remaining}")
print(f"  Total: {done + failed + remaining}")

# 4. Check if the issue is with the 158 vs 174 done count
print("\n4. INVESTIGATING DONE COUNT DISCREPANCY (UI shows 158, DB shows 174):")
done_by_date_query = """
SELECT DATE(scheduled_at) as date, COUNT(DISTINCT recipient_phone) as done_count
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND status = 'sent' 
AND (error_message IS NULL OR error_message = '')
GROUP BY DATE(scheduled_at)
ORDER BY date DESC
LIMIT 5
"""

cursor.execute(done_by_date_query, (cold_sequence_id, user_id))
done_dates = cursor.fetchall()
print("\nDone messages by date:")
for date, count in done_dates:
    print(f"  {date}: {count} done")

# 5. Final check - get the raw numbers UI is likely using
print("\n5. MOST LIKELY UI CALCULATION:")
# The UI might be filtering by a specific time range or have additional conditions
check_time_query = """
SELECT 
    COUNT(DISTINCT recipient_phone) as total_contacts,
    SUM(CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN 1 ELSE 0 END) as sent_messages,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_messages,
    SUM(CASE WHEN status IN ('pending', 'queued') THEN 1 ELSE 0 END) as pending_messages
FROM broadcast_messages
WHERE sequence_id = %s
AND user_id = %s
AND DATE(scheduled_at) = %s
"""

cursor.execute(check_time_query, (cold_sequence_id, user_id, ui_date))
total_contacts, sent_msgs, failed_msgs, pending_msgs = cursor.fetchone()
print(f"\nMessage counts (not distinct):")
print(f"  Total contacts (distinct): {total_contacts}")
print(f"  Sent messages: {sent_msgs}")
print(f"  Failed messages: {failed_msgs}")
print(f"  Pending messages: {pending_msgs}")

cursor.close()
conn.close()
