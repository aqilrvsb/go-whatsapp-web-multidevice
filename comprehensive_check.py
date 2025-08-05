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
        print("COMPREHENSIVE CHECK OF ALL MESSAGES FOR 601117089042")
        print("=" * 80)
        
        # Get ALL messages sent between 22:00 Aug 4 and 14:00 Aug 5 (server time)
        # This covers 6:00 AM to 10:00 PM Aug 5 Malaysia time
        cursor.execute("""
            SELECT 
                bm.id,
                bm.content,
                bm.status,
                bm.scheduled_at,
                bm.sent_at,
                DATE_ADD(bm.sent_at, INTERVAL 8 HOUR) as sent_malaysia,
                TIME(DATE_ADD(bm.sent_at, INTERVAL 8 HOUR)) as sent_time_malaysia,
                bm.media_url,
                s.name as sequence_name,
                ud.device_name
            FROM broadcast_messages bm
            LEFT JOIN sequences s ON s.id = bm.sequence_id
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.recipient_phone = '601117089042'
            AND bm.status = 'sent'
            AND bm.sent_at >= '2025-08-04 22:00:00'
            AND bm.sent_at <= '2025-08-05 14:00:00'
            ORDER BY bm.sent_at
        """)
        
        messages = cursor.fetchall()
        
        print(f"Messages sent between 6 AM and 10 PM on Aug 5 (Malaysia time): {len(messages)}\n")
        
        for i, msg in enumerate(messages):
            print(f"MESSAGE {i+1}:")
            print(f"  Sent at: {msg['sent_time_malaysia']} Malaysia time")
            print(f"  Content: {msg['content'][:100]}...")
            print(f"  Has image: {'YES' if msg['media_url'] else 'NO'}")
            print(f"  From: {msg['sequence_name']}")
            print(f"  Device: {msg['device_name']}")
            print("-" * 60)
            
        # Check if messages exist with different phone format
        print("\n" + "=" * 80)
        print("Checking alternative phone formats:")
        
        alt_formats = ['601117089042', '60111-708-9042', '60111-7089042', '+601117089042', '01117089042']
        
        for phone in alt_formats:
            cursor.execute("""
                SELECT COUNT(*) as count
                FROM broadcast_messages
                WHERE recipient_phone = %s
                AND sent_at >= '2025-08-04 22:00:00'
                AND sent_at <= '2025-08-05 14:00:00'
            """, (phone,))
            
            result = cursor.fetchone()
            if result['count'] > 0:
                print(f"  {phone}: {result['count']} messages")
                
        # Check for any log entries or audit trail
        print("\n" + "=" * 80)
        print("Checking if there's a pattern with this device/sequence:")
        
        cursor.execute("""
            SELECT 
                DATE(DATE_ADD(sent_at, INTERVAL 8 HOUR)) as sent_date,
                COUNT(*) as messages_sent,
                COUNT(DISTINCT recipient_phone) as unique_recipients
            FROM broadcast_messages
            WHERE device_id = (
                SELECT device_id 
                FROM broadcast_messages 
                WHERE recipient_phone = '601117089042' 
                AND status = 'sent'
                LIMIT 1
            )
            AND sent_at >= '2025-08-04 00:00:00'
            GROUP BY DATE(DATE_ADD(sent_at, INTERVAL 8 HOUR))
            ORDER BY sent_date
        """)
        
        device_stats = cursor.fetchall()
        
        print("\nDevice sending pattern:")
        for stat in device_stats:
            print(f"  {stat['sent_date']}: {stat['messages_sent']} messages to {stat['unique_recipients']} recipients")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
