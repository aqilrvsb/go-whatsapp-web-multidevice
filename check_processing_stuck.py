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
print("CHECKING STUCK 'PROCESSING' MESSAGES")
print("="*80)

# 1. Check all processing messages
print("\n1. ALL PROCESSING MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        device_id,
        COUNT(*) as count,
        MIN(processing_started_at) as earliest_start,
        MAX(processing_started_at) as latest_start,
        COUNT(DISTINCT processing_worker_id) as unique_workers
    FROM broadcast_messages 
    WHERE status = 'processing'
    GROUP BY device_id
    ORDER BY count DESC
""")
results = cursor.fetchall()
if results:
    print(f"{'Device ID':<40} {'Count':<8} {'Earliest Start':<25} {'Latest Start':<25} {'Workers':<8}")
    for row in results:
        print(f"{row[0]:<40} {row[1]:<8} {row[2]!s:<25} {row[3]!s:<25} {row[4]:<8}")
else:
    print("No messages currently in 'processing' status")

# 2. Check how old these processing messages are
print("\n2. STUCK PROCESSING MESSAGES (older than 5 minutes):")
print("-" * 50)
cursor.execute("""
    SELECT 
        device_id,
        COUNT(*) as stuck_count,
        MIN(processing_started_at) as oldest,
        MAX(processing_started_at) as newest
    FROM broadcast_messages 
    WHERE status = 'processing'
    AND processing_started_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 5 MINUTE)
    GROUP BY device_id
""")
results = cursor.fetchall()
if results:
    print(f"{'Device ID':<40} {'Stuck':<8} {'Oldest':<25} {'Newest':<25}")
    for row in results:
        print(f"{row[0]:<40} {row[1]:<8} {row[2]!s:<25} {row[3]!s:<25}")
else:
    print("No stuck processing messages (older than 5 minutes)")

# 3. Sample of processing messages
print("\n3. SAMPLE PROCESSING MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        id,
        device_id,
        recipient_phone,
        status,
        processing_started_at,
        processing_worker_id,
        scheduled_at,
        TIMESTAMPDIFF(MINUTE, processing_started_at, DATE_ADD(NOW(), INTERVAL 8 HOUR)) as minutes_stuck
    FROM broadcast_messages 
    WHERE status = 'processing'
    ORDER BY processing_started_at ASC
    LIMIT 10
""")
results = cursor.fetchall()
if results:
    print(f"{'ID':<40} {'Device':<25} {'Phone':<15} {'Started At':<25} {'Mins Stuck':<10}")
    for row in results:
        device_short = row[1][:23] if row[1] else 'NULL'
        print(f"{row[0]:<40} {device_short:<25} {row[2]:<15} {row[4]!s:<25} {row[7]:<10}")
        print(f"  Worker ID: {row[5] if row[5] else 'NULL'}")
else:
    print("No messages in processing status")

# 4. Check processing messages by date
print("\n4. PROCESSING MESSAGES BY DATE:")
print("-" * 50)
cursor.execute("""
    SELECT 
        DATE(processing_started_at) as date,
        COUNT(*) as count,
        COUNT(DISTINCT device_id) as devices,
        COUNT(DISTINCT processing_worker_id) as workers
    FROM broadcast_messages 
    WHERE status = 'processing'
    GROUP BY DATE(processing_started_at)
    ORDER BY date DESC
""")
results = cursor.fetchall()
if results:
    print(f"{'Date':<15} {'Count':<10} {'Devices':<10} {'Workers':<10}")
    for row in results:
        print(f"{row[0]!s:<15} {row[1]:<10} {row[2]:<10} {row[3]:<10}")
else:
    print("No processing messages found")

# 5. Check for processing messages without worker ID (potential issue)
print("\n5. PROCESSING MESSAGES WITHOUT WORKER ID (PROBLEM!):")
print("-" * 50)
cursor.execute("""
    SELECT 
        COUNT(*) as count,
        MIN(processing_started_at) as earliest,
        MAX(processing_started_at) as latest
    FROM broadcast_messages 
    WHERE status = 'processing'
    AND processing_worker_id IS NULL
""")
result = cursor.fetchone()
if result[0] > 0:
    print(f"Found {result[0]} processing messages WITHOUT worker ID!")
    print(f"Earliest: {result[1]}")
    print(f"Latest: {result[2]}")
    print("These messages are likely stuck!")
else:
    print("All processing messages have worker IDs (good)")

# 6. Current time check
print("\n6. TIME CHECK:")
print("-" * 50)
cursor.execute("""
    SELECT 
        NOW() as server_now,
        DATE_ADD(NOW(), INTERVAL 8 HOUR) as malaysia_now
""")
result = cursor.fetchone()
print(f"Server Time (UTC): {result[0]}")
print(f"Malaysia Time:     {result[1]}")

# 7. Recommendation
print("\n7. RECOMMENDED FIX FOR STUCK MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT COUNT(*) 
    FROM broadcast_messages 
    WHERE status = 'processing'
    AND processing_started_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 30 MINUTE)
""")
stuck_count = cursor.fetchone()[0]
if stuck_count > 0:
    print(f"Found {stuck_count} messages stuck in processing for >30 minutes")
    print("\nTo reset these stuck messages, run this SQL:")
    print("UPDATE broadcast_messages")
    print("SET status = 'pending', processing_worker_id = NULL, processing_started_at = NULL")
    print("WHERE status = 'processing'")
    print("AND processing_started_at < DATE_SUB(DATE_ADD(NOW(), INTERVAL 8 HOUR), INTERVAL 30 MINUTE);")
else:
    print("No messages stuck for more than 30 minutes")

cursor.close()
conn.close()

print("\n" + "="*80)
print("ANALYSIS COMPLETE")
print("="*80)
