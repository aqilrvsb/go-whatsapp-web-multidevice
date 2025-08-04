import pymysql
import pandas as pd
from datetime import datetime

# Connect to MySQL
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)

print("Connected to MySQL successfully\n")

# Get today's date
today = datetime.now().strftime('%Y-%m-%d')
print(f"Checking data for today: {today}\n")

# 1. Count total sequences
cursor = conn.cursor()
cursor.execute("SELECT COUNT(*) FROM sequences")
total_sequences = cursor.fetchone()[0]
print(f"Total Sequences: {total_sequences}")

# 2. Count total flows (sequence steps)
cursor.execute("SELECT COUNT(*) FROM sequence_steps")
total_flows = cursor.fetchone()[0]
print(f"Total Flows: {total_flows}")

# 3. Get overall statistics for today
query = """
SELECT 
    COUNT(DISTINCT recipient_phone) as total_contacts,
    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') THEN recipient_phone END) as done_send,
    COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) as failed_send,
    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') THEN recipient_phone END) as remaining_send
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND DATE(scheduled_at) = %s
"""

cursor.execute(query, (today,))
result = cursor.fetchone()
total_contacts = result[0]
done_send = result[1]
failed_send = result[2]
remaining_send = result[3]

print(f"\nOverall Statistics for Today ({today}):")
print(f"Total Contacts Should Send: {total_contacts}")
print(f"Contacts Done Send: {done_send}")
print(f"Contacts Failed Send: {failed_send}")
print(f"Contacts Remaining Send: {remaining_send}")
print(f"Calculated Total: {done_send + failed_send + remaining_send}")

# 4. Check individual sequences
print("\n\nDetailed Sequence Statistics:")
print("-" * 100)

# Get all sequences with their stats for today
sequence_query = """
SELECT 
    s.id,
    s.name,
    s.niche,
    s.trigger,
    s.status,
    (SELECT COUNT(*) FROM sequence_steps WHERE sequence_id = s.id) as total_flows,
    COUNT(DISTINCT bm.recipient_phone) as total_contacts,
    COUNT(DISTINCT CASE WHEN bm.status = 'sent' AND (bm.error_message IS NULL OR bm.error_message = '') THEN bm.recipient_phone END) as done_send,
    COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as failed_send,
    COUNT(DISTINCT CASE WHEN bm.status IN ('pending', 'queued') THEN bm.recipient_phone END) as remaining_send
FROM sequences s
LEFT JOIN broadcast_messages bm ON s.id = bm.sequence_id AND DATE(bm.scheduled_at) = %s
GROUP BY s.id, s.name, s.niche, s.trigger, s.status
ORDER BY s.name
"""

cursor.execute(sequence_query, (today,))
sequences = cursor.fetchall()

# Create DataFrame for better display
df = pd.DataFrame(sequences, columns=['ID', 'Name', 'Niche', 'Trigger', 'Status', 'Total Flows', 'Total Contacts', 'Done Send', 'Failed Send', 'Remaining Send'])
print(df.to_string(index=False))

# Calculate totals from individual sequences
print("\n\nTotals from Individual Sequences:")
total_should_send_sum = df['Total Contacts'].sum()
total_done_sum = df['Done Send'].sum()
total_failed_sum = df['Failed Send'].sum()
total_remaining_sum = df['Remaining Send'].sum()

print(f"Sum of Total Contacts: {total_should_send_sum}")
print(f"Sum of Done Send: {total_done_sum}")
print(f"Sum of Failed Send: {total_failed_sum}")
print(f"Sum of Remaining Send: {total_remaining_sum}")

# Check if they match
print("\n\nVerification:")
print(f"Does Total Contacts match? {total_contacts} == {total_should_send_sum}: {total_contacts == total_should_send_sum}")
print(f"Does Done Send match? {done_send} == {total_done_sum}: {done_send == total_done_sum}")
print(f"Does Failed Send match? {failed_send} == {total_failed_sum}: {failed_send == total_failed_sum}")
print(f"Does Remaining Send match? {remaining_send} == {total_remaining_sum}: {remaining_send == total_remaining_sum}")

# Close connection
cursor.close()
conn.close()
