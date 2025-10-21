import subprocess
import sys

# Try to install pymysql if not available
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
print("CHECKING DEVICE: 9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107")
print("="*80)

# 1. Check the overall status of messages for this device
print("\n1. MESSAGE STATUS SUMMARY:")
print("-" * 50)
cursor.execute("""
    SELECT 
        status, 
        COUNT(*) as count,
        MIN(scheduled_at) as earliest,
        MAX(scheduled_at) as latest
    FROM broadcast_messages 
    WHERE device_id = '9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107'
    GROUP BY status
""")
results = cursor.fetchall()
print(f"{'Status':<15} {'Count':<10} {'Earliest':<25} {'Latest':<25}")
for row in results:
    print(f"{row[0]:<15} {row[1]:<10} {row[2]!s:<25} {row[3]!s:<25}")

# 2. Check pending messages specifically
print("\n2. PENDING MESSAGES TIME ANALYSIS:")
print("-" * 50)
cursor.execute("""
    SELECT 
        COUNT(*) as total_pending,
        SUM(CASE WHEN scheduled_at IS NULL THEN 1 ELSE 0 END) as null_scheduled,
        SUM(CASE WHEN processing_worker_id IS NOT NULL THEN 1 ELSE 0 END) as has_worker_id
    FROM broadcast_messages 
    WHERE device_id = '9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107'
    AND status = 'pending'
""")
result = cursor.fetchone()
print(f"Total Pending: {result[0]}")
print(f"With NULL scheduled_at: {result[1]}")
print(f"Has processing_worker_id: {result[2]}")

# 3. Check the scheduled_at distribution for pending messages
print("\n3. PENDING MESSAGES BY DATE:")
print("-" * 50)
cursor.execute("""
    SELECT 
        DATE(scheduled_at) as scheduled_date,
        COUNT(*) as count,
        MIN(TIME(scheduled_at)) as earliest_time,
        MAX(TIME(scheduled_at)) as latest_time
    FROM broadcast_messages 
    WHERE device_id = '9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107'
    AND status = 'pending'
    AND scheduled_at IS NOT NULL
    GROUP BY DATE(scheduled_at)
    ORDER BY scheduled_date DESC
    LIMIT 10
""")
results = cursor.fetchall()
print(f"{'Date':<15} {'Count':<8} {'Earliest':<10} {'Latest':<10}")
for row in results:
    print(f"{row[0]!s:<15} {row[1]:<8} {row[2]!s:<10} {row[3]!s:<10}")

# 4. Check current server time and Malaysia time
print("\n4. TIME CHECK:")
print("-" * 50)
cursor.execute("""
    SELECT 
        NOW() as server_now,
        DATE_ADD(NOW(), INTERVAL 8 HOUR) as malaysia_now
""")
result = cursor.fetchone()
print(f"Server Time (UTC): {result[0]}")
print(f"Malaysia Time:     {result[1]}")

# 5. Check the time window issue
print("\n5. TIME WINDOW ANALYSIS:")
print("-" * 50)
cursor.execute("""
    SELECT 
        COUNT(*) as total_pending,
        SUM(CASE 
            WHEN scheduled_at <= DATE_ADD(NOW(), INTERVAL 8 HOUR) 
            THEN 1 ELSE 0 
        END) as not_future,
        SUM(CASE 
            WHEN scheduled_at >= DATE_ADD(DATE_SUB(NOW(), INTERVAL 1 HOUR), INTERVAL 8 HOUR)
            THEN 1 ELSE 0 
        END) as within_1hour,
        SUM(CASE 
            WHEN scheduled_at >= DATE_ADD(DATE_SUB(NOW(), INTERVAL 1 DAY), INTERVAL 8 HOUR)
            THEN 1 ELSE 0 
        END) as within_1day,
        SUM(CASE 
            WHEN scheduled_at >= DATE_ADD(DATE_SUB(NOW(), INTERVAL 7 DAY), INTERVAL 8 HOUR)
            THEN 1 ELSE 0 
        END) as within_7days
    FROM broadcast_messages 
    WHERE device_id = '9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107'
    AND status = 'pending'
    AND processing_worker_id IS NULL
    AND scheduled_at IS NOT NULL
""")
result = cursor.fetchone()
print(f"Total Pending (eligible): {result[0]}")
print(f"Not in future: {result[1]}")
print(f"Within 1 hour window: {result[2]}")
print(f"Within 1 day window: {result[3]}")
print(f"Within 7 days window: {result[4]}")

# 6. Show some sample pending messages
print("\n6. SAMPLE PENDING MESSAGES:")
print("-" * 50)
cursor.execute("""
    SELECT 
        id,
        recipient_phone,
        scheduled_at,
        processing_worker_id,
        campaign_id,
        sequence_id
    FROM broadcast_messages 
    WHERE device_id = '9ca1bd3d-fc79-48ba-b6b1-1fe8d8ec0107'
    AND status = 'pending'
    ORDER BY scheduled_at ASC
    LIMIT 5
""")
results = cursor.fetchall()
print(f"{'ID':<40} {'Phone':<15} {'Scheduled At':<25} {'Worker':<10} {'Campaign':<10} {'Sequence':<10}")
for row in results:
    print(f"{row[0]:<40} {row[1]:<15} {row[2]!s:<25} {row[3] or 'NULL':<10} {row[4] or 'NULL':<10} {row[5] or 'NULL':<10}")

cursor.close()
conn.close()

print("\n" + "="*80)
print("ANALYSIS COMPLETE")
print("="*80)
