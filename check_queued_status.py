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

# Check if the issue is "queued" vs "pending" status
print("Checking 'queued' vs 'pending' status handling:\n")

query = """
SELECT 
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) AS done,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) AS failed,
    COUNT(DISTINCT CASE WHEN status = 'pending' THEN recipient_phone END) AS pending_only,
    COUNT(DISTINCT CASE WHEN status = 'queued' THEN recipient_phone END) AS queued_only,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) AS pending_and_queued
FROM broadcast_messages
WHERE sequence_id = '0be82745-8f68-4352-abd0-0b405b43a905'
AND DATE(scheduled_at) = '2025-08-04'
"""

cursor.execute(query)
done, failed, pending_only, queued_only, pending_and_queued = cursor.fetchone()

print(f"Done: {done}")
print(f"Failed: {failed}")
print(f"Pending only: {pending_only}")
print(f"Queued only: {queued_only}")
print(f"Pending + Queued: {pending_and_queued}")
print(f"\nTotal should be: {done} + {failed} + {pending_and_queued} = {done + failed + pending_and_queued}")

# Check if maybe the UI is looking at wrong date or has time component
print("\n\nChecking if there's a time component issue:")
time_query = """
SELECT 
    DATE(scheduled_at) as date_only,
    MIN(scheduled_at) as earliest,
    MAX(scheduled_at) as latest,
    COUNT(DISTINCT recipient_phone) as contacts
FROM broadcast_messages
WHERE sequence_id = '0be82745-8f68-4352-abd0-0b405b43a905'
AND DATE(scheduled_at) = '2025-08-04'
GROUP BY DATE(scheduled_at)
"""

cursor.execute(time_query)
result = cursor.fetchone()
if result:
    date_only, earliest, latest, contacts = result
    print(f"Date: {date_only}")
    print(f"Earliest scheduled_at: {earliest}")
    print(f"Latest scheduled_at: {latest}")
    print(f"Total contacts: {contacts}")

cursor.close()
conn.close()
