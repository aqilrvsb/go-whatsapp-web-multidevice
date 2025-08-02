import pymysql
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
cursor = conn.cursor(pymysql.cursors.DictCursor)

print("=== SEQUENCE DEBUGGING ===")
print(f"Current Time: {datetime.now()}")
print()

# 1. Check active sequences
print("1. ACTIVE SEQUENCES:")
cursor.execute("""
    SELECT id, name, status, is_active, `trigger`, target_status, 
           total_contacts, active_contacts, device_id
    FROM sequences 
    WHERE is_active = 1 AND status = 'active'
""")
sequences = cursor.fetchall()
print(f"Found {len(sequences)} active sequences")
for seq in sequences:
    print(f"  - {seq['name']} (ID: {seq['id']})")
    print(f"    Trigger: {seq['trigger']}, Target: {seq['target_status']}")
    print(f"    Contacts: {seq['active_contacts']}/{seq['total_contacts']}")
    print(f"    Device: {seq['device_id']}")
print()

# 2. Check sequence steps
print("2. SEQUENCE STEPS:")
for seq in sequences[:1]:  # Check first sequence
    cursor.execute("""
        SELECT id, day_number, `trigger`, next_trigger, is_entry_point,
               trigger_delay_hours, min_delay_seconds, max_delay_seconds
        FROM sequence_steps 
        WHERE sequence_id = %s
        ORDER BY day_number
    """, (seq['id'],))
    steps = cursor.fetchall()
    print(f"Steps for '{seq['name']}':")
    for step in steps:
        print(f"  Day {step['day_number']}: {step['trigger']} -> {step['next_trigger']}")
        print(f"    Entry: {step['is_entry_point']}, Delay: {step['trigger_delay_hours']}h")
print()

# 3. Check leads with matching triggers
print("3. LEADS WITH MATCHING TRIGGERS:")
if sequences:
    seq = sequences[0]
    # Get entry point triggers
    cursor.execute("""
        SELECT `trigger` FROM sequence_steps 
        WHERE sequence_id = %s AND is_entry_point = 1
    """, (seq['id'],))
    entry_triggers = [row['trigger'] for row in cursor.fetchall()]
    
    if entry_triggers:
        print(f"Entry triggers: {entry_triggers}")
        for trigger in entry_triggers[:1]:  # Check first trigger
            cursor.execute("""
                SELECT COUNT(*) as count FROM leads 
                WHERE `trigger` LIKE %s
            """, (f'%{trigger}%',))
            count = cursor.fetchone()['count']
            print(f"  Leads with trigger '{trigger}': {count}")
            
            # Show sample leads
            cursor.execute("""
                SELECT id, name, phone, `trigger`, device_id 
                FROM leads 
                WHERE `trigger` LIKE %s
                LIMIT 5
            """, (f'%{trigger}%',))
            sample_leads = cursor.fetchall()
            for lead in sample_leads:
                print(f"    - {lead['name']} ({lead['phone']})")
                print(f"      Triggers: {lead['trigger']}")
                print(f"      Device: {lead['device_id']}")
print()

# 4. Check sequence contacts
print("4. SEQUENCE CONTACTS (Active):")
cursor.execute("""
    SELECT sc.*, s.name as sequence_name
    FROM sequence_contacts sc
    JOIN sequences s ON sc.sequence_id = s.id
    WHERE sc.status = 'active'
    ORDER BY sc.next_trigger_time
    LIMIT 10
""")
contacts = cursor.fetchall()
print(f"Found {len(contacts)} active sequence contacts")
for contact in contacts:
    print(f"  - {contact['contact_name']} ({contact['contact_phone']})")
    print(f"    Sequence: {contact['sequence_name']}")
    print(f"    Current Step: {contact['current_step']}")
    print(f"    Next Trigger: {contact['next_trigger_time']}")
    print(f"    Device: {contact['assigned_device_id']}")
print()

# 5. Check broadcast messages from sequences
print("5. RECENT SEQUENCE MESSAGES IN BROADCAST_MESSAGES:")
cursor.execute("""
    SELECT id, sequence_id, recipient_phone, status, created_at, error_message
    FROM broadcast_messages 
    WHERE sequence_id IS NOT NULL
    ORDER BY created_at DESC
    LIMIT 10
""")
messages = cursor.fetchall()
print(f"Found {len(messages)} sequence messages")
for msg in messages:
    print(f"  - {msg['recipient_phone']} - Status: {msg['status']}")
    print(f"    Created: {msg['created_at']}")
    if msg['error_message']:
        print(f"    Error: {msg['error_message']}")
print()

# 6. Check device status
print("6. DEVICE STATUS:")
cursor.execute("""
    SELECT id, device_name, status, platform
    FROM user_devices
    WHERE status = 'online'
    LIMIT 10
""")
devices = cursor.fetchall()
print(f"Found {len(devices)} online devices")
for device in devices:
    print(f"  - {device['device_name']} (ID: {device['id']})")
    print(f"    Status: {device['status']}, Platform: {device['platform']}")

# 7. Check if sequence trigger service is running properly
print("\n7. CHECKING SEQUENCE TRIGGER TIMING:")
cursor.execute("""
    SELECT sc.id, sc.contact_phone, sc.next_trigger_time, sc.status,
           TIMESTAMPDIFF(MINUTE, sc.next_trigger_time, NOW()) as minutes_overdue
    FROM sequence_contacts sc
    WHERE sc.status = 'active' 
    AND sc.next_trigger_time <= NOW()
    LIMIT 10
""")
overdue = cursor.fetchall()
print(f"Found {len(overdue)} overdue sequence contacts that should have triggered")
for contact in overdue:
    print(f"  - {contact['contact_phone']} - Overdue by {contact['minutes_overdue']} minutes")
    print(f"    Should have triggered at: {contact['next_trigger_time']}")

conn.close()
print("\n=== DEBUG COMPLETE ===")
