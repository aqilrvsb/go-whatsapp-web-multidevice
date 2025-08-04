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

# List of devices to revert
DEVICE_NAMES = [
    'SCAST-S30', 'SCARS-S46', 'SCRY-S08', 'SCAS-S74', 'SCARR-S39',
    'SCSHQ-S05', 'SCARS-S35', 'SCAS-S05', 'SMHQ-S05', 'SCAST-S59',
    'SCTTN-S77', 'SCAS-S40', 'SCHQ-S105', 'SCHQ-S02', 'SCHQ-S09'
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
    print("\nREVERTING RESCHEDULED MESSAGES BACK TO FAILED")
    print("=" * 100)
    
    # Ask for confirmation
    print("\n⚠️  WARNING: This will revert all pending messages for the 15 devices back to 'failed' status.")
    response = input("\nAre you sure you want to continue? (yes/no): ")
    if response.lower() != 'yes':
        print("Operation cancelled.")
        exit()
    
    with connection.cursor() as cursor:
        total_reverted = 0
        
        for device_name in DEVICE_NAMES:
            print(f"\n\nProcessing Device: {device_name}")
            print("-" * 50)
            
            # Get device ID
            cursor.execute("""
                SELECT id FROM user_devices WHERE device_name = %s
            """, (device_name,))
            
            device_result = cursor.fetchone()
            if not device_result:
                print(f"  Device not found: {device_name}")
                continue
                
            device_id = device_result['id']
            
            # Count pending messages that were rescheduled today
            cursor.execute("""
                SELECT COUNT(*) as count
                FROM broadcast_messages
                WHERE device_id = %s
                AND status = 'pending'
                AND DATE(updated_at) = CURDATE()
                AND scheduled_at >= NOW()
            """, (device_id,))
            
            count_result = cursor.fetchone()
            pending_count = count_result['count']
            
            if pending_count == 0:
                print(f"  No pending messages to revert")
                continue
            
            print(f"  Found {pending_count} pending messages to revert")
            
            # Determine appropriate error message based on platform
            cursor.execute("""
                SELECT platform FROM user_devices WHERE id = %s
            """, (device_id,))
            
            platform_result = cursor.fetchone()
            platform = platform_result['platform']
            
            # Set appropriate error message
            if platform == 'Whacenter':
                error_message = 'whacenter API error: status 401, body: {"status":false,"data":{},"message":"Invalid API Key"}'
            elif platform == 'Wablas':
                if device_name in ['SCHQ-S105', 'SCHQ-S02', 'SCHQ-S09']:
                    error_message = 'wablas API error: status 500, body: {"status":false,"message":"token invalid or device expired"}'
                else:
                    error_message = 'whacenter API error: status 401, body: {"status":false,"data":{},"message":"Invalid API Key"}'
            else:
                error_message = 'API error'
            
            # Revert messages back to failed
            cursor.execute("""
                UPDATE broadcast_messages
                SET status = 'failed',
                    error_message = %s,
                    updated_at = CURRENT_TIMESTAMP
                WHERE device_id = %s
                AND status = 'pending'
                AND DATE(updated_at) = CURDATE()
                AND scheduled_at >= NOW()
            """, (error_message, device_id))
            
            affected = cursor.rowcount
            connection.commit()
            
            print(f"  ✅ Reverted {affected} messages back to 'failed' status")
            total_reverted += affected
        
        print("\n" + "=" * 100)
        print(f"SUMMARY: Total messages reverted: {total_reverted}")
        print("=" * 100)
        
        # Show current status
        print("\n\nCURRENT STATUS:")
        print("-" * 60)
        
        for device_name in DEVICE_NAMES:
            cursor.execute("""
                SELECT 
                    COUNT(CASE WHEN bm.status = 'failed' THEN 1 END) as failed_count,
                    COUNT(CASE WHEN bm.status = 'pending' THEN 1 END) as pending_count
                FROM user_devices ud
                LEFT JOIN broadcast_messages bm ON bm.device_id = ud.id
                WHERE ud.device_name = %s
                GROUP BY ud.id
            """, (device_name,))
            
            result = cursor.fetchone()
            if result:
                print(f"{device_name}: {result['failed_count']} failed, {result['pending_count']} pending")
                
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
