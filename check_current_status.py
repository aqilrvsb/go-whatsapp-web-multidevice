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
        # 1. Check current server time
        cursor.execute("SELECT NOW() as server_time")
        result = cursor.fetchone()
        server_time = result['server_time']
        
        print("1. CURRENT TIME STATUS:")
        print(f"   Server time: {server_time}")
        print(f"   Server time + 8 hours: {server_time + timedelta(hours=8)}")
        print(f"   Malaysia actual time: ~12:15 AM (August 5)")
        print()
        
        # 2. Show the processing window
        print("2. PROCESSING WINDOW (with 8-hour adjustment):")
        window_start = server_time + timedelta(hours=8) - timedelta(minutes=10)
        window_end = server_time + timedelta(hours=8)
        
        print(f"   Messages will be processed if scheduled between:")
        print(f"   {window_start} and {window_end}")
        print()
        
        # Check what's in the current window
        cursor.execute("""
            SELECT COUNT(*) as count,
                   MIN(scheduled_at) as earliest,
                   MAX(scheduled_at) as latest
            FROM broadcast_messages
            WHERE status = 'pending'
            AND scheduled_at >= %s
            AND scheduled_at <= %s
        """, (window_start, window_end))
        
        result = cursor.fetchone()
        if result['count'] > 0:
            print(f"   Currently {result['count']} messages in processing window")
            print(f"   From: {result['earliest']} to {result['latest']}")
        else:
            print("   No messages currently in processing window")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
