import pymysql
import os
import sys

# Set UTF-8 encoding for Windows
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')

# Get MySQL connection from environment
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
if mysql_uri.startswith('mysql://'):
    mysql_uri = mysql_uri[8:]
    
parts = mysql_uri.split('@')
user_pass = parts[0].split(':')
host_db = parts[1].split('/')

user = user_pass[0]
password = user_pass[1]
host_port = host_db[0].split(':')
host = host_port[0]
port = int(host_port[1]) if len(host_port) > 1 else 3306
database = host_db[1].split('?')[0]

try:
    # Connect to MySQL
    connection = pymysql.connect(
        host=host,
        port=port,
        user=user,
        password=password,
        database=database,
        cursorclass=pymysql.cursors.DictCursor
    )
    
    print("Connected to MySQL database")
    print("=" * 100)
    print("\nFINDING THE EXACT ISSUE - UI NOT FILTERING BY SEQUENCE STEP")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Get all sequence steps for SCVTC-S21 with pending messages
        print("\nGetting all sequence steps for SCVTC-S21...")
        
        cursor.execute("""
            SELECT 
                ss.id as step_id,
                s.name as sequence_name,
                ss.day,
                ss.message_type,
                COUNT(DISTINCT bm.recipient_phone) as unique_pending,
                COUNT(*) as total_pending
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            JOIN sequences s ON s.id = ss.sequence_id
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCVTC-S21'
            GROUP BY ss.id, s.name, ss.day, ss.message_type
            ORDER BY s.name, ss.day
        """)
        
        steps = cursor.fetchall()
        
        print(f"\nFound {len(steps)} sequence steps with pending messages:\n")
        
        for step in steps:
            print(f"{step['sequence_name']} - Day {step['day']} ({step['message_type']})")
            print(f"  Step ID: {step['step_id']}")
            print(f"  Unique pending: {step['unique_pending']}")
            print(f"  Total pending: {step['total_pending']}")
            print()
        
        # Find a step with exactly 20 pending
        twenty_step = None
        for step in steps:
            if step['unique_pending'] == 20:
                twenty_step = step
                break
        
        if twenty_step:
            print(f"\n\nFound step with exactly 20 pending: {twenty_step['sequence_name']} Day {twenty_step['day']}")
            print(f"Step ID: {twenty_step['step_id']}")
            
            # Get the actual 20 leads for this step
            print("\nThe 20 leads that SHOULD be shown:")
            cursor.execute("""
                SELECT 
                    recipient_phone,
                    recipient_name
                FROM broadcast_messages
                WHERE status = 'pending'
                AND sequence_stepid = %s
                ORDER BY recipient_name, recipient_phone
            """, (twenty_step['step_id'],))
            
            leads = cursor.fetchall()
            for i, lead in enumerate(leads, 1):
                print(f"  {i}. {lead['recipient_name'] or 'No name'} - {lead['recipient_phone']}")
            
            print(f"\n\nTHE PROBLEM:")
            print("When you click on this step showing '20 remaining', the UI should:")
            print(f"1. Filter by sequence_stepid = '{twenty_step['step_id']}'")
            print("2. Show ONLY these 20 leads")
            print("\nBut instead, it's showing ALL pending messages for the device!")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
