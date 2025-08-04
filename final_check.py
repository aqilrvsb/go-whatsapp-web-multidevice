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
today = datetime.now().strftime('%Y-%m-%d')

print(f"Final verification for {today}\n")

# Check the exact numbers the UI shows vs what we calculate
ui_data = {
    'meow': {'should': 0, 'done': 0, 'failed': 0, 'remaining': 0},
    'HOT VITAC SEQUENCE': {'should': 243, 'done': 46, 'failed': 0, 'remaining': 197},
    'WARM VITAC SEQUENCE': {'should': 534, 'done': 20, 'failed': 0, 'remaining': 514},
    'COLD VITAC SEQUENCE': {'should': 457, 'done': 53, 'failed': 20, 'remaining': 384},
    'HOT Seqeunce': {'should': 429, 'done': 17, 'failed': 22, 'remaining': 390},
    'WARM Sequence': {'should': 381, 'done': 4, 'failed': 0, 'remaining': 377},
    'COLD Sequence': {'should': 278, 'done': 26, 'failed': 2, 'remaining': 250}
}

print("UI Shows:")
ui_total_should = sum(seq['should'] for seq in ui_data.values())
ui_total_done = sum(seq['done'] for seq in ui_data.values())
ui_total_failed = sum(seq['failed'] for seq in ui_data.values())
ui_total_remaining = sum(seq['remaining'] for seq in ui_data.values())

print(f"Total Should Send: {ui_total_should}")
print(f"Total Done: {ui_total_done}")
print(f"Total Failed: {ui_total_failed}")
print(f"Total Remaining: {ui_total_remaining}")

# Now check what the database actually has, including ALL statuses
print("\n\nDatabase Check (including all statuses):")
query = """
SELECT 
    s.name,
    COUNT(DISTINCT bm.recipient_phone) as total_contacts,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) as done_send,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as failed_send,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) as remaining_send,
    COUNT(DISTINCT CASE WHEN bm.status NOT IN ('sent', 'failed', 'pending', 'queued') THEN bm.recipient_phone END) as other_status
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id 
    AND DATE(bm.scheduled_at) = %s
GROUP BY s.id, s.name
ORDER BY s.name
"""

cursor.execute(query, (today,))
results = cursor.fetchall()

db_totals = {'total': 0, 'done': 0, 'failed': 0, 'remaining': 0, 'other': 0}

print(f"{'Sequence Name':<25} {'Total':<8} {'Done':<8} {'Failed':<8} {'Remaining':<10} {'Other':<8}")
print("-" * 80)

for name, total, done, failed, remaining, other in results:
    print(f"{name:<25} {total:<8} {done:<8} {failed:<8} {remaining:<10} {other:<8}")
    db_totals['total'] += total
    db_totals['done'] += done
    db_totals['failed'] += failed
    db_totals['remaining'] += remaining
    db_totals['other'] += other

print("-" * 80)
print(f"{'Database Totals:':<25} {db_totals['total']:<8} {db_totals['done']:<8} {db_totals['failed']:<8} {db_totals['remaining']:<10} {db_totals['other']:<8}")

# Check if date filter is the issue - maybe UI is not filtering by today?
print("\n\nChecking without date filter:")
query_no_date = """
SELECT 
    s.name,
    COUNT(DISTINCT bm.recipient_phone) as total_contacts
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id
WHERE bm.sequence_id IS NOT NULL
GROUP BY s.id, s.name
ORDER BY s.name
"""

cursor.execute(query_no_date)
results_no_date = cursor.fetchall()

print(f"\n{'Sequence Name':<25} {'Total (All Time)':<15}")
print("-" * 50)
total_all_time = 0
for name, total in results_no_date:
    print(f"{name:<25} {total:<15}")
    total_all_time += total
print("-" * 50)
print(f"{'Total All Time:':<25} {total_all_time:<15}")

cursor.close()
conn.close()
