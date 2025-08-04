import pymysql
import os
import sys
from datetime import datetime

# Set UTF-8 encoding for Windows
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')

# Get MySQL connection from environment
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
if mysql_uri.startswith('mysql://'):
    mysql_uri = mysql_uri[8:]  # Remove mysql://
    
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
    print("\nCOUNT VS DISPLAY ANALYSIS - ENSURING COUNTS MATCH DISPLAYED LEADS")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Get all devices with pending messages
        cursor.execute("""
            SELECT DISTINCT 
                ud.id as device_id,
                ud.device_name,
                ud.platform,
                COUNT(DISTINCT bm.id) as total_pending
            FROM user_devices ud
            JOIN broadcast_messages bm ON bm.device_id = ud.id
            WHERE bm.status = 'pending'
            GROUP BY ud.id, ud.device_name, ud.platform
            ORDER BY ud.device_name
        """)
        
        devices = cursor.fetchall()
        
        print(f"\nFound {len(devices)} devices with pending messages\n")
        
        mismatches = []
        
        for device in devices:
            device_id = device['device_id']
            device_name = device['device_name']
            
            print(f"\n{'=' * 80}")
            print(f"DEVICE: {device_name} ({device['platform']})")
            print(f"{'=' * 80}")
            
            # Get sequence steps with counts
            cursor.execute("""
                SELECT 
                    ss.id as step_id,
                    ss.sequence_id,
                    s.name as sequence_name,
                    ss.day,
                    ss.message_type,
                    COUNT(DISTINCT bm.recipient_phone) as unique_recipients,
                    COUNT(bm.id) as total_messages,
                    GROUP_CONCAT(DISTINCT bm.status) as statuses
                FROM broadcast_messages bm
                JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
                JOIN sequences s ON s.id = ss.sequence_id
                WHERE bm.device_id = %s
                AND bm.status = 'pending'
                GROUP BY ss.id, ss.sequence_id, s.name, ss.day, ss.message_type
                ORDER BY s.name, ss.day
            """, (device_id,))
            
            steps = cursor.fetchall()
            
            if not steps:
                # Check for messages without sequence steps
                cursor.execute("""
                    SELECT 
                        COUNT(DISTINCT recipient_phone) as unique_recipients,
                        COUNT(*) as total_messages
                    FROM broadcast_messages
                    WHERE device_id = %s
                    AND status = 'pending'
                    AND sequence_stepid IS NULL
                """, (device_id,))
                
                no_step = cursor.fetchone()
                if no_step['total_messages'] > 0:
                    print(f"\n⚠️  Messages without sequence step: {no_step['total_messages']} messages, {no_step['unique_recipients']} unique recipients")
            
            for step in steps:
                print(f"\nSequence: {step['sequence_name']}")
                print(f"Step: Day {step['day'] or 'Unknown'} - {step['message_type']}")
                print(f"Count: {step['unique_recipients']} unique recipients")
                print(f"Total: {step['total_messages']} total messages")
                
                if step['unique_recipients'] != step['total_messages']:
                    print(f"❌ MISMATCH: Count shows {step['unique_recipients']} but has {step['total_messages']} messages!")
                    
                    # Get the actual pending messages for this step
                    cursor.execute("""
                        SELECT 
                            recipient_phone,
                            recipient_name,
                            COUNT(*) as duplicates
                        FROM broadcast_messages
                        WHERE device_id = %s
                        AND sequence_stepid = %s
                        AND status = 'pending'
                        GROUP BY recipient_phone, recipient_name
                        ORDER BY recipient_phone
                        LIMIT 10
                    """, (device_id, step['step_id']))
                    
                    recipients = cursor.fetchall()
                    
                    print("\n  Pending recipients (first 10):")
                    for idx, r in enumerate(recipients, 1):
                        if r['duplicates'] > 1:
                            print(f"  {idx}. {r['recipient_name'] or 'No name'} - {r['recipient_phone']} (⚠️  {r['duplicates']} duplicates)")
                        else:
                            print(f"  {idx}. {r['recipient_name'] or 'No name'} - {r['recipient_phone']}")
                    
                    # Check for duplicate entries
                    cursor.execute("""
                        SELECT 
                            recipient_phone,
                            COUNT(*) as count
                        FROM broadcast_messages
                        WHERE device_id = %s
                        AND sequence_stepid = %s
                        AND status = 'pending'
                        GROUP BY recipient_phone
                        HAVING COUNT(*) > 1
                    """, (device_id, step['step_id']))
                    
                    duplicates = cursor.fetchall()
                    
                    if duplicates:
                        print(f"\n  ⚠️  Found {len(duplicates)} phone numbers with duplicate entries!")
                        mismatches.append({
                            'device': device_name,
                            'sequence': step['sequence_name'],
                            'day': step['day'],
                            'unique': step['unique_recipients'],
                            'total': step['total_messages'],
                            'duplicates': len(duplicates)
                        })
                else:
                    print("✅ Count matches displayed leads")
        
        # Summary of mismatches
        if mismatches:
            print(f"\n\n{'=' * 100}")
            print("SUMMARY OF MISMATCHES")
            print("=" * 100)
            
            for m in mismatches:
                print(f"\n{m['device']} - {m['sequence']} Day {m['day']}:")
                print(f"  Shows: {m['unique']} unique")
                print(f"  Has: {m['total']} total messages") 
                print(f"  Duplicates: {m['duplicates']} phone numbers")
            
            print(f"\n\n{'=' * 100}")
            print("RECOMMENDATION")
            print("=" * 100)
            print("\nTo fix this issue, the system should:")
            print("1. Use DISTINCT recipient_phone in both count AND display queries")
            print("2. OR show all messages but update the count to match")
            print("3. OR remove duplicate entries from the database")
            
            # Create SQL to find and remove duplicates
            print("\n\nSQL TO IDENTIFY DUPLICATES:")
            print("-" * 50)
            print("""
SELECT 
    device_id,
    sequence_stepid,
    recipient_phone,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(id) as message_ids
FROM broadcast_messages
WHERE status = 'pending'
GROUP BY device_id, sequence_stepid, recipient_phone
HAVING COUNT(*) > 1;
            """)
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
