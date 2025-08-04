import pymysql
import os
from datetime import datetime

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
        # First, let's check the table structure
        cursor.execute("DESCRIBE broadcast_messages")
        columns = cursor.fetchall()
        print("Broadcast_messages table columns:")
        for col in columns:
            print(f"  {col['Field']} - {col['Type']}")
        print()
        
        print("Checking broadcast_messages scheduled for 4/08 but still pending...\n")
        
        # Check messages scheduled for August 4th that are still pending
        cursor.execute("""
            SELECT 
                id,
                recipient_phone,
                device_id,
                campaign_id,
                sequence_id,
                status,
                scheduled_at,
                created_at,
                updated_at,
                error_message
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-04'
            AND status IN ('pending', 'scheduled')
            ORDER BY scheduled_at
            LIMIT 20
        """)
        
        pending_messages = cursor.fetchall()
        
        if pending_messages:
            print(f"Found {len(pending_messages)} pending messages scheduled for August 4, 2025:\n")
            
            for msg in pending_messages:
                print(f"ID: {msg['id']}")
                print(f"  Phone: {msg['recipient_phone']}")
                print(f"  Status: {msg['status']}")
                print(f"  Scheduled: {msg['scheduled_at']}")
                print(f"  Created: {msg['created_at']}")
                print(f"  Device ID: {msg['device_id']}")
                print(f"  Campaign ID: {msg['campaign_id']}")
                print(f"  Sequence ID: {msg['sequence_id']}")
                if msg['error_message']:
                    print(f"  Error: {msg['error_message']}")
                print("-" * 80)
        else:
            print("No pending messages found scheduled for August 4, 2025")
            
        # Check overall statistics for that date
        print("\nOverall statistics for August 4, 2025:")
        cursor.execute("""
            SELECT 
                status,
                COUNT(*) as count
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-04'
            GROUP BY status
        """)
        
        stats = cursor.fetchall()
        
        total = 0
        for stat in stats:
            print(f"  {stat['status']}: {stat['count']} messages")
            total += stat['count']
        print(f"  TOTAL: {total} messages")
            
        # Check if there are any messages scheduled for future dates
        print("\nChecking for any future scheduled messages:")
        cursor.execute("""
            SELECT 
                DATE(scheduled_at) as scheduled_date,
                COUNT(*) as count,
                GROUP_CONCAT(DISTINCT status) as statuses
            FROM broadcast_messages
            WHERE scheduled_at > NOW()
            GROUP BY DATE(scheduled_at)
            ORDER BY scheduled_date
            LIMIT 10
        """)
        
        future_messages = cursor.fetchall()
        
        if future_messages:
            print("\nFuture scheduled messages by date:")
            for msg in future_messages:
                print(f"  {msg['scheduled_date']}: {msg['count']} messages (statuses: {msg['statuses']})")
        else:
            print("\nNo future scheduled messages found")
            
        # Let's also check the current date/time on the server
        cursor.execute("SELECT NOW() as current_time")
        result = cursor.fetchone()
        print(f"\nCurrent server time: {result['current_time']}")
        
        # Check if we have August 4 messages that should have been sent
        print("\nChecking messages that were scheduled for today but not sent:")
        cursor.execute("""
            SELECT 
                status,
                COUNT(*) as count,
                MIN(scheduled_at) as earliest,
                MAX(scheduled_at) as latest
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = DATE(NOW())
            AND status = 'pending'
            GROUP BY status
        """)
        
        today_pending = cursor.fetchall()
        
        if today_pending:
            for pending in today_pending:
                print(f"\nToday's pending messages: {pending['count']}")
                print(f"  Earliest scheduled: {pending['earliest']}")
                print(f"  Latest scheduled: {pending['latest']}")
        
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
