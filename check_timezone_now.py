import pymysql
import os
from datetime import datetime, timedelta

# Get MySQL connection
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
uri_parts = mysql_uri.replace('mysql://', '').split('@')
user_pass = uri_parts[0].split(':')
host_db = uri_parts[1].split('/')
host_port = host_db[0].split(':')

connection = pymysql.connect(
    host=host_port[0],
    port=int(host_port[1]),
    user=user_pass[0],
    password=user_pass[1],
    database=host_db[1],
    cursorclass=pymysql.cursors.DictCursor
)

try:
    with connection.cursor() as cursor:
        # Check current server time
        cursor.execute("SELECT NOW() as server_time")
        result = cursor.fetchone()
        server_time = result['server_time']
        
        print("CURRENT TIME CHECK:")
        print(f"Server time: {server_time}")
        print(f"Server + 8h: {server_time + timedelta(hours=8)}")
        
        # Calculate Malaysia time
        # If server is at 16:00 (4PM) and Malaysia is at 00:00 (midnight), difference is 8 hours
        # But now server shows 22:00 (10PM), so let's check
        
        print(f"\nIf Malaysia is now 6:45 AM (Aug 5), server should show 10:45 PM (Aug 4)")
        print(f"Actual server time: {server_time}")
        
        # Check time difference
        expected_server_time = datetime(2025, 8, 4, 22, 45, 0)  # 10:45 PM Aug 4
        actual_server_time = server_time.replace(microsecond=0)
        
        time_diff = actual_server_time - expected_server_time
        print(f"\nTime difference from expected: {time_diff}")
        
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
