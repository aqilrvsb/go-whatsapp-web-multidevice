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
    print("=" * 80)
    
    with connection.cursor() as cursor:
        # Check user_devices table structure
        print("\nUSER_DEVICES TABLE STRUCTURE:")
        print("-" * 80)
        
        cursor.execute("DESCRIBE user_devices")
        columns = cursor.fetchall()
        
        print("\nColumns in user_devices table:")
        for col in columns:
            print(f"  {col['Field']:20} | {col['Type']:30} | Null: {col['Null']}")
            
        # Look for any column that might contain API keys
        print("\n\nSearching for API key related columns...")
        
        # Check if there's a platform_devices or device_config table
        cursor.execute("SHOW TABLES LIKE '%device%'")
        tables = cursor.fetchall()
        
        print("\nDevice-related tables:")
        for table in tables:
            table_name = list(table.values())[0]
            print(f"  - {table_name}")
            
        # Check platform_devices table if it exists
        cursor.execute("SHOW TABLES LIKE 'platform_devices'")
        if cursor.fetchone():
            print("\n\nPLATFORM_DEVICES TABLE STRUCTURE:")
            print("-" * 80)
            cursor.execute("DESCRIBE platform_devices")
            columns = cursor.fetchall()
            
            print("\nColumns in platform_devices table:")
            for col in columns:
                print(f"  {col['Field']:20} | {col['Type']:30} | Null: {col['Null']}")
                
        # Let's check the actual data for platform devices
        print("\n\nDEVICES WITH API ERRORS - CHECKING PLATFORM CONFIG:")
        print("-" * 80)
        
        cursor.execute("""
            SELECT DISTINCT
                bm.device_id,
                ud.device_name,
                ud.platform,
                ud.status,
                ud.webhook_url,
                COUNT(DISTINCT bm.id) as error_count
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.error_message LIKE '%invalid api key%' 
               OR bm.error_message LIKE '%Invalid API%'
            GROUP BY bm.device_id, ud.device_name, ud.platform, ud.status, ud.webhook_url
            ORDER BY error_count DESC
            LIMIT 10
        """)
        
        devices = cursor.fetchall()
        
        for device in devices:
            print(f"\nDevice: {device['device_name']} ({device['device_id'][:12]}...)")
            print(f"  Platform: {device['platform']}")
            print(f"  Status: {device['status']}")
            print(f"  Webhook URL: {device['webhook_url'] or 'NOT SET'}")
            print(f"  Error Count: {device['error_count']}")
            
            # Check if there's any config stored elsewhere
            # Maybe in webhook_url or other fields
            if device['webhook_url']:
                print(f"  Webhook contains API info: {'api' in str(device['webhook_url']).lower()}")
                
except Exception as e:
    print(f"Error: {e}")
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\n\nDatabase connection closed")
