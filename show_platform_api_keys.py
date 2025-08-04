import pymysql
import os
import sys
import json

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
    
    with connection.cursor() as cursor:
        # Get devices with API errors and their platform config
        print("\nDEVICES WITH INVALID API KEY ERRORS - SHOWING PLATFORM CONFIGURATION:")
        print("-" * 100)
        
        cursor.execute("""
            SELECT DISTINCT
                bm.device_id,
                ud.device_name,
                ud.platform,
                ud.status,
                ud.whacenter_instance,
                ud.wablas_instance,
                COUNT(DISTINCT bm.id) as error_count,
                MAX(bm.created_at) as last_error
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
            GROUP BY bm.device_id, ud.device_name, ud.platform, ud.status, 
                     ud.whacenter_instance, ud.wablas_instance
            ORDER BY error_count DESC
        """)
        
        devices = cursor.fetchall()
        
        if devices:
            print(f"\nFound {len(devices)} devices with API key errors:\n")
            
            for idx, device in enumerate(devices, 1):
                print(f"\n{idx}. DEVICE: {device['device_name']} (ID: {device['device_id'][:12]}...)")
                print("   " + "=" * 80)
                print(f"   Platform: {device['platform'] or 'whatsapp'}")
                print(f"   Status: {device['status'] or 'unknown'}")
                print(f"   Error Count: {device['error_count']}")
                print(f"   Last Error: {device['last_error'].strftime('%Y-%m-%d %H:%M:%S') if device['last_error'] else 'N/A'}")
                
                # Parse platform configuration
                if device['platform'] == 'Whacenter' and device['whacenter_instance']:
                    print("\n   WHACENTER CONFIGURATION:")
                    try:
                        config = json.loads(device['whacenter_instance'])
                        print(f"   - API Key: {config.get('api_key', 'NOT FOUND')}")
                        print(f"   - Device ID: {config.get('device_id', 'NOT FOUND')}")
                        print(f"   - URL: {config.get('url', 'NOT FOUND')}")
                    except:
                        print(f"   - Raw Config: {device['whacenter_instance'][:100]}...")
                        
                elif device['platform'] == 'Wablas' and device['wablas_instance']:
                    print("\n   WABLAS CONFIGURATION:")
                    try:
                        config = json.loads(device['wablas_instance'])
                        print(f"   - API Key: {config.get('api_key', 'NOT FOUND')}")
                        print(f"   - Domain: {config.get('domain', 'NOT FOUND')}")
                    except:
                        print(f"   - Raw Config: {device['wablas_instance'][:100]}...")
                else:
                    print("\n   ❌ NO PLATFORM CONFIGURATION FOUND!")
                    
        # Also show all Whacenter/Wablas devices for comparison
        print("\n\n\nALL PLATFORM DEVICES WITH CONFIGURATION:")
        print("-" * 100)
        
        cursor.execute("""
            SELECT 
                id,
                device_name,
                platform,
                status,
                whacenter_instance,
                wablas_instance
            FROM user_devices
            WHERE platform IN ('Whacenter', 'Wablas')
               AND (whacenter_instance IS NOT NULL OR wablas_instance IS NOT NULL)
            ORDER BY platform, device_name
            LIMIT 20
        """)
        
        all_devices = cursor.fetchall()
        
        for device in all_devices:
            print(f"\n{device['device_name']} ({device['platform']}) - Status: {device['status']}")
            
            if device['platform'] == 'Whacenter' and device['whacenter_instance']:
                try:
                    config = json.loads(device['whacenter_instance'])
                    api_key = config.get('api_key', 'NOT FOUND')
                    print(f"  API Key: {api_key[:20]}..." if len(str(api_key)) > 20 else f"  API Key: {api_key}")
                except:
                    print(f"  Invalid JSON config")
                    
            elif device['platform'] == 'Wablas' and device['wablas_instance']:
                try:
                    config = json.loads(device['wablas_instance'])
                    api_key = config.get('api_key', 'NOT FOUND')
                    print(f"  API Key: {api_key[:20]}..." if len(str(api_key)) > 20 else f"  API Key: {api_key}")
                except:
                    print(f"  Invalid JSON config")
                    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
