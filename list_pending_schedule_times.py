import pymysql
import os
from datetime import datetime
from collections import defaultdict

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
        current_time = result['server_time']
        print(f"Current server time: {current_time}")
        print("=" * 80)
        print()
        
        # Get all pending messages scheduled for August 4, grouped by time
        print("Pending messages scheduled for August 4, 2025 (grouped by time):")
        print()
        
        cursor.execute("""
            SELECT 
                scheduled_at,
                COUNT(*) as message_count,
                GROUP_CONCAT(DISTINCT sequence_id) as sequence_ids,
                GROUP_CONCAT(DISTINCT device_id) as device_ids
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-04'
            AND status = 'pending'
            GROUP BY scheduled_at
            ORDER BY scheduled_at
        """)
        
        pending_times = cursor.fetchall()
        
        total_pending = 0
        time_groups = defaultdict(int)
        
        for row in pending_times:
            scheduled_time = row['scheduled_at']
            count = row['message_count']
            total_pending += count
            
            # Group by hour:minute
            time_str = scheduled_time.strftime("%H:%M")
            time_groups[time_str] += count
            
            # Check if this time has passed
            status = "OVERDUE" if scheduled_time < current_time else "Future"
            
            print(f"{scheduled_time} - {count} messages [{status}]")
            
        print()
        print("=" * 80)
        print(f"SUMMARY BY TIME SLOT:")
        print()
        
        for time_slot, count in sorted(time_groups.items()):
            hour, minute = time_slot.split(':')
            print(f"  {time_slot} - {count} messages")
            
        print()
        print(f"Total pending messages for August 4: {total_pending}")
        
        # Check which sequences have pending messages
        print()
        print("=" * 80)
        print("SEQUENCES WITH PENDING MESSAGES:")
        print()
        
        cursor.execute("""
            SELECT 
                s.name as sequence_name,
                bm.sequence_id,
                COUNT(*) as pending_count,
                MIN(bm.scheduled_at) as next_scheduled,
                MAX(bm.scheduled_at) as last_scheduled
            FROM broadcast_messages bm
            LEFT JOIN sequences s ON s.id = bm.sequence_id
            WHERE DATE(bm.scheduled_at) = '2025-08-04'
            AND bm.status = 'pending'
            GROUP BY bm.sequence_id, s.name
            ORDER BY pending_count DESC
        """)
        
        sequences = cursor.fetchall()
        
        for seq in sequences:
            print(f"{seq['sequence_name'] or 'Unknown'}")
            print(f"  Sequence ID: {seq['sequence_id']}")
            print(f"  Pending: {seq['pending_count']} messages")
            print(f"  Next: {seq['next_scheduled']}")
            print(f"  Last: {seq['last_scheduled']}")
            print()
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
