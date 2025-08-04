import pymysql
from datetime import datetime

# Connect to MySQL
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

# Get COLD Sequence ID
cursor.execute("SELECT id FROM sequences WHERE name = 'COLD Sequence'")
sequence_id = cursor.fetchone()[0]
print(f"COLD Sequence ID: {sequence_id}")

# Check overall statistics for Aug 3, 2025
print("\n=== OVERALL STATISTICS (Aug 3, 2025) ===")
cursor.execute("""
    SELECT 
        COUNT(DISTINCT CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id)) as total_messages,
        COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') 
            THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as done_send,
        COUNT(DISTINCT CASE WHEN status = 'failed' 
            THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as failed_send,
        COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') 
            THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as remaining_send,
        COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
    FROM broadcast_messages
    WHERE sequence_id = %s 
    AND DATE(scheduled_at) = '2025-08-03'
""", (sequence_id,))

result = cursor.fetchone()
total_should = result[1] + result[2] + result[3]
print(f"Total Should Send: {total_should}")
print(f"Done Send: {result[1]}")
print(f"Failed Send: {result[2]}")
print(f"Remaining: {result[3]}")
print(f"Total Leads: {result[4]}")

# Check per step statistics
print("\n=== STEP-WISE STATISTICS ===")
cursor.execute("""
    SELECT 
        sequence_stepid,
        COUNT(DISTINCT CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id)) as total_messages,
        COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') 
            THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as done_send,
        COUNT(DISTINCT CASE WHEN status = 'failed' 
            THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as failed_send,
        COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads_per_step
    FROM broadcast_messages
    WHERE sequence_id = %s 
    AND DATE(scheduled_at) = '2025-08-03'
    GROUP BY sequence_stepid
    ORDER BY sequence_stepid
""", (sequence_id,))

steps = cursor.fetchall()
for i, step in enumerate(steps, 1):
    step_id, total, done, failed, leads = step
    should_send = done + failed
    print(f"\nStep {i} (ID: {step_id[:8]}...):")
    print(f"  Should Send: {should_send}")
    print(f"  Done Send: {done}")
    print(f"  Failed Send: {failed}")
    print(f"  Total Leads: {leads}")

# Check unique devices
print("\n=== DEVICE STATISTICS ===")
cursor.execute("""
    SELECT COUNT(DISTINCT device_id) as unique_devices
    FROM broadcast_messages
    WHERE sequence_id = %s 
    AND DATE(scheduled_at) = '2025-08-03'
""", (sequence_id,))
print(f"Unique Devices: {cursor.fetchone()[0]}")

conn.close()