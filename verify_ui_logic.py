import pymysql
from datetime import datetime

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

print(f"Understanding the UI calculation logic for {today}\n")
print("Backend calculates: shouldSend = doneSend + failedSend + remainingSend")
print("This might count some contacts multiple times!\n")

# Check each sequence with the backend's logic
query = """
SELECT 
    s.name,
    COUNT(DISTINCT bm.recipient_phone) AS total_distinct,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) AS done_send,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) AS remaining
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id 
    AND DATE(bm.scheduled_at) = %s
WHERE bm.sequence_id IS NOT NULL
GROUP BY s.id, s.name
ORDER BY s.name
"""

cursor.execute(query, (today,))
results = cursor.fetchall()

print(f"{'Sequence':<25} {'Total':<8} {'Done':<8} {'Failed':<8} {'Remain':<8} {'UI Logic':<10}")
print("-" * 75)

ui_total = 0
actual_total = 0

for name, total, done, failed, remaining in results:
    # Backend's calculation
    ui_should_send = done + failed + remaining
    print(f"{name:<25} {total:<8} {done:<8} {failed:<8} {remaining:<8} {ui_should_send:<10}")
    ui_total += ui_should_send
    actual_total += total

print("-" * 75)
print(f"{'Sum using UI logic:':<25} {' ':<8} {' ':<8} {' ':<8} {' ':<8} {ui_total:<10}")
print(f"{'Actual distinct total:':<25} {actual_total:<8}")

print(f"\nUI shows 2322, our calculation shows {ui_total}")
print(f"Difference: {2322 - ui_total}")

# Check if a contact can have multiple statuses in same sequence
print("\n\nChecking for contacts with multiple statuses in same sequence:")
check_query = """
SELECT 
    recipient_phone,
    sequence_id,
    GROUP_CONCAT(DISTINCT status) as statuses,
    COUNT(*) as msg_count
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND DATE(scheduled_at) = %s
GROUP BY recipient_phone, sequence_id
HAVING COUNT(DISTINCT status) > 1
LIMIT 10
"""

cursor.execute(check_query, (today,))
multi_status = cursor.fetchall()

if multi_status:
    print(f"Found {len(multi_status)} contacts with multiple statuses:")
    for phone, seq_id, statuses, count in multi_status[:5]:
        print(f"  Phone: {phone}, Sequence: {seq_id[:8]}..., Statuses: {statuses}, Messages: {count}")
else:
    print("No contacts found with multiple statuses in same sequence")

# Final check - let's see the actual UI data vs our calculation
print("\n\nFinal comparison:")
ui_sequences = {
    'meow': {'should': 0, 'done': 0, 'failed': 0, 'remaining': 0},
    'HOT VITAC SEQUENCE': {'should': 243, 'done': 46, 'failed': 0, 'remaining': 197},
    'WARM VITAC SEQUENCE': {'should': 534, 'done': 20, 'failed': 0, 'remaining': 514},
    'COLD VITAC SEQUENCE': {'should': 457, 'done': 53, 'failed': 20, 'remaining': 384},
    'HOT Seqeunce': {'should': 429, 'done': 17, 'failed': 22, 'remaining': 390},
    'WARM Sequence': {'should': 381, 'done': 4, 'failed': 0, 'remaining': 377},
    'COLD Sequence': {'should': 278, 'done': 26, 'failed': 2, 'remaining': 250}
}

# Verify UI's internal calculation
for seq, data in ui_sequences.items():
    calc_should = data['done'] + data['failed'] + data['remaining']
    matches = calc_should == data['should']
    print(f"{seq}: {data['done']} + {data['failed']} + {data['remaining']} = {calc_should} (UI shows: {data['should']}) Match: {matches}")

cursor.close()
conn.close()
