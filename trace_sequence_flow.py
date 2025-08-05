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
        print("COMPLETE SEQUENCE TO MESSAGE FLOW:")
        print("=" * 80)
        
        # 1. When a lead gets a trigger
        print("\n1. LEAD WITH TRIGGER:")
        cursor.execute("""
            SELECT l.id, l.phone, l.name, l.status
            FROM leads l
            WHERE l.status = 'hot_start'
            LIMIT 1
        """)
        lead = cursor.fetchone()
        if lead:
            print(f"   Lead: {lead['name']} ({lead['phone']})")
            print(f"   Trigger: {lead['status']}")
            
        # 2. Find sequences that match this trigger
        print("\n2. MATCHING SEQUENCES:")
        cursor.execute("""
            SELECT id, name, `trigger`
            FROM sequences
            WHERE `trigger` = 'hot_start'
            LIMIT 3
        """)
        sequences = cursor.fetchall()
        for seq in sequences:
            print(f"   - {seq['name']} (trigger: {seq['trigger']})")
            
        # 3. Check sequence steps
        if sequences:
            seq_id = sequences[0]['id']
            print(f"\n3. SEQUENCE STEPS FOR '{sequences[0]['name']}':")
            cursor.execute("""
                SELECT id, day, send_time, content, time_schedule
                FROM sequence_steps
                WHERE sequence_id = %s
                ORDER BY day
            """, (seq_id,))
            
            steps = cursor.fetchall()
            for step in steps:
                print(f"   Day {step['day']}: Send at {step['send_time']} (schedule: {step['time_schedule']})")
                
        # 4. Check how broadcast messages are created
        print("\n4. BROADCAST MESSAGE CREATION:")
        cursor.execute("""
            SELECT 
                bm.id,
                bm.created_at,
                bm.scheduled_at,
                bm.status,
                l.phone,
                l.status as lead_trigger,
                s.name as sequence_name,
                ss.day,
                ss.send_time
            FROM broadcast_messages bm
            JOIN leads l ON l.phone = bm.recipient_phone
            JOIN sequences s ON s.id = bm.sequence_id
            JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE s.`trigger` = l.status
            AND bm.scheduled_at >= '2025-08-05'
            ORDER BY bm.created_at DESC
            LIMIT 5
        """)
        
        messages = cursor.fetchall()
        print(f"\nRecent messages created from triggers:")
        for msg in messages:
            print(f"\n   Message ID: {msg['id'][:8]}...")
            print(f"   Lead trigger: {msg['lead_trigger']}")
            print(f"   Sequence: {msg['sequence_name']}")
            print(f"   Step: Day {msg['day']} at {msg['send_time']}")
            print(f"   Created: {msg['created_at']}")
            print(f"   Scheduled: {msg['scheduled_at']}")
            
            # Calculate scheduling logic
            created_date = msg['created_at'].date()
            scheduled_date = msg['scheduled_at'].date()
            days_added = (scheduled_date - created_date).days
            
            print(f"   => Scheduled {days_added} days after creation")
            
        # 5. Check timezone in scheduled_at
        print("\n5. TIMEZONE CHECK FOR SCHEDULED MESSAGES:")
        cursor.execute("""
            SELECT 
                DATE_FORMAT(scheduled_at, '%H:%i') as scheduled_time,
                COUNT(*) as count
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            AND sequence_id IS NOT NULL
            GROUP BY DATE_FORMAT(scheduled_at, '%H:%i')
            ORDER BY scheduled_time
            LIMIT 10
        """)
        
        times = cursor.fetchall()
        print("\nScheduled times for Aug 5 (sequence messages):")
        for t in times:
            print(f"   {t['scheduled_time']} - {t['count']} messages")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
