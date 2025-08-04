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

# Check if there's something special about those 9 messages
cold_id = '0be82745-8f68-4352-abd0-0b405b43a905'
target_date = '2025-08-04'

print("Investigating the 9 missing 'done' messages...")
print("=" * 60)

# Check all sent messages
sent_query = """
SELECT 
    recipient_phone,
    status,
    error_message,
    scheduled_at,
    sent_at
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
AND status = 'sent'
ORDER BY sent_at DESC
LIMIT 10
"""

cursor.execute(sent_query, (cold_id, target_date))
sent_msgs = cursor.fetchall()

print(f"\nSample of 'sent' messages:")
for phone, status, error, scheduled, sent in sent_msgs:
    error_str = 'NULL' if error is None else f"'{error}'"
    print(f"Phone: {phone}, Error: {error_str}, Sent: {sent}")

# Check if some have error_message not null
error_check = """
SELECT 
    COUNT(DISTINCT recipient_phone) as with_error,
    COUNT(DISTINCT CASE WHEN error_message = '' THEN recipient_phone END) as empty_error,
    COUNT(DISTINCT CASE WHEN error_message IS NULL THEN recipient_phone END) as null_error,
    COUNT(DISTINCT CASE WHEN error_message IS NOT NULL AND error_message != '' THEN recipient_phone END) as has_error
FROM broadcast_messages
WHERE sequence_id = %s
AND DATE(scheduled_at) = %s
AND status = 'sent'
"""

cursor.execute(error_check, (cold_id, target_date))
with_error, empty_error, null_error, has_error = cursor.fetchone()

print(f"\nError message analysis for 'sent' status:")
print(f"  With empty error (''): {empty_error}")
print(f"  With NULL error: {null_error}")
print(f"  With actual error text: {has_error}")
print(f"  Total sent: {empty_error + null_error + has_error}")

# The query uses: status = 'sent' AND (error_message IS NULL OR error_message = '')
# So it should count: empty_error + null_error
print(f"\nQuery should count: {empty_error} + {null_error} = {empty_error + null_error}")
print(f"But UI shows: 217")
print(f"Database has: {empty_error + null_error}")

cursor.close()
conn.close()
