import pymysql
import os

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
        # First, find the WARM Sequence ID
        cursor.execute("""
            SELECT id, name 
            FROM sequences 
            WHERE name = 'WARM Sequence'
            LIMIT 1
        """)
        
        sequence = cursor.fetchone()
        if not sequence:
            print("WARM Sequence not found!")
            exit()
            
        sequence_id = sequence['id']
        print(f"Found sequence: {sequence['name']} (ID: {sequence_id})")
        print("=" * 80)
        
        # Check messages with and without date filter
        print("\n1. Total messages for this sequence (no date filter):")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(DISTINCT device_id) as unique_devices,
                MIN(scheduled_at) as earliest,
                MAX(scheduled_at) as latest
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"   Total messages: {result['total']}")
        print(f"   Unique devices: {result['unique_devices']}")
        print(f"   Date range: {result['earliest']} to {result['latest']}")
        
        # Check with August 5 filter
        print("\n2. Messages filtered for August 5, 2025:")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(DISTINCT device_id) as unique_devices
            FROM broadcast_messages
            WHERE sequence_id = %s
            AND DATE(scheduled_at) = '2025-08-05'
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"   Total messages: {result['total']}")
        print(f"   Unique devices: {result['unique_devices']}")
        
        # Check what dates actually have messages
        print("\n3. Messages by date:")
        cursor.execute("""
            SELECT 
                DATE(scheduled_at) as scheduled_date,
                COUNT(*) as count,
                COUNT(DISTINCT device_id) as devices
            FROM broadcast_messages
            WHERE sequence_id = %s
            GROUP BY DATE(scheduled_at)
            ORDER BY scheduled_date
            LIMIT 10
        """, (sequence_id,))
        
        dates = cursor.fetchall()
        for d in dates:
            print(f"   {d['scheduled_date']}: {d['count']} messages, {d['devices']} devices")
            
        # Check devices with messages
        print("\n4. Devices with messages for this sequence:")
        cursor.execute("""
            SELECT 
                bm.device_id,
                ud.device_name,
                COUNT(*) as message_count,
                MIN(bm.scheduled_at) as first_message,
                MAX(bm.scheduled_at) as last_message
            FROM broadcast_messages bm
            LEFT JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.sequence_id = %s
            GROUP BY bm.device_id, ud.device_name
            ORDER BY message_count DESC
            LIMIT 10
        """, (sequence_id,))
        
        devices = cursor.fetchall()
        for dev in devices:
            print(f"\n   Device: {dev['device_name']} (ID: {dev['device_id']})")
            print(f"   Messages: {dev['message_count']}")
            print(f"   Date range: {dev['first_message']} to {dev['last_message']}")
            
        # Check sequence steps
        print("\n5. Sequence steps:")
        cursor.execute("""
            SELECT 
                id,
                COALESCE(day_number, day, 1) as day_num,
                message_type
            FROM sequence_steps
            WHERE sequence_id = %s
            ORDER BY COALESCE(day_number, day, 1)
        """, (sequence_id,))
        
        steps = cursor.fetchall()
        for step in steps:
            print(f"   Step {step['day_num']}: {step['message_type']} (ID: {step['id']})")
            
            # Check messages for this step
            cursor.execute("""
                SELECT COUNT(*) as count
                FROM broadcast_messages
                WHERE sequence_stepid = %s
            """, (step['id'],))
            
            count = cursor.fetchone()
            print(f"      Messages: {count['count']}")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
