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
    print("\nINVESTIGATING STEP 1 REMAINING COUNT ISSUE FOR SCAS-S74")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # First, get the device ID for SCAS-S74
        cursor.execute("""
            SELECT id, jid FROM user_devices WHERE device_name = 'SCAS-S74'
        """)
        
        device_result = cursor.fetchone()
        if not device_result:
            print("Device SCAS-S74 not found!")
            exit()
            
        device_id = device_result['id']
        device_jid = device_result['jid']
        
        print(f"\nDevice ID: {device_id}")
        print(f"Device JID: {device_jid}")
        
        # Get the sequence and step information for Step 1
        print("\n\nSTEP 1 (DAY 4) ANALYSIS:")
        print("-" * 80)
        
        # Find all pending messages for this device for Step 1 (Day 4)
        cursor.execute("""
            SELECT 
                bm.id,
                bm.recipient_phone,
                bm.recipient_name,
                bm.status,
                bm.sequence_id,
                bm.sequence_stepid,
                ss.message_type,
                ss.day,
                bm.scheduled_at,
                bm.created_at
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.device_id = %s
            AND bm.status = 'pending'
            AND ss.day = 4
            ORDER BY bm.recipient_phone
        """, (device_id,))
        
        pending_messages = cursor.fetchall()
        
        print(f"\nTotal PENDING messages found for Step 1 (Day 4): {len(pending_messages)}")
        
        if pending_messages:
            print("\nPending messages details:")
            for idx, msg in enumerate(pending_messages, 1):
                print(f"{idx}. {msg['recipient_name'] or 'No name'} - {msg['recipient_phone']} - Created: {msg['created_at']}")
        
        # Now check the count using DISTINCT recipient_phone (as the system might be doing)
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT bm.recipient_phone) as unique_count,
                COUNT(*) as total_count
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.device_id = %s
            AND bm.status = 'pending'
            AND ss.day = 4
        """, (device_id,))
        
        count_result = cursor.fetchone()
        
        print(f"\n\nCOUNT ANALYSIS:")
        print(f"Unique phone numbers: {count_result['unique_count']}")
        print(f"Total messages: {count_result['total_count']}")
        
        # Check for duplicates
        if count_result['unique_count'] != count_result['total_count']:
            print("\n⚠️  DUPLICATE PHONE NUMBERS DETECTED!")
            
            cursor.execute("""
                SELECT 
                    recipient_phone,
                    COUNT(*) as count
                FROM broadcast_messages bm
                LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
                WHERE bm.device_id = %s
                AND bm.status = 'pending'
                AND ss.day = 4
                GROUP BY recipient_phone
                HAVING COUNT(*) > 1
            """, (device_id,))
            
            duplicates = cursor.fetchall()
            
            for dup in duplicates:
                print(f"Phone {dup['recipient_phone']} has {dup['count']} pending messages")
        
        # Check how the system calculates remaining (might be total - done - failed)
        print("\n\nCHECKING CALCULATION METHOD:")
        print("-" * 50)
        
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT CASE WHEN bm.status = 'pending' THEN bm.recipient_phone END) as pending_unique,
                COUNT(DISTINCT CASE WHEN bm.status = 'sent' THEN bm.recipient_phone END) as sent_unique,
                COUNT(DISTINCT CASE WHEN bm.status = 'failed' THEN bm.recipient_phone END) as failed_unique,
                COUNT(DISTINCT bm.recipient_phone) as total_unique
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.device_id = %s
            AND ss.day = 4
        """, (device_id,))
        
        calc_result = cursor.fetchone()
        
        print(f"Total unique recipients: {calc_result['total_unique']}")
        print(f"Sent (unique): {calc_result['sent_unique']}")
        print(f"Failed (unique): {calc_result['failed_unique']}")
        print(f"Pending (unique): {calc_result['pending_unique']}")
        
        # Calculate remaining as the system might be doing
        calculated_remaining = calc_result['total_unique'] - calc_result['sent_unique'] - calc_result['failed_unique']
        print(f"\nCalculated remaining (Total - Sent - Failed): {calculated_remaining}")
        print(f"Actual pending count: {calc_result['pending_unique']}")
        
        if calculated_remaining != calc_result['pending_unique']:
            print("\n❌ MISMATCH: The calculated remaining doesn't match actual pending count!")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
