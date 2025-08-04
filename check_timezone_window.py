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
        # Check current server time and +8 hours
        cursor.execute("SELECT NOW() as server_time, DATE_ADD(NOW(), INTERVAL 8 HOUR) as adjusted_time")
        result = cursor.fetchone()
        server_time = result['server_time']
        adjusted_time = result['adjusted_time']
        
        print(f"Current server time: {server_time}")
        print(f"Adjusted time (+8h): {adjusted_time}")
        print(f"Malaysia actual time should be around: 12:08 AM (past midnight)")
        print("=" * 80)
        print()
        
        # Check if any August 5 messages fall within the processing window
        print("Checking if August 5 messages are within processing window:")
        
        # Calculate the window
        window_start = adjusted_time - timedelta(minutes=10)
        window_end = adjusted_time
        
        print(f"Processing window: {window_start} to {window_end}")
        print()
        
        # Check messages in this window
        cursor.execute("""
            SELECT 
                COUNT(*) as count,
                MIN(scheduled_at) as earliest,
                MAX(scheduled_at) as latest
            FROM broadcast_messages
            WHERE status = 'pending'
            AND scheduled_at >= %s
            AND scheduled_at <= %s
        """, (window_start, window_end))
        
        result = cursor.fetchone()
        
        if result['count'] > 0:
            print(f"WARNING: {result['count']} messages are in the processing window!")
            print(f"  Earliest: {result['earliest']}")
            print(f"  Latest: {result['latest']}")
            
            # Get details
            cursor.execute("""
                SELECT 
                    DATE(scheduled_at) as scheduled_date,
                    COUNT(*) as count
                FROM broadcast_messages
                WHERE status = 'pending'
                AND scheduled_at >= %s
                AND scheduled_at <= %s
                GROUP BY DATE(scheduled_at)
            """, (window_start, window_end))
            
            by_date = cursor.fetchall()
            print("\nBy date:")
            for row in by_date:
                print(f"  {row['scheduled_date']}: {row['count']} messages")
                
        else:
            print("No messages in the current processing window")
            
        # Check the earliest August 5 message
        print("\n" + "=" * 80)
        print("Earliest August 5 messages:")
        cursor.execute("""
            SELECT 
                id,
                scheduled_at,
                recipient_phone,
                status
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            ORDER BY scheduled_at
            LIMIT 5
        """)
        
        early_msgs = cursor.fetchall()
        
        for msg in early_msgs:
            # Calculate when it will be processed
            process_time = msg['scheduled_at'] - timedelta(hours=8)
            print(f"\nScheduled: {msg['scheduled_at']} (Malaysia time)")
            print(f"  Will process at: {process_time} (server time)")
            print(f"  Status: {msg['status']}")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
