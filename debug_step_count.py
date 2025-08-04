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
    print("\nFINDING DEVICE WITH JID FROM SCREENSHOT")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Find device by the JID shown in screenshot
        jid_from_screenshot = 'e9c46f74-e3ba-4148-8f5b-d1e9bdfc8d89'
        
        cursor.execute("""
            SELECT id, device_name, platform, jid 
            FROM user_devices 
            WHERE jid LIKE %s
        """, (f'%{jid_from_screenshot}%',))
        
        device = cursor.fetchone()
        if device:
            print(f"\nFound device: {device['device_name']}")
            print(f"Device ID: {device['id']}")
            print(f"Platform: {device['platform']}")
            device_id = device['id']
        else:
            # Try finding SCAS-S74 device
            cursor.execute("""
                SELECT id, device_name, platform, jid 
                FROM user_devices 
                WHERE device_name = 'SCAS-S74'
            """)
            device = cursor.fetchone()
            if device:
                print(f"\nFound SCAS-S74 device")
                print(f"Device ID: {device['id']}")
                print(f"JID: {device['jid']}")
                device_id = device['id']
            else:
                print("Device not found!")
                exit()
        
        # Now check all pending messages for this device
        print("\n\nALL PENDING MESSAGES FOR THIS DEVICE:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                bm.id,
                bm.recipient_phone,
                bm.recipient_name,
                bm.status,
                bm.sequence_id,
                bm.sequence_stepid,
                ss.day,
                ss.message_type,
                s.name as sequence_name
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            LEFT JOIN sequences s ON s.id = bm.sequence_id
            WHERE bm.device_id = %s
            AND bm.status = 'pending'
            ORDER BY ss.day, bm.recipient_phone
        """, (device_id,))
        
        pending_messages = cursor.fetchall()
        
        print(f"\nTotal pending messages: {len(pending_messages)}")
        
        # Group by day
        messages_by_day = {}
        for msg in pending_messages:
            day = msg['day'] or 'Unknown'
            if day not in messages_by_day:
                messages_by_day[day] = []
            messages_by_day[day].append(msg)
        
        for day, messages in sorted(messages_by_day.items()):
            print(f"\nDay {day}: {len(messages)} messages")
            for idx, msg in enumerate(messages[:10], 1):  # Show first 10
                print(f"  {idx}. {msg['recipient_name'] or 'No name'} - {msg['recipient_phone']} - Sequence: {msg['sequence_name']}")
            if len(messages) > 10:
                print(f"  ... and {len(messages) - 10} more")
        
        # Check specifically for day 4 messages
        print("\n\nDETAILED ANALYSIS FOR DAY 4:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                COUNT(*) as total_count,
                COUNT(DISTINCT recipient_phone) as unique_count,
                COUNT(DISTINCT sequence_stepid) as step_count
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.device_id = %s
            AND bm.status = 'pending'
            AND ss.day = 4
        """, (device_id,))
        
        stats = cursor.fetchone()
        print(f"Total messages: {stats['total_count']}")
        print(f"Unique recipients: {stats['unique_count']}")
        print(f"Different steps: {stats['step_count']}")
        
        # Check the summary calculation
        print("\n\nCHECKING HOW SUMMARY IS CALCULATED:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                ss.day,
                COUNT(DISTINCT CASE WHEN bm.status = 'pending' THEN bm.recipient_phone END) as pending,
                COUNT(DISTINCT CASE WHEN bm.status = 'sent' THEN bm.recipient_phone END) as sent,
                COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as failed,
                COUNT(DISTINCT bm.recipient_phone) as total
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.device_id = %s
            AND ss.day IN (4, 5)
            GROUP BY ss.day
        """, (device_id,))
        
        results = cursor.fetchall()
        
        for result in results:
            print(f"\nDay {result['day']}:")
            print(f"  Total unique recipients: {result['total']}")
            print(f"  Sent: {result['sent']}")
            print(f"  Failed: {result['failed']}")
            print(f"  Pending: {result['pending']}")
            print(f"  Calculated remaining (total - sent - failed): {result['total'] - result['sent'] - result['failed']}")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
