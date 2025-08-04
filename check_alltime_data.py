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

print("Checking if UI is showing ALL-TIME data (no date filter)...\n")

# Get statistics without any date filter
query = """
SELECT 
    s.name,
    COUNT(DISTINCT bm.recipient_phone) as total_contacts,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) as done_send,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as failed_send,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) as remaining_send
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id
WHERE bm.sequence_id IS NOT NULL
GROUP BY s.id, s.name
ORDER BY s.name
"""

cursor.execute(query)
results = cursor.fetchall()

print(f"{'Sequence Name':<25} {'Total':<10} {'Done':<10} {'Failed':<10} {'Remaining':<10}")
print("-" * 70)

totals = {'total': 0, 'done': 0, 'failed': 0, 'remaining': 0}

for name, total, done, failed, remaining in results:
    print(f"{name:<25} {total:<10} {done:<10} {failed:<10} {remaining:<10}")
    totals['total'] += total
    totals['done'] += done
    totals['failed'] += failed
    totals['remaining'] += remaining

print("-" * 70)
print(f"{'TOTALS:':<25} {totals['total']:<10} {totals['done']:<10} {totals['failed']:<10} {totals['remaining']:<10}")

print(f"\nDoes this match UI?")
print(f"Total: {totals['total']} vs UI: 2322 - Match: {totals['total'] == 2322}")
print(f"Done: {totals['done']} vs UI: 166 - Match: {totals['done'] == 166}")
print(f"Failed: {totals['failed']} vs UI: 44 - Match: {totals['failed'] == 44}")
print(f"Remaining: {totals['remaining']} vs UI: 2112 - Match: {totals['remaining'] == 2112}")

# Let's also check the exact numbers for each sequence vs what UI shows
print("\n\nComparing with UI data:")
ui_data = {
    'meow': 0,
    'HOT VITAC SEQUENCE': 243,
    'WARM VITAC SEQUENCE': 534,
    'COLD VITAC SEQUENCE': 457,
    'HOT Seqeunce': 429,
    'WARM Sequence': 381,
    'COLD Sequence': 278
}

db_data = {}
cursor.execute(query)
for name, total, done, failed, remaining in cursor.fetchall():
    db_data[name] = total

print(f"\n{'Sequence':<25} {'UI Shows':<10} {'DB Has':<10} {'Difference':<10}")
print("-" * 60)
for seq_name, ui_count in ui_data.items():
    db_count = db_data.get(seq_name, 0)
    diff = ui_count - db_count
    print(f"{seq_name:<25} {ui_count:<10} {db_count:<10} {diff:<10}")

cursor.close()
conn.close()
