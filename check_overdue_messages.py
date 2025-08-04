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
        # Check current server time
        cursor.execute("SELECT NOW() as server_time, CURDATE() as server_date")
        result = cursor.fetchone()
        print(f"Current server time: {result['server_time']}")
        print(f"Current server date: {result['server_date']}")
        print()
        
        # Check messages that should have been sent by now
        cursor.execute("""
            SELECT 
                COUNT(*) as total_overdue,
                MIN(scheduled_at) as earliest_scheduled,
                MAX(scheduled_at) as latest_scheduled
            FROM broadcast_messages
            WHERE scheduled_at <= NOW()
            AND status = 'pending'
        """)
        
        overdue = cursor.fetchone()
        
        if overdue['total_overdue'] > 0:
            print(f"ALERT: {overdue['total_overdue']} messages are overdue!")
            print(f"  Earliest scheduled: {overdue['earliest_scheduled']}")
            print(f"  Latest scheduled: {overdue['latest_scheduled']}")
            print()
        
        # Check by sequence
        print("Overdue messages by sequence:")
        cursor.execute("""
            SELECT 
                s.name as sequence_name,
                bm.sequence_id,
                COUNT(*) as pending_count,
                MIN(bm.scheduled_at) as earliest,
                MAX(bm.scheduled_at) as latest
            FROM broadcast_messages bm
            LEFT JOIN sequences s ON s.id = bm.sequence_id
            WHERE bm.scheduled_at <= NOW()
            AND bm.status = 'pending'
            GROUP BY bm.sequence_id, s.name
            ORDER BY pending_count DESC
        """)
        
        sequences = cursor.fetchall()
        
        for seq in sequences:
            print(f"\n{seq['sequence_name'] or 'Unknown'} (ID: {seq['sequence_id']})")
            print(f"  Pending messages: {seq['pending_count']}")
            print(f"  Scheduled from: {seq['earliest']} to {seq['latest']}")
            
        # Check device status for these pending messages
        print("\nDevice status for pending messages:")
        cursor.execute("""
            SELECT 
                ud.device_name,
                ud.status as device_status,
                COUNT(bm.id) as pending_messages
            FROM broadcast_messages bm
            INNER JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.scheduled_at <= NOW()
            AND bm.status = 'pending'
            GROUP BY ud.id, ud.device_name, ud.status
            ORDER BY pending_messages DESC
            LIMIT 10
        """)
        
        devices = cursor.fetchall()
        
        for device in devices:
            print(f"\n{device['device_name']}:")
            print(f"  Device status: {device['device_status']}")
            print(f"  Pending messages: {device['pending_messages']}")
            
        # Check if there's a worker or scheduler issue
        print("\nRecent sent messages (to check if system is working):")
        cursor.execute("""
            SELECT 
                DATE(sent_at) as sent_date,
                COUNT(*) as messages_sent
            FROM broadcast_messages
            WHERE sent_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
            AND status = 'sent'
            GROUP BY DATE(sent_at)
            ORDER BY sent_date DESC
        """)
        
        recent_sent = cursor.fetchall()
        
        for sent in recent_sent:
            print(f"  {sent['sent_date']}: {sent['messages_sent']} messages sent")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
