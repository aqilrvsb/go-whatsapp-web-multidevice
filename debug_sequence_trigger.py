import pymysql
import sys
from datetime import datetime

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

print("=== INVESTIGATING WHY SEQUENCE NOT CREATING MESSAGES ===")
print(f"Time: {datetime.now()}")
print("="*60)

# 1. Check active sequences
print("\n1. CHECKING ACTIVE SEQUENCES:")
active_seq_query = """
SELECT 
    id,
    name,
    status,
    is_active,
    `trigger`,
    start_trigger,
    end_trigger,
    total_days,
    created_at,
    updated_at
FROM sequences
WHERE is_active = 1 OR status = 'active'
"""

cursor.execute(active_seq_query)
active_sequences = cursor.fetchall()

print(f"Found {len(active_sequences)} active sequences")
for seq in active_sequences:
    print(f"\nSequence: {seq['name']}")
    print(f"  ID: {seq['id']}")
    print(f"  Status: {seq['status']}")
    print(f"  Is Active: {seq['is_active']}")
    print(f"  Trigger: {seq['trigger']}")
    print(f"  Start Trigger: {seq['start_trigger']}")
    print(f"  Total Days: {seq['total_days']}")
    print(f"  Updated: {seq['updated_at']}")

# 2. Check sequence steps
if active_sequences:
    print("\n2. CHECKING SEQUENCE STEPS:")
    for seq in active_sequences:
        steps_query = """
        SELECT 
            id,
            day_number,
            is_entry_point,
            trigger,
            next_trigger,
            message_type,
            content,
            media_url
        FROM sequence_steps
        WHERE sequence_id = %s
        ORDER BY day_number
        """
        cursor.execute(steps_query, (seq['id'],))
        steps = cursor.fetchall()
        
        print(f"\nSteps for {seq['name']}:")
        print(f"  Total steps: {len(steps)}")
        for step in steps[:3]:  # Show first 3 steps
            print(f"  - Day {step['day_number']}: Entry={step['is_entry_point']}, Trigger={step['trigger']}")

# 3. Check if there are leads with matching triggers
print("\n3. CHECKING LEADS WITH MATCHING TRIGGERS:")
if active_sequences:
    for seq in active_sequences:
        # Get entry point triggers
        entry_query = """
        SELECT trigger
        FROM sequence_steps
        WHERE sequence_id = %s AND is_entry_point = 1
        """
        cursor.execute(entry_query, (seq['id'],))
        entry_triggers = cursor.fetchall()
        
        for trigger_row in entry_triggers:
            trigger = trigger_row['trigger']
            if trigger:
                # Check leads with this trigger
                lead_query = """
                SELECT COUNT(*) as count
                FROM leads
                WHERE `trigger` LIKE %s
                """
                cursor.execute(lead_query, (f'%{trigger}%',))
                lead_count = cursor.fetchone()
                print(f"\nSequence: {seq['name']}")
                print(f"  Entry trigger: {trigger}")
                print(f"  Leads with trigger: {lead_count['count']}")

# 4. Check if Direct Broadcast Processor ran recently
print("\n4. CHECKING RECENT BROADCAST MESSAGES:")
recent_query = """
SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN sequence_id IS NOT NULL THEN 1 ELSE 0 END) as sequence_msgs,
    MAX(created_at) as latest_created
FROM broadcast_messages
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
"""

cursor.execute(recent_query)
recent = cursor.fetchone()
print(f"Messages created in last hour: {recent['total']}")
print(f"Sequence messages: {recent['sequence_msgs']}")
print(f"Latest message: {recent['latest_created']}")

# 5. Check sequence_contacts table
print("\n5. CHECKING SEQUENCE_CONTACTS:")
contacts_query = """
SELECT 
    sc.sequence_id,
    s.name as sequence_name,
    COUNT(*) as contact_count,
    SUM(CASE WHEN sc.status = 'active' THEN 1 ELSE 0 END) as active_count,
    MAX(sc.created_at) as latest_enrollment
FROM sequence_contacts sc
JOIN sequences s ON sc.sequence_id = s.id
WHERE s.is_active = 1 OR s.status = 'active'
GROUP BY sc.sequence_id, s.name
"""

cursor.execute(contacts_query)
contacts = cursor.fetchall()

if contacts:
    for contact in contacts:
        print(f"\nSequence: {contact['sequence_name']}")
        print(f"  Total contacts: {contact['contact_count']}")
        print(f"  Active contacts: {contact['active_count']}")
        print(f"  Latest enrollment: {contact['latest_enrollment']}")
else:
    print("No contacts found in active sequences")

# 6. Check if there's a filter preventing enrollment
print("\n6. CHECKING ENROLLMENT QUERY:")
print("The Direct Broadcast Processor looks for:")
print("- Active sequences (is_active = true)")
print("- Entry point steps (is_entry_point = true)")
print("- Leads with matching triggers")
print("- Leads that don't already have messages in broadcast_messages")

# Check a sample enrollment query
if active_sequences:
    seq = active_sequences[0]
    check_query = """
    SELECT 
        l.id, l.phone, l.name, l.trigger
    FROM leads l
    WHERE l.trigger IS NOT NULL 
    AND l.trigger != ''
    AND NOT EXISTS (
        SELECT 1 FROM broadcast_messages bm
        WHERE bm.sequence_id = %s 
        AND bm.recipient_phone = l.phone
    )
    LIMIT 5
    """
    cursor.execute(check_query, (seq['id'],))
    eligible_leads = cursor.fetchall()
    
    print(f"\nEligible leads for {seq['name']}:")
    for lead in eligible_leads:
        print(f"  {lead['phone']} - {lead['name']} - Trigger: {lead['trigger']}")

cursor.close()
connection.close()
