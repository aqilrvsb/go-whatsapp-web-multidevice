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
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Get devices with API errors and their API keys
        print("\nDEVICES WITH INVALID API KEY ERRORS - SHOWING API KEYS:")
        print("-" * 100)
        
        cursor.execute("""
            SELECT DISTINCT
                bm.device_id,
                ud.device_name,
                ud.platform,
                ud.status as device_status,
                ud.api_key,
                ud.webhook_url,
                COUNT(DISTINCT bm.id) as error_count,
                MAX(bm.created_at) as last_error
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
               OR bm.error_message LIKE '%API key%'
               OR bm.error_message LIKE '%Invalid Token%'
               OR bm.error_message LIKE '%Unauthorized%'
            GROUP BY bm.device_id, ud.device_name, ud.platform, ud.status, ud.api_key, ud.webhook_url
            ORDER BY error_count DESC
        """)
        
        devices = cursor.fetchall()
        
        if devices:
            print(f"\nFound {len(devices)} devices with API key errors:\n")
            
            for idx, device in enumerate(devices, 1):
                print(f"\n{idx}. DEVICE: {device['device_name']} (ID: {device['device_id'][:12]}...)")
                print("   " + "=" * 80)
                print(f"   Platform: {device['platform'] or 'whatsapp'}")
                print(f"   Status: {device['device_status'] or 'unknown'}")
                print(f"   Error Count: {device['error_count']}")
                print(f"   Last Error: {device['last_error'].strftime('%Y-%m-%d %H:%M:%S') if device['last_error'] else 'N/A'}")
                print(f"   API Key: {device['api_key'] or 'NOT SET'}")
                if device['webhook_url']:
                    print(f"   Webhook URL: {device['webhook_url']}")
                
                # Check if API key format looks correct
                if device['api_key']:
                    if device['platform'] == 'Whacenter':
                        if not device['api_key'].startswith('whacenter-'):
                            print("   ⚠️  WARNING: Whacenter API keys usually start with 'whacenter-'")
                    elif device['platform'] == 'Wablas':
                        if len(device['api_key']) < 20:
                            print("   ⚠️  WARNING: Wablas API keys are usually longer")
                else:
                    print("   ❌ ERROR: No API key configured!")
                    
        else:
            print("No devices found with API key errors!")
            
        # Also check devices without errors but with platform set
        print("\n\nALL PLATFORM DEVICES (FOR COMPARISON):")
        print("-" * 100)
        
        cursor.execute("""
            SELECT 
                id,
                device_name,
                platform,
                status,
                api_key,
                webhook_url
            FROM user_devices
            WHERE platform IN ('Whacenter', 'Wablas', 'whacenter', 'wablas')
               OR api_key IS NOT NULL
            ORDER BY platform, device_name
        """)
        
        all_platform_devices = cursor.fetchall()
        
        if all_platform_devices:
            # Group by platform
            by_platform = {}
            for device in all_platform_devices:
                platform = device['platform'] or 'Unknown'
                if platform not in by_platform:
                    by_platform[platform] = []
                by_platform[platform].append(device)
            
            for platform, devices in by_platform.items():
                print(f"\n{platform.upper()} DEVICES ({len(devices)}):")
                print("-" * 50)
                
                for device in devices:
                    api_key_display = device['api_key'] if device['api_key'] else 'NOT SET'
                    if len(api_key_display) > 50:
                        api_key_display = api_key_display[:47] + '...'
                    
                    print(f"{device['device_name']:15} | Status: {device['status']:8} | API Key: {api_key_display}")
                    
        # Check for duplicate API keys
        print("\n\nCHECKING FOR DUPLICATE API KEYS:")
        print("-" * 100)
        
        cursor.execute("""
            SELECT 
                api_key,
                COUNT(*) as device_count,
                GROUP_CONCAT(device_name SEPARATOR ', ') as devices
            FROM user_devices
            WHERE api_key IS NOT NULL 
              AND api_key != ''
            GROUP BY api_key
            HAVING COUNT(*) > 1
            ORDER BY device_count DESC
        """)
        
        duplicates = cursor.fetchall()
        
        if duplicates:
            print("\n⚠️  WARNING: Found duplicate API keys!")
            for dup in duplicates:
                print(f"\nAPI Key: {dup['api_key'][:30]}...")
                print(f"Used by {dup['device_count']} devices: {dup['devices']}")
        else:
            print("\n✅ No duplicate API keys found")
                
except Exception as e:
    print(f"Error: {e}")
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
