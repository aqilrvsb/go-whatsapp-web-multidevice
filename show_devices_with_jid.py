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
        # Get the 18 devices with API errors and their JID
        print("\n18 DEVICES WITH INVALID API KEY ERRORS - SHOWING JID:")
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
                MAX(bm.created_at) as last_error
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
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
                    device['platform'] or 'N/A',
                    device['status'] or 'N/A',
                    device['jid'] or 'NO JID',
                    device['phone'] or 'N/A',
                    device['error_count']
                ])
            
            headers = ['#', 'Device ID', 'Device Name', 'Platform', 'Status', 'JID', 'Phone', 'Errors']
            print(tabulate(table_data, headers=headers, tablefmt='grid'))
            
            # Count devices with and without JID
            with_jid = sum(1 for d in devices if d['jid'])
            without_jid = sum(1 for d in devices if not d['jid'])
            
            print(f"\n\nSUMMARY:")
            print(f"Total devices with API errors: {len(devices)}")
            print(f"Devices WITH JID: {with_jid}")
            print(f"Devices WITHOUT JID: {without_jid}")
            
            # Show detailed info for devices WITHOUT JID
            if without_jid > 0:
                print(f"\n\nDEVICES WITHOUT JID (These might not be properly connected):")
                print("-" * 80)
                for device in devices:
                    if not device['jid']:
                        print(f"  - {device['device_name']} ({device['device_id'][:12]}...) - Platform: {device['platform']}")
                        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
