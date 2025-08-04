import pymysql
import os
import sys
from datetime import datetime, timedelta
import time

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
    'SCAST-S30',    # Done
    'SCARS-S46',    # Done
    'SCRY-S08',     # Done
    'SCAS-S74',     # Done
    'SCARR-S39',    # Need to process
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

# Check which devices already have pending messages scheduled for today
def check_device_status(cursor, device_name):
    cursor.execute("""
        SELECT COUNT(*) as pending_count
        FROM broadcast_messages bm
        JOIN user_devices ud ON ud.id = bm.device_id
        WHERE ud.device_name = %s
        AND bm.status = 'pending'
        AND DATE(bm.scheduled_at) = CURDATE()
    """, (device_name,))
    
    result = cursor.fetchone()
    return result['pending_count'] > 0

try:
    # Connect to MySQL
    connection = pymysql.connect(
        host=host,
        port=port,
        user=user,
        password=password,
        database=database,
        cursorclass=pymysql.cursors.DictCursor,
        autocommit=False  # Manual commit for better control
    )
    
    print("Connected to MySQL database")
    print("=" * 100)
    print("\nRESCHEDULING FAILED MESSAGES - CHECKING STATUS")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # First, check which devices have already been processed
        print("\nChecking device status...")
        processed_devices = []
        pending_devices = []
        
        for device_name in DEVICE_NAMES:
            if check_device_status(cursor, device_name):
                processed_devices.append(device_name)
            else:
                pending_devices.append(device_name)
        
        print(f"\n✅ Already processed: {len(processed_devices)} devices")
        for d in processed_devices:
            print(f"   - {d}")
            
        print(f"\n⏳ Need to process: {len(pending_devices)} devices")
        for d in pending_devices:
            print(f"   - {d}")
        
        # Ask for confirmation
        if pending_devices:
            response = input("\nDo you want to process the remaining devices? (yes/no): ")
            if response.lower() != 'yes':
                print("Operation cancelled.")
                exit()
        
        total_messages_updated = 0
        
        # Process only pending devices
        for device_name in pending_devices:
            print(f"\n\nProcessing Device: {device_name}")
            print("-" * 50)
            
            try:
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
                
                # Update in batches to avoid deadlock
                batch_size = 50
                for batch_start in range(0, len(failed_messages), batch_size):
                    batch_end = min(batch_start + batch_size, len(failed_messages))
                    batch = failed_messages[batch_start:batch_end]
                    
                    for idx, message in enumerate(batch):
                        # Calculate new scheduled time (30 minutes apart)
                        overall_idx = batch_start + idx
                        new_scheduled_time = start_time + timedelta(minutes=30 * overall_idx)
                        
                        # Update the message with retry on deadlock
                        retry_count = 0
                        while retry_count < 3:
                            try:
                                cursor.execute("""
                                    UPDATE broadcast_messages
                                    SET status = 'pending',
                                        error_message = NULL,
                                        scheduled_at = %s,
                                        updated_at = CURRENT_TIMESTAMP
                                    WHERE id = %s
                                """, (new_scheduled_time, message['id']))
                                break
                            except pymysql.err.OperationalError as e:
                                if e.args[0] == 1213:  # Deadlock
                                    retry_count += 1
                                    time.sleep(0.5)  # Wait before retry
                                else:
                                    raise
                        
                        if overall_idx < 5:  # Show first 5 messages as examples
                            print(f"    Message {overall_idx+1}: {message['recipient_phone']} -> Scheduled at {new_scheduled_time.strftime('%Y-%m-%d %H:%M:%S')}")
                    
                    # Commit after each batch
                    connection.commit()
                    print(f"    Processed batch {batch_start+1}-{batch_end} of {len(failed_messages)}")
                
                total_messages_updated += len(failed_messages)
                
                print(f"  ✅ Updated {len(failed_messages)} messages")
                print(f"  📅 First message: {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
                print(f"  📅 Last message: {(start_time + timedelta(minutes=30 * (len(failed_messages)-1))).strftime('%Y-%m-%d %H:%M:%S')}")
                print(f"  ⏱️ Total time span: {(30 * (len(failed_messages)-1)) / 60:.1f} hours")
                
            except Exception as e:
                print(f"  ❌ Error processing {device_name}: {e}")
                connection.rollback()
                continue
        
        print("\n" + "=" * 100)
        print(f"SUMMARY: Total messages rescheduled in this run: {total_messages_updated}")
        print("=" * 100)
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
