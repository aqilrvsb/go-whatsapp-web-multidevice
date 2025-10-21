import subprocess
import sys

# Try to import pymysql
try:
    import pymysql
except ImportError:
    print("Installing pymysql...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "pymysql"])
    import pymysql

from datetime import datetime, timedelta

# Database connection
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)

cursor = conn.cursor()

print("\n" + "="*80)
print("SEQUENCE SUMMARY ANALYSIS - September 24, 2025")
print("="*80)

# 1. Check current time
print("\n1. CURRENT TIME CHECK:")
print("-" * 50)
cursor.execute("""
    SELECT 
        NOW() as server_utc,
        DATE_ADD(NOW(), INTERVAL 8 HOUR) as malaysia_time
""")
result = cursor.fetchone()
print(f"Server Time (UTC): {result[0]}")
print(f"Malaysia Time:     {result[1]}")

# 2. Check sequence messages status
print("\n2. SEQUENCE MESSAGES STATUS:")
print("-" * 50)
cursor.execute("""
    SELECT 
        status,
        COUNT(*) as count,
        MIN(scheduled_at) as earliest,
        MAX(scheduled_at) as latest
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    GROUP BY status
    ORDER BY FIELD(status, 'pending', 'processing', 'queued', 'sent', 'failed', 'skipped')
""")
results = cursor.fetchall()
print(f"{'Status':<15} {'Count':<10} {'Earliest Scheduled':<25} {'Latest Scheduled':<25}")
for row in results:
    print(f"{row[0]:<15} {row[1]:<10} {row[2]!s:<25} {row[3]!s:<25}")

# 3. Check pending sequence messages by date
print("\n3. PENDING SEQUENCE MESSAGES BY DATE:")
print("-" * 50)
cursor.execute("""
    SELECT 
        DATE(scheduled_at) as date,
        COUNT(*) as count,
        COUNT(DISTINCT sequence_id) as sequences,
        COUNT(DISTINCT recipient_phone) as unique_contacts
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    AND status = 'pending'
    GROUP BY DATE(scheduled_at)
    ORDER BY date ASC
    LIMIT 10
""")
results = cursor.fetchall()
if results:
    print(f"{'Date':<15} {'Messages':<10} {'Sequences':<10} {'Contacts':<10}")
    for row in results:
        print(f"{row[0]!s:<15} {row[1]:<10} {row[2]:<10} {row[3]:<10}")
else:
    print("No pending sequence messages found")

# 4. Check why messages might not be processing
print("\n4. PENDING MESSAGES TIME ANALYSIS:")
print("-" * 50)
cursor.execute("""
    SELECT 
        COUNT(*) as total_pending,
        SUM(CASE 
            WHEN scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR) 
            THEN 1 ELSE 0 
        END) as should_process_now,
        SUM(CASE 
            WHEN scheduled_at > DATE_ADD(NOW(), INTERVAL 8 HOUR) 
            THEN 1 ELSE 0 
        END) as future_scheduled,
        SUM(CASE 
            WHEN scheduled_at < DATE_ADD(DATE_SUB(NOW(), INTERVAL 1 HOUR), INTERVAL 8 HOUR)
            THEN 1 ELSE 0 
        END) as outside_time_window
    FROM broadcast_messages
    WHERE sequence_id IS NOT NULL
    AND status = 'pending'
""")
result = cursor.fetchone()
print(f"Total Pending: {result[0]}")
print(f"Should Process Now: {result[1]}")
print(f"Future Scheduled: {result[2]}")
print(f"Outside 1-Hour Time Window: {result[3]} ⚠️")

# 5. Check sequence contacts status
print("\n5. SEQUENCE CONTACTS STATUS:")
print("-" * 50)
cursor.execute("""
    SELECT 
        status,
        COUNT(*) as count,
        COUNT(DISTINCT sequence_id) as sequences
    FROM sequence_contacts
    GROUP BY status
""")
results = cursor.fetchall()
print(f"{'Status':<15} {'Contacts':<10} {'Sequences':<10}")
for row in results:
    print(f"{row[0] or 'NULL':<15} {row[1]:<10} {row[2]:<10}")

# 6. Check specific sequences with pending messages
print("\n6. SEQUENCES WITH PENDING MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        s.name,
        bm.sequence_id,
        COUNT(*) as pending_count,
        MIN(bm.scheduled_at) as earliest,
        MAX(bm.scheduled_at) as latest
    FROM broadcast_messages bm
    LEFT JOIN sequences s ON s.id = bm.sequence_id
    WHERE bm.status = 'pending'
    AND bm.sequence_id IS NOT NULL
    GROUP BY bm.sequence_id, s.name
    ORDER BY pending_count DESC
    LIMIT 5
""")
results = cursor.fetchall()
if results:
    print(f"{'Sequence Name':<30} {'ID':<40} {'Pending':<10} {'Earliest':<20}")
    for row in results:
        name = (row[0] or 'Unknown')[:29]
        seq_id = row[1][:38] if row[1] else 'NULL'
        print(f"{name:<30} {seq_id:<40} {row[2]:<10} {row[3]!s:<20}")

# 7. Check for stuck processing messages
print("\n7. STUCK PROCESSING MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        COUNT(*) as stuck_count,
        MIN(processing_started_at) as oldest,
        MAX(processing_started_at) as newest
    FROM broadcast_messages
    WHERE status = 'processing'
    AND sequence_id IS NOT NULL
""")
result = cursor.fetchone()
if result[0] > 0:
    print(f"Stuck in Processing: {result[0]} messages")
    print(f"Oldest: {result[1]}")
    print(f"Newest: {result[2]}")
else:
    print("No stuck processing messages")

# 8. Sample of pending messages
print("\n8. SAMPLE PENDING SEQUENCE MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        recipient_phone,
        scheduled_at,
        TIMESTAMPDIFF(HOUR, DATE_ADD(NOW(), INTERVAL 8 HOUR), scheduled_at) as hours_until,
        sequence_stepid
    FROM broadcast_messages
    WHERE status = 'pending'
    AND sequence_id IS NOT NULL
    ORDER BY scheduled_at ASC
    LIMIT 5
""")
results = cursor.fetchall()
if results:
    print(f"{'Phone':<15} {'Scheduled At':<25} {'Hours Until':<12} {'Step ID':<10}")
    for row in results:
        hours = f"{row[2]:+d}h" if row[2] is not None else "NOW"
        step_id = str(row[3])[:8] if row[3] else "NULL"
        print(f"{row[0]:<15} {row[1]!s:<25} {hours:<12} {step_id:<10}")

cursor.close()
conn.close()

print("\n" + "="*80)
print("ANALYSIS COMPLETE")
print("="*80)
