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
        sequence_id = 'deccef4f-8ae1-4ed6-891c-bcb7d12baa8a'  # WARM Sequence
        
        print("Comparing sequence steps...")
        print("=" * 80)
        
        # Get steps from sequence_steps table
        print("\nSteps in sequence_steps table:")
        cursor.execute("""
            SELECT id, COALESCE(day_number, day, 1) as day_num
            FROM sequence_steps
            WHERE sequence_id = %s
            ORDER BY COALESCE(day_number, day, 1)
        """, (sequence_id,))
        
        db_steps = cursor.fetchall()
        db_step_ids = set()
        for step in db_steps:
            print(f"  Day {step['day_num']}: {step['id']}")
            db_step_ids.add(step['id'])
            
        # Get unique step IDs from broadcast_messages
        print("\nUnique step IDs in broadcast_messages:")
        cursor.execute("""
            SELECT DISTINCT sequence_stepid, COUNT(*) as count
            FROM broadcast_messages
            WHERE sequence_id = %s
            AND sequence_stepid IS NOT NULL
            GROUP BY sequence_stepid
        """, (sequence_id,))
        
        msg_steps = cursor.fetchall()
        msg_step_ids = set()
        for step in msg_steps:
            print(f"  {step['sequence_stepid']}: {step['count']} messages")
            msg_step_ids.add(step['sequence_stepid'])
            
        # Compare
        print(f"\nStep IDs in sequence_steps table: {len(db_step_ids)}")
        print(f"Unique step IDs in messages: {len(msg_step_ids)}")
        print(f"\nDo they match? {db_step_ids == msg_step_ids}")
        
        if db_step_ids != msg_step_ids:
            print("\nMismatch found!")
            print(f"In sequence_steps but not in messages: {db_step_ids - msg_step_ids}")
            print(f"In messages but not in sequence_steps: {msg_step_ids - db_step_ids}")
            
        # Check if this affects the query
        print("\n" + "=" * 80)
        print("Testing the actual query used in device report:")
        
        # This is the exact query from GetSequenceDeviceReport
        test_query = """
            SELECT 
                bm.sequence_stepid,
                COUNT(DISTINCT CONCAT(bm.sequence_stepid, '|', bm.recipient_phone, '|', bm.device_id)) as total
            FROM broadcast_messages bm
            WHERE bm.sequence_id = %s 
            AND bm.sequence_stepid IS NOT NULL
            AND DATE(bm.scheduled_at) = '2025-08-05'
            GROUP BY bm.sequence_stepid
        """
        
        cursor.execute(test_query, (sequence_id,))
        results = cursor.fetchall()
        
        print(f"\nResults for Aug 5:")
        for r in results:
            print(f"  Step {r['sequence_stepid']}: {r['total']} messages")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
