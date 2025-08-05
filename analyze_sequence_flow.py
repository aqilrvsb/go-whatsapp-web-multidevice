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
        # Get a sample sequence message to understand the flow
        print("SEQUENCE MESSAGE FLOW ANALYSIS:")
        print("=" * 80)
        
        # 1. Check a sequence with steps
        cursor.execute("""
            SELECT s.id, s.name, s.trigger, 
                   COUNT(DISTINCT ss.id) as step_count
            FROM sequences s
            LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
            WHERE s.name LIKE '%HOT%'
            GROUP BY s.id, s.name, s.trigger
            LIMIT 1
        """)
        
        sequence = cursor.fetchone()
        if sequence:
            print(f"\n1. SEQUENCE: {sequence['name']}")
            print(f"   Trigger: {sequence['trigger']}")
            print(f"   Steps: {sequence['step_count']}")
            
            # 2. Get the steps
            cursor.execute("DESCRIBE sequence_steps")
            columns = cursor.fetchall()
            print("\n   Sequence steps columns:")
            for col in columns:
                print(f"   - {col['Field']}: {col['Type']}")
                
            cursor.execute("""
                SELECT id, day, content
                FROM sequence_steps
                WHERE sequence_id = %s
                ORDER BY day
            """, (sequence['id'],))
            
            steps = cursor.fetchall()
            print(f"\n2. SEQUENCE STEPS:")
            for i, step in enumerate(steps[:3]):  # Show first 3
                print(f"   - Step {i+1} (Day {step['day']})")
                
            # 3. Check how messages are scheduled
            print(f"\n3. SAMPLE SCHEDULED MESSAGES:")
            cursor.execute("""
                SELECT 
                    bm.id,
                    bm.recipient_phone,
                    bm.scheduled_at,
                    bm.created_at,
                    bm.status,
                    ss.id as step_id,
                    ss.day,
                    ss.hour,
                    ss.minute
                FROM broadcast_messages bm
                JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
                WHERE bm.sequence_id = %s
                ORDER BY bm.scheduled_at DESC
                LIMIT 5
            """, (sequence['id'],))
            
            messages = cursor.fetchall()
            for msg in messages:
                print(f"\n   Message: {msg['id'][:8]}...")
                print(f"   Step ID: {msg['step_id']} (Day {msg['day']} {msg['hour']:02d}:{msg['minute']:02d})")
                print(f"   Created: {msg['created_at']}")
                print(f"   Scheduled: {msg['scheduled_at']}")
                print(f"   Status: {msg['status']}")
                
                # Calculate when it was scheduled relative to creation
                time_diff = msg['scheduled_at'] - msg['created_at']
                days = time_diff.days
                hours = time_diff.seconds // 3600
                minutes = (time_diff.seconds % 3600) // 60
                print(f"   Scheduled for: {days} days, {hours} hours, {minutes} minutes after creation")
                
        # 4. Check timezone handling
        print("\n" + "=" * 80)
        print("4. TIMEZONE ANALYSIS:")
        
        # Get messages scheduled for specific times
        cursor.execute("""
            SELECT 
                HOUR(scheduled_at) as scheduled_hour,
                COUNT(*) as count
            FROM broadcast_messages
            WHERE DATE(scheduled_at) = '2025-08-05'
            AND status = 'pending'
            GROUP BY HOUR(scheduled_at)
            ORDER BY scheduled_hour
            LIMIT 10
        """)
        
        hourly = cursor.fetchall()
        print("\nMessages scheduled by hour (Aug 5):")
        for h in hourly:
            print(f"   Hour {h['scheduled_hour']:02d}:00 - {h['count']} messages")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
