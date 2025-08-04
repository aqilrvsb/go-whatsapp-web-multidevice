import pymysql
import os
import sys

# Set UTF-8 encoding for Windows
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')

# Get MySQL connection from environment
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
if mysql_uri.startswith('mysql://'):
    mysql_uri = mysql_uri[8:]
    
parts = mysql_uri.split('@')
user_pass = parts[0].split(':')
host_db = parts[1].split('/')

user = user_pass[0]
password = user_pass[1]
host_port = host_db[0].split(':')
host = host_port[0]
port = int(host_port[1]) if len(host_port) > 1 else 3306
database = host_db[1].split('?')[0]

try:
    # Connect to MySQL
    connection = pymysql.connect(
        host=host,
        port=port,
        user=user,
        password=password,
        database=database,
        cursorclass=pymysql.cursors.DictCursor
    )
    
    print("Connected to MySQL database")
    print("=" * 100)
    print("\nINVESTIGATING SEQUENCE COUNT MISMATCH - COLD Sequence EXSTART")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Find the COLD Sequence for EXSTART niche
        cursor.execute("""
            SELECT 
                s.id,
                s.name,
                s.niche,
                s.status,
                s.trigger,
                COUNT(DISTINCT ss.id) as step_count
            FROM sequences s
            LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
            WHERE s.name = 'COLD Sequence'
            AND s.niche = 'EXSTART'
            GROUP BY s.id
        """)
        
        sequence = cursor.fetchone()
        
        if not sequence:
            print("❌ COLD Sequence for EXSTART not found!")
            exit()
        
        print(f"\nSequence found:")
        print(f"  ID: {sequence['id']}")
        print(f"  Name: {sequence['name']}")
        print(f"  Niche: {sequence['niche']}")
        print(f"  Status: {sequence['status']}")
        print(f"  Trigger: {sequence['trigger']}")
        print(f"  Steps: {sequence['step_count']}")
        
        sequence_id = sequence['id']
        
        # Get sequence contacts (the 278 leads)
        print("\n\n1. CHECKING SEQUENCE CONTACTS (leads enrolled in sequence):")
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT contact_phone) as total_leads,
                COUNT(*) as total_entries,
                COUNT(DISTINCT assigned_device_id) as devices_assigned,
                SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count,
                SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count
            FROM sequence_contacts
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        contacts_result = cursor.fetchone()
        print(f"   Total unique phones in sequence_contacts: {contacts_result['total_leads']}")
        print(f"   Total entries: {contacts_result['total_entries']}")
        print(f"   Devices assigned: {contacts_result['devices_assigned']}")
        print(f"   Active: {contacts_result['active_count']}")
        print(f"   Completed: {contacts_result['completed_count']}")
        
        # Get broadcast messages stats
        print("\n\n2. CHECKING BROADCAST MESSAGES (actual messages to be sent):")
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT recipient_phone) as unique_recipients,
                COUNT(*) as total_messages,
                COUNT(DISTINCT sequence_stepid) as steps_with_messages,
                SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent_count,
                SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
                SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_count
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        broadcast_stats = cursor.fetchone()
        print(f"   Unique recipients: {broadcast_stats['unique_recipients']}")
        print(f"   Total messages: {broadcast_stats['total_messages']}")
        print(f"   Steps with messages: {broadcast_stats['steps_with_messages']}")
        print(f"   Sent: {broadcast_stats['sent_count']}")
        print(f"   Failed: {broadcast_stats['failed_count']}")
        print(f"   Pending: {broadcast_stats['pending_count']}")
        
        # Check why the difference
        print("\n\n3. CHECKING WHY THE DIFFERENCE:")
        
        # Get leads that are in sequence_contacts but NOT in broadcast_messages
        cursor.execute("""
            SELECT 
                sc.contact_phone,
                sc.contact_name,
                sc.created_at as enrolled_at,
                sc.status as sc_status,
                sc.assigned_device_id
            FROM sequence_contacts sc
            WHERE sc.sequence_id = %s
            AND sc.contact_phone NOT IN (
                SELECT DISTINCT recipient_phone 
                FROM broadcast_messages 
                WHERE sequence_id = %s
            )
            LIMIT 10
        """, (sequence_id, sequence_id))
        
        missing_leads = cursor.fetchall()
        
        if missing_leads:
            print(f"\n   Found leads in sequence_contacts but NOT in broadcast_messages:")
            for lead in missing_leads:
                print(f"     - {lead['contact_name'] or 'No name'} ({lead['contact_phone']}) - Status: {lead['sc_status']}, Device: {lead['assigned_device_id'] or 'None'}")
        
        # Check device-wise stats
        print("\n\n4. DEVICE-WISE BREAKDOWN:")
        cursor.execute("""
            SELECT 
                ud.device_name,
                COUNT(DISTINCT bm.recipient_phone) as unique_recipients,
                SUM(CASE WHEN bm.status = 'sent' THEN 1 ELSE 0 END) as sent,
                SUM(CASE WHEN bm.status = 'failed' THEN 1 ELSE 0 END) as failed,
                SUM(CASE WHEN bm.status = 'pending' THEN 1 ELSE 0 END) as pending
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.sequence_id = %s
            GROUP BY ud.device_name
            ORDER BY unique_recipients DESC
        """, (sequence_id,))
        
        devices = cursor.fetchall()
        
        for device in devices:
            print(f"   {device['device_name']}: {device['unique_recipients']} recipients")
            print(f"      Sent: {device['sent']}, Failed: {device['failed']}, Pending: {device['pending']}")
        
        print("\n\n5. THE ISSUE:")
        print("   - Sequence List shows 278 leads (from sequence_contacts)")
        print("   - Device Report shows 250 contacts should send (from broadcast_messages)")
        print("   - This means 28 leads are enrolled but have NO messages created")
        print("\n   Possible reasons:")
        print("   - These 28 leads might not have devices assigned")
        print("   - Or they were enrolled after messages were created")
        print("   - Or there's a filter preventing message creation")
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
