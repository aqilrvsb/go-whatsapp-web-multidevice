import pymysql
import pandas as pd
from datetime import datetime, timedelta
import json

# Connect to MySQL
print("Connecting to MySQL...")
connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)
print("Connected to MySQL successfully!")

# 1. Check for sequence messages in broadcast_messages
print("\n=== CHECKING BROADCAST_MESSAGES FOR SEQUENCES ===")
query = """
SELECT 
    id,
    user_id,
    device_id,
    sequence_id,
    sequence_stepid,
    recipient_phone,
    recipient_name,
    content,
    status,
    created_at,
    sent_at,
    error_message
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
ORDER BY created_at DESC
LIMIT 100
"""

df_broadcast = pd.read_sql(query, connection)
print(f"Found {len(df_broadcast)} sequence messages in broadcast_messages")

# 2. Check for duplicates
print("\n=== CHECKING FOR DUPLICATES ===")
duplicate_check = """
SELECT 
    recipient_phone,
    sequence_id,
    sequence_stepid,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(id) as message_ids,
    GROUP_CONCAT(status) as statuses,
    GROUP_CONCAT(created_at) as created_times
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
GROUP BY recipient_phone, sequence_id, sequence_stepid
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC
"""

df_duplicates = pd.read_sql(duplicate_check, connection)
if len(df_duplicates) > 0:
    print(f"WARNING: Found {len(df_duplicates)} duplicate entries!")
    print("\nDuplicate Details:")
    for idx, row in df_duplicates.iterrows():
        print(f"\nPhone: {row['recipient_phone']}, Sequence: {row['sequence_id']}, Step: {row['sequence_stepid']}")
        print(f"  Duplicate Count: {row['duplicate_count']}")
        print(f"  Message IDs: {row['message_ids']}")
        print(f"  Statuses: {row['statuses']}")
else:
    print("✅ No duplicates found!")

# 3. Check sequence flow
print("\n=== CHECKING SEQUENCE FLOW ===")
sequence_flow = """
SELECT 
    sc.sequence_id,
    s.name as sequence_name,
    sc.contact_phone,
    sc.current_step,
    sc.status as contact_status,
    sc.next_trigger_time,
    ss.day_number,
    ss.trigger,
    ss.next_trigger
FROM sequence_contacts sc
JOIN sequences s ON sc.sequence_id = s.id
LEFT JOIN sequence_steps ss ON sc.sequence_stepid = ss.id
WHERE sc.status = 'active'
ORDER BY sc.sequence_id, sc.contact_phone, sc.current_step
LIMIT 50
"""

df_flow = pd.read_sql(sequence_flow, connection)
print(f"Found {len(df_flow)} active sequence contacts")

# 4. Check for wrong sequence order
print("\n=== CHECKING MESSAGE ORDER ===")
order_check = """
SELECT 
    bm.recipient_phone,
    bm.sequence_id,
    ss.day_number,
    bm.created_at,
    bm.status,
    bm.content
FROM broadcast_messages bm
JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
WHERE bm.sequence_id IS NOT NULL
ORDER BY bm.recipient_phone, bm.sequence_id, bm.created_at
"""

df_order = pd.read_sql(order_check, connection)

# Group by phone and sequence to check order
if len(df_order) > 0:
    grouped = df_order.groupby(['recipient_phone', 'sequence_id'])
    wrong_order_count = 0
    
    for (phone, seq_id), group in grouped:
        days = group['day_number'].tolist()
        if days != sorted(days):
            wrong_order_count += 1
            print(f"\n❌ Wrong order for {phone} in sequence {seq_id}")
            print(f"   Day order: {days} (should be: {sorted(days)})")
    
    if wrong_order_count == 0:
        print("✅ All messages are in correct order!")

# 5. Check timing issues
print("\n=== CHECKING TIMING ISSUES ===")
timing_check = """
SELECT 
    sc.contact_phone,
    sc.sequence_id,
    sc.next_trigger_time,
    sc.status,
    COUNT(bm.id) as messages_sent,
    MAX(bm.created_at) as last_message_time
FROM sequence_contacts sc
LEFT JOIN broadcast_messages bm ON 
    sc.contact_phone = bm.recipient_phone AND 
    sc.sequence_id = bm.sequence_id
WHERE sc.status = 'active'
GROUP BY sc.contact_phone, sc.sequence_id
HAVING next_trigger_time < NOW() AND next_trigger_time IS NOT NULL
"""

df_timing = pd.read_sql(timing_check, connection)
if len(df_timing) > 0:
    print(f"⚠️ Found {len(df_timing)} contacts with overdue messages")
    print(df_timing[['contact_phone', 'next_trigger_time', 'messages_sent']].head())

# 6. Summary statistics
print("\n=== SUMMARY STATISTICS ===")
stats_query = """
SELECT 
    COUNT(DISTINCT sequence_id) as total_sequences,
    COUNT(DISTINCT recipient_phone) as total_recipients,
    COUNT(*) as total_messages,
    SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent_messages,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_messages,
    SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_messages
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
"""

stats = pd.read_sql(stats_query, connection)
print(stats.to_string(index=False))

# Export results
print("\n=== EXPORTING RESULTS ===")
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

# Export all sequence messages
df_broadcast.to_csv(f'sequence_messages_{timestamp}.csv', index=False)
print(f"✅ Exported sequence messages to sequence_messages_{timestamp}.csv")

# Export duplicates if any
if len(df_duplicates) > 0:
    df_duplicates.to_csv(f'sequence_duplicates_{timestamp}.csv', index=False)
    print(f"✅ Exported duplicates to sequence_duplicates_{timestamp}.csv")

connection.close()
print("\n✅ Analysis complete!")
