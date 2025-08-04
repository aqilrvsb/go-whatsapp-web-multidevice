import pymysql
import os
import sys
from datetime import datetime, timedelta

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

# List of devices to process
DEVICE_NAMES = [
    'SCAST-S30',
    'SCARS-S46', 
    'SCRY-S08',
    'SCAS-S74',
    'SCARR-S39',
    'SCSHQ-S05',
    'SCARS-S35',
    'SCAS-S05',
    'SMHQ-S05',
    'SCAST-S59',
    'SCTTN-S77',
    'SCAS-S40',
    'SCHQ-S105',
    'SCHQ-S02',
    'SCHQ-S09'
]

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
    print("\nRESCHEDULING FAILED MESSAGES FOR 15 DEVICES")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        total_messages_updated = 0
        
        # Process each device
        for device_name in DEVICE_NAMES:
            print(f"\n\nProcessing Device: {device_name}")
            print("-" * 50)
            
            # First, get the device ID
            cursor.execute("""
                SELECT id FROM user_devices WHERE device_name = %s
            """, (device_name,))
            
            device_result = cursor.fetchone()
            if not device_result:
                print(f"  Device not found: {device_name}")
                continue
                
            device_id = device_result['id']
            
            # Get all failed messages for this device, ordered by original scheduled_at
            cursor.execute("""
                SELECT id, recipient_phone, scheduled_at, error_message
                FROM broadcast_messages
                WHERE device_id = %s
                AND status = 'failed'
                ORDER BY scheduled_at ASC, created_at ASC
            """, (device_id,))
            
            failed_messages = cursor.fetchall()
            
            if not failed_messages:
                print(f"  No failed messages found for {device_name}")
                continue
            
            print(f"  Found {len(failed_messages)} failed messages")
            
            # Start time is NOW
            start_time = datetime.now()
            
            # Update each message with new schedule
            for idx, message in enumerate(failed_messages):
                # Calculate new scheduled time (30 minutes apart)
                new_scheduled_time = start_time + timedelta(minutes=30 * idx)
                
                # Update the message
                cursor.execute("""
                    UPDATE broadcast_messages
                    SET status = 'pending',
                        error_message = NULL,
                        scheduled_at = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """, (new_scheduled_time, message['id']))
                
                if idx < 5:  # Show first 5 messages as examples
                    print(f"    Message {idx+1}: {message['recipient_phone']} -> Scheduled at {new_scheduled_time.strftime('%Y-%m-%d %H:%M:%S')}")
            
            # Commit after each device
            connection.commit()
            
            total_messages_updated += len(failed_messages)
            
            print(f"  Updated {len(failed_messages)} messages")
            print(f"  First message: {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
            print(f"  Last message: {(start_time + timedelta(minutes=30 * (len(failed_messages)-1))).strftime('%Y-%m-%d %H:%M:%S')}")
            print(f"  Total time span: {(30 * (len(failed_messages)-1)) / 60:.1f} hours")
        
        print("\n" + "=" * 100)
        print(f"SUMMARY: Total messages rescheduled: {total_messages_updated}")
        print("=" * 100)
        
        # Show overall statistics
        print("\n\nOVERALL STATISTICS:")
        print("-" * 50)
        
        for device_name in DEVICE_NAMES:
            cursor.execute("""
                SELECT 
                    COUNT(*) as total,
                    MIN(scheduled_at) as first_send,
                    MAX(scheduled_at) as last_send
                FROM broadcast_messages bm
                JOIN user_devices ud ON ud.id = bm.device_id
                WHERE ud.device_name = %s
                AND bm.status = 'pending'
                AND bm.scheduled_at >= %s
            """, (device_name, start_time))
            
            stats = cursor.fetchone()
            if stats['total'] > 0:
                print(f"{device_name}: {stats['total']} messages, "
                      f"First: {stats['first_send'].strftime('%H:%M') if stats['first_send'] else 'N/A'}, "
                      f"Last: {stats['last_send'].strftime('%d %b %H:%M') if stats['last_send'] else 'N/A'}")
                
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
