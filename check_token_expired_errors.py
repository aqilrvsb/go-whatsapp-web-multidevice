import pymysql
import os
import sys
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
    print("=" * 120)
    
    with connection.cursor() as cursor:
        # Get devices with token invalid or device expired errors
        print("\nDEVICES WITH TOKEN INVALID OR DEVICE EXPIRED ERRORS:")
        print("-" * 120)
        
        cursor.execute("""
            SELECT DISTINCT
                bm.device_id,
                ud.device_name,
                ud.platform,
                ud.status,
                ud.jid,
                ud.phone,
                COUNT(DISTINCT bm.id) as error_count,
                MIN(bm.created_at) as first_error,
                MAX(bm.created_at) as last_error,
                MAX(bm.error_message) as sample_error
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%token invalid%' 
               OR bm.error_message LIKE '%device expired%'
               OR bm.error_message LIKE '%token expired%'
               OR bm.error_message LIKE '%invalid token%'
               OR bm.error_message LIKE '%expired device%'
            GROUP BY bm.device_id, ud.device_name, ud.platform, ud.status, ud.jid, ud.phone
            ORDER BY error_count DESC
        """)
        
        devices = cursor.fetchall()
        
        if devices:
            # Prepare data for table
            table_data = []
            for idx, device in enumerate(devices, 1):
                table_data.append([
                    idx,
                    device['device_id'][:12] + '...',
                    device['device_name'],
                    device['platform'] or 'whatsapp',
                    device['status'] or 'N/A',
                    device['jid'][:20] + '...' if device['jid'] and len(device['jid']) > 20 else (device['jid'] or 'NO JID'),
                    device['error_count'],
                    device['last_error'].strftime('%Y-%m-%d %H:%M') if device['last_error'] else 'N/A'
                ])
            
            headers = ['#', 'Device ID', 'Device Name', 'Platform', 'Status', 'JID', 'Errors', 'Last Error']
            print(tabulate(table_data, headers=headers, tablefmt='grid'))
            
            # Create text file with detailed info
            with open('devices_token_expired_errors.txt', 'w', encoding='utf-8') as f:
                f.write("DEVICES WITH TOKEN INVALID OR DEVICE EXPIRED ERRORS\n")
                f.write("=" * 70 + "\n\n")
                f.write(f"Total Devices Found: {len(devices)}\n")
                f.write("-" * 70 + "\n\n")
                
                for idx, device in enumerate(devices, 1):
                    f.write(f"{idx}. Device Name: {device['device_name']}\n")
                    f.write(f"   Platform: {device['platform'] or 'whatsapp'}\n")
                    f.write(f"   JID: {device['jid'] or 'NO JID'}\n")
                    f.write(f"   Device ID: {device['device_id']}\n")
                    f.write(f"   Status: {device['status'] or 'N/A'}\n")
                    f.write(f"   Error Count: {device['error_count']}\n")
                    f.write(f"   First Error: {device['first_error'].strftime('%Y-%m-%d %H:%M:%S') if device['first_error'] else 'N/A'}\n")
                    f.write(f"   Last Error: {device['last_error'].strftime('%Y-%m-%d %H:%M:%S') if device['last_error'] else 'N/A'}\n")
                    f.write(f"   Sample Error: {device['sample_error'][:100]}...\n" if device['sample_error'] and len(device['sample_error']) > 100 else f"   Sample Error: {device['sample_error']}\n")
                    f.write("\n")
                
                f.write("\n" + "=" * 70 + "\n")
                f.write("SUMMARY BY PLATFORM:\n")
                f.write("-" * 70 + "\n\n")
                
                # Get platform breakdown
                cursor.execute("""
                    SELECT 
                        COALESCE(ud.platform, 'whatsapp') as platform,
                        COUNT(DISTINCT bm.device_id) as device_count,
                        COUNT(DISTINCT bm.id) as message_count
                    FROM broadcast_messages bm
                    LEFT JOIN user_devices ud ON ud.id = bm.device_id
                    WHERE bm.error_message LIKE '%token invalid%' 
                       OR bm.error_message LIKE '%device expired%'
                       OR bm.error_message LIKE '%token expired%'
                       OR bm.error_message LIKE '%invalid token%'
                       OR bm.error_message LIKE '%expired device%'
                    GROUP BY COALESCE(ud.platform, 'whatsapp')
                    ORDER BY message_count DESC
                """)
                
                platforms = cursor.fetchall()
                
                for platform in platforms:
                    f.write(f"{platform['platform']}: {platform['device_count']} devices, {platform['message_count']} messages\n")
            
            print(f"\n\nFile saved: devices_token_expired_errors.txt")
            print(f"Total devices with token/expired errors: {len(devices)}")
            
            # Show summary
            with_jid = sum(1 for d in devices if d['jid'])
            without_jid = sum(1 for d in devices if not d['jid'])
            
            print(f"\nSUMMARY:")
            print(f"Devices WITH JID: {with_jid}")
            print(f"Devices WITHOUT JID: {without_jid}")
            
            # Show recent error examples
            print("\n\nRECENT ERROR EXAMPLES:")
            print("-" * 80)
            
            cursor.execute("""
                SELECT 
                    bm.device_id,
                    ud.device_name,
                    bm.error_message,
                    bm.created_at
                FROM broadcast_messages bm
                LEFT JOIN user_devices ud ON ud.id = bm.device_id
                WHERE bm.error_message LIKE '%token invalid%' 
                   OR bm.error_message LIKE '%device expired%'
                   OR bm.error_message LIKE '%token expired%'
                   OR bm.error_message LIKE '%invalid token%'
                   OR bm.error_message LIKE '%expired device%'
                ORDER BY bm.created_at DESC
                LIMIT 5
            """)
            
            recent = cursor.fetchall()
            
            for msg in recent:
                print(f"\nDevice: {msg['device_name']}")
                print(f"Time: {msg['created_at']}")
                print(f"Error: {msg['error_message']}")
                
        else:
            print("No devices found with token invalid or device expired errors!")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
