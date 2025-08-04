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
        cursor.execute("SELECT NOW() as server_time")
        result = cursor.fetchone()
        print(f"Current server time: {result['server_time']}")
        print("=" * 80)
        print()
        
        # Check August 5 messages status
        print("August 5, 2025 broadcast messages status:")
        cursor.execute("""
            SELECT 
                status,
                COUNT(*) as count,
                MIN(scheduled_at) as earliest,
                MAX(scheduled_at) as latest,
                MIN(sent_at) as first_sent,
                MAX(sent_at) as last_sent
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            GROUP BY status
            ORDER BY 
                CASE 
                    WHEN status = 'sent' THEN 1
                    WHEN status = 'pending' THEN 2
                    WHEN status = 'failed' THEN 3
                    ELSE 4
                END
        """)
        
        statuses = cursor.fetchall()
        
        total_messages = 0
        for status in statuses:
            total_messages += status['count']
            print(f"\n{status['status'].upper()}:")
            print(f"  Count: {status['count']}")
            print(f"  Scheduled from: {status['earliest']} to {status['latest']}")
            if status['status'] == 'sent' and status['first_sent']:
                print(f"  Actually sent from: {status['first_sent']} to {status['last_sent']}")
        
        print(f"\nTotal messages for August 5: {total_messages}")
        
        # Check when these messages were sent (by hour)
        print("\n" + "=" * 80)
        print("Messages sent by hour (server time):")
        cursor.execute("""
            SELECT 
                HOUR(sent_at) as hour,
                COUNT(*) as count,
                MIN(sent_at) as first_sent,
                MAX(sent_at) as last_sent
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            AND status = 'sent'
            AND sent_at IS NOT NULL
            GROUP BY HOUR(sent_at)
            ORDER BY hour
        """)
        
        hourly_data = cursor.fetchall()
        
        for hour in hourly_data:
            print(f"\nHour {hour['hour']:02d}:00:")
            print(f"  Messages sent: {hour['count']}")
            print(f"  From: {hour['first_sent']} to {hour['last_sent']}")
            
        # Check some sample messages to understand the pattern
        print("\n" + "=" * 80)
        print("Sample of sent messages for August 5:")
        cursor.execute("""
            SELECT 
                id,
                scheduled_at,
                sent_at,
                recipient_phone,
                device_id,
                sequence_id,
                campaign_id
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            AND status = 'sent'
            ORDER BY sent_at DESC
            LIMIT 10
        """)
        
        samples = cursor.fetchall()
        
        for msg in samples:
            time_diff = msg['sent_at'] - msg['scheduled_at']
            hours_early = time_diff.total_seconds() / 3600
            print(f"\nMessage ID: {msg['id'][:8]}...")
            print(f"  Scheduled: {msg['scheduled_at']}")
            print(f"  Sent: {msg['sent_at']}")
            print(f"  Sent {abs(hours_early):.1f} hours {'early' if hours_early < 0 else 'after schedule'}")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
