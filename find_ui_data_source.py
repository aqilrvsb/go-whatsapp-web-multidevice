import pymysql
from datetime import datetime, timedelta

conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)

cursor = conn.cursor()

# Check different dates to find where the UI numbers come from
print("Checking broadcast_messages by date to match UI numbers...\n")

# Check last 7 days
for i in range(7):
    check_date = (datetime.now() - timedelta(days=i)).strftime('%Y-%m-%d')
    
    query = """
    SELECT 
        COUNT(DISTINCT recipient_phone) as unique_total,
        SUM(counts.per_seq_total) as sum_total,
        SUM(counts.per_seq_done) as sum_done,
        SUM(counts.per_seq_failed) as sum_failed,
        SUM(counts.per_seq_remaining) as sum_remaining
    FROM (
        SELECT 
            s.id,
            COUNT(DISTINCT bm.recipient_phone) as per_seq_total,
            COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) as per_seq_done,
            COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as per_seq_failed,
            COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) as per_seq_remaining
        FROM sequences s
        LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id 
            AND DATE(bm.scheduled_at) = %s
        WHERE bm.sequence_id IS NOT NULL
        GROUP BY s.id
    ) as counts
    """
    
    cursor.execute(query, (check_date,))
    result = cursor.fetchone()
    
    if result and result[1] > 0:
        print(f"Date: {check_date}")
        print(f"  Sum Total (as UI calculates): {result[1]}")
        print(f"  Sum Done: {result[2]}")
        print(f"  Sum Failed: {result[3]}")
        print(f"  Sum Remaining: {result[4]}")
        print(f"  Match UI total? {result[1] == 2322}")
        print()

# Also check if scheduled_at could be NULL or have time zone issues
print("\nChecking for NULL scheduled_at or timezone issues:")
query2 = """
SELECT 
    COUNT(*) as null_scheduled,
    MIN(scheduled_at) as earliest,
    MAX(scheduled_at) as latest
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND scheduled_at IS NULL
"""
cursor.execute(query2)
null_result = cursor.fetchone()
print(f"NULL scheduled_at count: {null_result[0]}")

# Check created_at as alternative
print("\nChecking if UI might be using created_at instead of scheduled_at:")
today = datetime.now().strftime('%Y-%m-%d')
query3 = """
SELECT 
    SUM(counts.per_seq_total) as sum_total
FROM (
    SELECT 
        s.id,
        COUNT(DISTINCT bm.recipient_phone) as per_seq_total
    FROM sequences s
    LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id 
        AND DATE(bm.created_at) = %s
    WHERE bm.sequence_id IS NOT NULL
    GROUP BY s.id
) as counts
"""
cursor.execute(query3, (today,))
created_result = cursor.fetchone()
print(f"Total using created_at for today: {created_result[0]}")

# One more check - maybe the 175 difference is from a specific sequence
print("\n\nChecking each sequence for today with both created_at and scheduled_at:")
query4 = """
SELECT 
    s.name,
    COUNT(DISTINCT CASE WHEN DATE(bm.scheduled_at) = %s THEN bm.recipient_phone END) as scheduled_today,
    COUNT(DISTINCT CASE WHEN DATE(bm.created_at) = %s THEN bm.recipient_phone END) as created_today,
    COUNT(DISTINCT bm.recipient_phone) as total_all_time
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id
WHERE bm.sequence_id IS NOT NULL
GROUP BY s.id, s.name
ORDER BY s.name
"""
cursor.execute(query4, (today, today))
results = cursor.fetchall()

print(f"{'Sequence':<25} {'Scheduled Today':<15} {'Created Today':<15} {'All Time':<10}")
print("-" * 70)
for name, sched, created, total in results:
    print(f"{name:<25} {sched:<15} {created:<15} {total:<10}")

cursor.close()
conn.close()
