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

# Specific date: August 4, 2025
target_date = '2025-08-04'
print(f"Checking REAL MySQL data for: {target_date}")
print("=" * 80)

# Find COLD Sequence EXSTART
cold_seq_query = """
SELECT id, name, user_id 
FROM sequences 
WHERE name = 'COLD Sequence' 
AND niche = 'EXSTART'
"""
cursor.execute(cold_seq_query)
cold_seq = cursor.fetchone()
if cold_seq:
    cold_id, cold_name, user_id = cold_seq
    print(f"\nCOLD Sequence (EXSTART)")
    print(f"ID: {cold_id}")
    print(f"User ID: {user_id}")

    # Get the REAL data for this specific date
    real_query = """
    SELECT 
        COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done,
        COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
        COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS remaining
    FROM broadcast_messages
    WHERE sequence_id = %s
    AND DATE(scheduled_at) = %s
    """
    
    cursor.execute(real_query, (cold_id, target_date))
    done, failed, remaining = cursor.fetchone()
    total = done + failed + remaining
    
    print(f"\nREAL DATABASE VALUES for {target_date}:")
    print(f"  Done: {done}")
    print(f"  Failed: {failed}")
    print(f"  Remaining: {remaining}")
    print(f"  Total Should Send: {total}")
    
    print(f"\nUI SHOWS:")
    print(f"  Done: 217")
    print(f"  Failed: 26")
    print(f"  Remaining: 250")
    print(f"  Total Should Send: 493")
    
    print(f"\nDIFFERENCE:")
    print(f"  Done: UI shows 217, DB has {done} (diff: {217 - done})")
    print(f"  Failed: UI shows 26, DB has {failed} (diff: {26 - failed})")
    print(f"  Remaining: UI shows 250, DB has {remaining} (diff: {250 - remaining})")
    print(f"  Total: UI shows 493, DB has {total} (diff: {493 - total})")

# Check all sequences for this date
print("\n\n" + "=" * 80)
print("ALL SEQUENCES DATA COMPARISON:")
print("=" * 80)

all_seq_query = """
SELECT 
    s.name,
    s.niche,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) AS done,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) AS remaining
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id AND DATE(bm.scheduled_at) = %s
WHERE s.status = 'active'
GROUP BY s.id, s.name, s.niche
ORDER BY s.name
"""

cursor.execute(all_seq_query, (target_date,))
sequences = cursor.fetchall()

print(f"\n{'Sequence Name':<20} {'Niche':<10} {'Done':<8} {'Failed':<8} {'Remain':<8} {'Total':<8} | UI Shows")
print("-" * 85)

ui_data = {
    'HOT VITAC SEQUENCE': {'done': 46, 'failed': 0, 'remaining': 197, 'total': 243},
    'WARM VITAC SEQUENCE': {'done': 179, 'failed': 0, 'remaining': 356, 'total': 535},
    'COLD VITAC SEQUENCE': {'done': 369, 'failed': 20, 'remaining': 68, 'total': 457},
    'HOT Seqeunce': {'done': 17, 'failed': 22, 'remaining': 390, 'total': 429},
    'WARM Sequence': {'done': 347, 'failed': 17, 'remaining': 60, 'total': 424},
    'COLD Sequence': {'done': 217, 'failed': 26, 'remaining': 250, 'total': 493}
}

for name, niche, done, failed, remaining in sequences:
    total = done + failed + remaining
    ui_vals = ui_data.get(name, {'done': 0, 'failed': 0, 'remaining': 0, 'total': 0})
    
    match = "OK" if total == ui_vals['total'] else "WRONG"
    print(f"{name:<20} {niche:<10} {done:<8} {failed:<8} {remaining:<8} {total:<8} | {ui_vals['total']} {match}")
    
    if total != ui_vals['total']:
        print(f"  → DB: {done}+{failed}+{remaining}={total}, UI: {ui_vals['done']}+{ui_vals['failed']}+{ui_vals['remaining']}={ui_vals['total']}")

cursor.close()
conn.close()
