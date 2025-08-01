import pymysql
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Parse MySQL URI
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')
uri_parts = mysql_uri.replace('mysql://', '').split('@')
user_pass = uri_parts[0].split(':')
host_db = uri_parts[1].split('/')
host_port = host_db[0].split(':')

# Connect to MySQL
connection = pymysql.connect(
    host=host_port[0],
    port=int(host_port[1]),
    user=user_pass[0],
    password=user_pass[1],
    database=host_db[1],
    charset='utf8mb4'
)

try:
    with connection.cursor() as cursor:
        # Check if sequence exists
        sequence_id = '4d47df03-19a3-4ed7-be01-b9d89d62cceb'
        
        print(f"=== CHECKING SEQUENCE {sequence_id} ===")
        
        # Check sequence
        cursor.execute("SELECT id, name, niche, `trigger`, status FROM sequences WHERE id = %s", (sequence_id,))
        sequence = cursor.fetchone()
        if sequence:
            print(f"Sequence found: {sequence[1]} (Status: {sequence[4]})")
        else:
            print("Sequence NOT FOUND!")
            
        # Check sequence steps
        print("\n=== SEQUENCE STEPS ===")
        cursor.execute("""
            SELECT id, step_order, message_type, day_number 
            FROM sequence_steps 
            WHERE sequence_id = %s 
            ORDER BY step_order
        """, (sequence_id,))
        
        steps = cursor.fetchall()
        print(f"Found {len(steps)} steps")
        for step in steps:
            print(f"  Step {step[1]}: {step[0]} (Type: {step[2]}, Day: {step[3]})")
            
        # Check broadcast messages
        print("\n=== BROADCAST MESSAGES FOR SEQUENCE ===")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(DISTINCT device_id) as devices,
                COUNT(DISTINCT sequence_stepid) as steps_used
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"Total messages: {result[0]}, Devices: {result[1]}, Steps used: {result[2]}")
        
        # Check per step stats
        print("\n=== PER STEP STATISTICS ===")
        cursor.execute("""
            SELECT 
                bm.sequence_stepid,
                ss.step_order,
                COUNT(*) as total,
                COUNT(CASE WHEN bm.status = 'success' THEN 1 END) as success,
                COUNT(CASE WHEN bm.status = 'failed' THEN 1 END) as failed,
                COUNT(CASE WHEN bm.status = 'pending' THEN 1 END) as pending
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.sequence_id = %s
            GROUP BY bm.sequence_stepid, ss.step_order
            ORDER BY ss.step_order
        """, (sequence_id,))
        
        step_stats = cursor.fetchall()
        for stat in step_stats:
            print(f"Step {stat[1]} ({stat[0]}): Total={stat[2]}, Success={stat[3]}, Failed={stat[4]}, Pending={stat[5]}")

finally:
    connection.close()
