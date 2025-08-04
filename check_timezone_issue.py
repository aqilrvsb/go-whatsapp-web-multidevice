import pymysql
import os
from datetime import datetime
import pytz

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
        # Check server timezone
        cursor.execute("SELECT @@global.time_zone, @@session.time_zone")
        tz_info = cursor.fetchone()
        print("MySQL Timezone Settings:")
        print(f"  Global: {tz_info['@@global.time_zone']}")
        print(f"  Session: {tz_info['@@session.time_zone']}")
        print()
        
        # Check current time
        cursor.execute("SELECT NOW() as server_time, CONVERT_TZ(NOW(), @@session.time_zone, '+00:00') as utc_time")
        result = cursor.fetchone()
        server_time = result['server_time']
        utc_time = result['utc_time']
        
        print(f"Server Time: {server_time}")
        print(f"UTC Time: {utc_time}")
        
        # Calculate Malaysia time
        malaysia_tz = pytz.timezone('Asia/Kuala_Lumpur')
        malaysia_time = datetime.now(malaysia_tz)
        print(f"Malaysia Time (actual): {malaysia_time.strftime('%Y-%m-%d %H:%M:%S')}")
        
        # Calculate the time difference
        server_dt = datetime.strptime(str(server_time), '%Y-%m-%d %H:%M:%S')
        malaysia_dt = datetime.strptime(malaysia_time.strftime('%Y-%m-%d %H:%M:%S'), '%Y-%m-%d %H:%M:%S')
        time_diff = malaysia_dt - server_dt
        hours_diff = time_diff.total_seconds() / 3600
        
        print(f"\nTime Difference: {hours_diff:.1f} hours")
        print("=" * 80)
        
        # Check messages that should have been sent by Malaysia time
        malaysia_time_str = malaysia_time.strftime('%Y-%m-%d %H:%M:%S')
        print(f"\nMessages that should have been sent by {malaysia_time_str} (Malaysia time):")
        print(f"But the server thinks it's only {server_time}")
        print()
        
        # Get overdue messages based on Malaysia time
        cursor.execute("""
            SELECT 
                COUNT(*) as total_overdue,
                MIN(scheduled_at) as earliest,
                MAX(scheduled_at) as latest
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-04'
            AND status = 'pending'
            AND scheduled_at <= DATE_ADD(NOW(), INTERVAL %s HOUR)
        """, (hours_diff,))
        
        overdue = cursor.fetchone()
        
        print(f"OVERDUE MESSAGES (based on Malaysia time):")
        print(f"  Total: {overdue['total_overdue']} messages")
        print(f"  Earliest scheduled: {overdue['earliest']}")
        print(f"  Latest scheduled: {overdue['latest']}")
        print()
        
        # Show what will happen at midnight
        print("URGENT: In 30 minutes when it becomes August 5 in Malaysia:")
        print("  - These 1,000 pending messages will be 1 day old")
        print("  - They may never be sent if the system only processes current day messages")
        print()
        
        # Check if there are messages for August 5
        cursor.execute("""
            SELECT COUNT(*) as aug5_count
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
        """)
        
        aug5 = cursor.fetchone()
        print(f"Messages already scheduled for August 5: {aug5['aug5_count']}")
        
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
