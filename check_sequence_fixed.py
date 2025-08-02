import pymysql
import pandas as pd
from datetime import datetime, timedelta
import json
import sys

# Set UTF-8 encoding for output
sys.stdout.reconfigure(encoding='utf-8')

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
AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
ORDER BY created_at DESC
LIMIT 100
"""

cursor = connection.cursor()
cursor.execute(query)
results = cursor.fetchall()
print(f"Found {len(results)} sequence messages in the last 7 days")

# Show first 5 records
if len(results) > 0:
    print("\nFirst 5 records:")
    for i, record in enumerate(results[:5]):
        print(f"\n--- Record {i+1} ---")
        print(f"ID: {record['id']}")
        print(f"Phone: {record['recipient_phone']}")
        print(f"Sequence: {record['sequence_id']}")
        print(f"Step: {record['sequence_stepid']}")
        print(f"Status: {record['status']}")
        print(f"Created: {record['created_at']}")
        print(f"Content preview: {record['content'][:50]}..." if record['content'] else "No content")

# 2. Check for duplicates
print("\n\n=== CHECKING FOR DUPLICATES ===")
duplicate_check = """
SELECT 
    recipient_phone,
    sequence_id,
    sequence_stepid,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(id SEPARATOR ', ') as message_ids,
    GROUP_CONCAT(status SEPARATOR ', ') as statuses,
    GROUP_CONCAT(DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') SEPARATOR ' | ') as created_times
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY recipient_phone, sequence_id, sequence_stepid
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC
LIMIT 20
"""

cursor.execute(duplicate_check)
duplicates = cursor.fetchall()

if len(duplicates) > 0:
    print(f"WARNING: Found {len(duplicates)} duplicate entries!")
    print("\nTop 10 Duplicate Details:")
    for i, dup in enumerate(duplicates[:10]):
        print(f"\n--- Duplicate {i+1} ---")
        print(f"Phone: {dup['recipient_phone']}")
        print(f"Sequence: {dup['sequence_id']}")
        print(f"Step: {dup['sequence_stepid']}")
        print(f"Duplicate Count: {dup['duplicate_count']}")
        print(f"Message IDs: {dup['message_ids']}")
        print(f"Statuses: {dup['statuses']}")
        print(f"Created Times: {dup['created_times']}")
else:
    print("No duplicates found!")

# 3. Check sequence flow
print("\n\n=== CHECKING SEQUENCE FLOW ===")
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
LIMIT 20
"""

cursor.execute(sequence_flow)
flow_results = cursor.fetchall()
print(f"Found {len(flow_results)} active sequence contacts")

if len(flow_results) > 0:
    print("\nFirst 10 active contacts:")
    for i, contact in enumerate(flow_results[:10]):
        print(f"\n--- Contact {i+1} ---")
        print(f"Phone: {contact['contact_phone']}")
        print(f"Sequence: {contact['sequence_name']} ({contact['sequence_id']})")
        print(f"Current Step: {contact['current_step']}")
        print(f"Day Number: {contact['day_number']}")
        print(f"Status: {contact['contact_status']}")
        print(f"Next Trigger Time: {contact['next_trigger_time']}")
        print(f"Current Trigger: {contact['trigger']}")
        print(f"Next Trigger: {contact['next_trigger']}")

# 4. Check for wrong sequence order
print("\n\n=== CHECKING MESSAGE ORDER ===")
order_check = """
SELECT 
    bm.recipient_phone,
    bm.sequence_id,
    ss.day_number,
    bm.created_at,
    bm.status,
    LEFT(bm.content, 50) as content_preview
FROM broadcast_messages bm
JOIN sequence_steps ss ON bm.sequence_stepid = ss.id
WHERE bm.sequence_id IS NOT NULL
AND bm.created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
ORDER BY bm.recipient_phone, bm.sequence_id, bm.created_at
"""

cursor.execute(order_check)
order_results = cursor.fetchall()

# Group by phone and sequence to check order
if len(order_results) > 0:
    # Convert to DataFrame for easier grouping
    df_order = pd.DataFrame(order_results)
    grouped = df_order.groupby(['recipient_phone', 'sequence_id'])
    
    wrong_order_count = 0
    wrong_order_details = []
    
    for (phone, seq_id), group in grouped:
        days = group['day_number'].tolist()
        if days != sorted(days):
            wrong_order_count += 1
            wrong_order_details.append({
                'phone': phone,
                'sequence_id': seq_id,
                'day_order': days,
                'expected_order': sorted(days)
            })
    
    if wrong_order_count == 0:
        print("All messages are in correct order!")
    else:
        print(f"WARNING: Found {wrong_order_count} contacts with wrong message order!")
        for detail in wrong_order_details[:5]:
            print(f"\nPhone: {detail['phone']}")
            print(f"Sequence: {detail['sequence_id']}")
            print(f"Day order: {detail['day_order']} (should be: {detail['expected_order']})")

# 5. Check timing issues
print("\n\n=== CHECKING TIMING ISSUES ===")
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
AND sc.next_trigger_time IS NOT NULL
GROUP BY sc.contact_phone, sc.sequence_id, sc.next_trigger_time, sc.status
HAVING next_trigger_time < NOW()
LIMIT 10
"""

cursor.execute(timing_check)
timing_results = cursor.fetchall()

if len(timing_results) > 0:
    print(f"WARNING: Found {len(timing_results)} contacts with overdue messages")
    for timing in timing_results[:5]:
        print(f"\nPhone: {timing['contact_phone']}")
        print(f"Next Trigger Time: {timing['next_trigger_time']}")
        print(f"Messages Sent: {timing['messages_sent']}")
        print(f"Last Message: {timing['last_message_time']}")
else:
    print("No timing issues found!")

# 6. Summary statistics
print("\n\n=== SUMMARY STATISTICS ===")
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
AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
"""

cursor.execute(stats_query)
stats = cursor.fetchone()

print(f"Total Sequences: {stats['total_sequences']}")
print(f"Total Recipients: {stats['total_recipients']}")
print(f"Total Messages: {stats['total_messages']}")
print(f"Sent Messages: {stats['sent_messages']}")
print(f"Failed Messages: {stats['failed_messages']}")
print(f"Pending Messages: {stats['pending_messages']}")

# 7. Check sequence step configuration
print("\n\n=== CHECKING SEQUENCE CONFIGURATION ===")
config_check = """
SELECT 
    s.id as sequence_id,
    s.name as sequence_name,
    s.status as sequence_status,
    COUNT(DISTINCT ss.id) as total_steps,
    MIN(ss.day_number) as min_day,
    MAX(ss.day_number) as max_day,
    GROUP_CONCAT(ss.day_number ORDER BY ss.day_number) as day_sequence
FROM sequences s
LEFT JOIN sequence_steps ss ON s.id = ss.sequence_id
WHERE s.status = 'active'
GROUP BY s.id, s.name, s.status
"""

cursor.execute(config_check)
config_results = cursor.fetchall()

print(f"Found {len(config_results)} active sequences")
for seq in config_results[:5]:
    print(f"\nSequence: {seq['sequence_name']} ({seq['sequence_id']})")
    print(f"Status: {seq['sequence_status']}")
    print(f"Total Steps: {seq['total_steps']}")
    print(f"Day Range: {seq['min_day']} to {seq['max_day']}")
    print(f"Day Sequence: {seq['day_sequence']}")

# Export results
print("\n\n=== EXPORTING RESULTS ===")
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

# Prepare export data
export_data = {
    'summary': {
        'analysis_date': datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        'total_sequences': stats['total_sequences'],
        'total_recipients': stats['total_recipients'],
        'total_messages': stats['total_messages'],
        'sent_messages': stats['sent_messages'],
        'failed_messages': stats['failed_messages'],
        'pending_messages': stats['pending_messages'],
        'duplicate_count': len(duplicates),
        'wrong_order_count': wrong_order_count if 'wrong_order_count' in locals() else 0,
        'overdue_contacts': len(timing_results)
    },
    'duplicates': duplicates[:20] if duplicates else [],
    'wrong_order': wrong_order_details if 'wrong_order_details' in locals() else [],
    'overdue_messages': timing_results
}

# Save JSON report
with open(f'sequence_analysis_{timestamp}.json', 'w') as f:
    json.dump(export_data, f, indent=2, default=str)

print(f"Exported analysis to sequence_analysis_{timestamp}.json")

# Close connection
cursor.close()
connection.close()
print("\nAnalysis complete!")

# Print recommendations
print("\n\n=== RECOMMENDATIONS ===")
if len(duplicates) > 0:
    print("1. DUPLICATES FOUND: Review and fix the sequence processing logic to prevent duplicate message creation")
if wrong_order_count > 0:
    print("2. WRONG ORDER: Check sequence step configuration and processing order")
if len(timing_results) > 0:
    print("3. OVERDUE MESSAGES: Check cron job execution and worker processing")
