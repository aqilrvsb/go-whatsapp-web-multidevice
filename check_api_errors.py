import pymysql
import os
import sys
from datetime import datetime
from tabulate import tabulate

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
    print("=" * 80)
    
    with connection.cursor() as cursor:
        # 1. Get devices with API key errors
        print("\nDEVICES WITH INVALID API KEY ERRORS:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT DISTINCT
                bm.device_id,
                ud.device_name,
                ud.platform,
                ud.status as device_status,
                COUNT(DISTINCT bm.id) as error_count,
                MIN(bm.created_at) as first_error,
                MAX(bm.created_at) as last_error,
                bm.error_message
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
               OR bm.error_message LIKE '%API key%'
               OR bm.error_message LIKE '%Invalid Token%'
               OR bm.error_message LIKE '%Unauthorized%'
            GROUP BY bm.device_id, ud.device_name, ud.platform, ud.status, bm.error_message
            ORDER BY error_count DESC
        """)
        
        devices = cursor.fetchall()
        
        if devices:
            # Format data for display
            table_data = []
            for device in devices:
                table_data.append([
                    device['device_id'][:8] + '...' if device['device_id'] else 'N/A',
                    device['device_name'] or 'Unknown',
                    device['platform'] or 'whatsapp',
                    device['device_status'] or 'unknown',
                    device['error_count'],
                    device['first_error'].strftime('%Y-%m-%d %H:%M') if device['first_error'] else 'N/A',
                    device['last_error'].strftime('%Y-%m-%d %H:%M') if device['last_error'] else 'N/A',
                    (device['error_message'][:50] + '...') if device['error_message'] and len(device['error_message']) > 50 else device['error_message']
                ])
            
            headers = ['Device ID', 'Name', 'Platform', 'Status', 'Errors', 'First Error', 'Last Error', 'Error Message']
            print(tabulate(table_data, headers=headers, tablefmt='grid'))
        else:
            print("No devices found with API key errors!")
        
        # 2. Get summary statistics
        print("\nSUMMARY STATISTICS:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                COUNT(*) as total_messages,
                COUNT(DISTINCT device_id) as devices_affected,
                COUNT(DISTINCT DATE(created_at)) as days_affected
            FROM broadcast_messages
            WHERE error_message LIKE '%invalid api key%' 
               OR error_message LIKE '%Invalid API%'
               OR error_message LIKE '%API key%'
               OR error_message LIKE '%Invalid Token%'
               OR error_message LIKE '%Unauthorized%'
        """)
        
        stats = cursor.fetchone()
        print(f"Total messages with API errors: {stats['total_messages']:,}")
        print(f"Total devices affected: {stats['devices_affected']}")
        print(f"Days with errors: {stats['days_affected']}")
        
        # 3. Get platform breakdown
        print("\nBREAKDOWN BY PLATFORM:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                COALESCE(ud.platform, 'whatsapp') as platform,
                COUNT(DISTINCT bm.device_id) as devices,
                COUNT(DISTINCT bm.id) as messages
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
               OR bm.error_message LIKE '%API key%'
               OR bm.error_message LIKE '%Invalid Token%'
               OR bm.error_message LIKE '%Unauthorized%'
            GROUP BY COALESCE(ud.platform, 'whatsapp')
            ORDER BY messages DESC
        """)
        
        platforms = cursor.fetchall()
        
        if platforms:
            platform_data = []
            for platform in platforms:
                platform_data.append([
                    platform['platform'],
                    platform['devices'],
                    platform['messages']
                ])
            
            headers = ['Platform', 'Devices', 'Messages']
            print(tabulate(platform_data, headers=headers, tablefmt='grid'))
        
        # 4. Get recent examples
        print("\nRECENT ERROR EXAMPLES (Last 10):")
        print("-" * 80)
        
        cursor.execute("""
            SELECT 
                bm.device_id,
                ud.device_name,
                ud.platform,
                bm.recipient_phone,
                bm.error_message,
                bm.created_at
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
               OR bm.error_message LIKE '%API key%'
               OR bm.error_message LIKE '%Invalid Token%'
               OR bm.error_message LIKE '%Unauthorized%'
            ORDER BY bm.created_at DESC
            LIMIT 10
        """)
        
        recent = cursor.fetchall()
        
        if recent:
            for msg in recent:
                print(f"\nTime: {msg['created_at']}")
                print(f"Device: {msg['device_name']} ({msg['platform'] or 'whatsapp'})")
                print(f"To: {msg['recipient_phone']}")
                print(f"Error: {msg['error_message']}")
                print("-" * 40)
                
except Exception as e:
    print(f"Error: {e}")
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
