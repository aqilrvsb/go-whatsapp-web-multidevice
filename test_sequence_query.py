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
        # Test the fixed query
        sequence_id = '4d47df03-19a3-4ed7-be01-b9d89d62cceb'
        
        print(f"=== Testing sequence steps query for sequence {sequence_id} ===")
        
        query = """
            SELECT id, COALESCE(day_number, day, 1) as step_order, message_type, content, COALESCE(day_number, day, 1) as day_num
            FROM sequence_steps
            WHERE sequence_id = %s
            ORDER BY COALESCE(day_number, day, 1)
        """
        
        cursor.execute(query, (sequence_id,))
        steps = cursor.fetchall()
        
        print(f"Found {len(steps)} steps")
        for step in steps:
            print(f"Step {step[1]}: {step[2]} (ID: {step[0]})")
            
        # Check if sequence exists
        print("\n=== Checking sequence exists ===")
        cursor.execute("SELECT id, name FROM sequences WHERE id = %s", (sequence_id,))
        seq = cursor.fetchone()
        if seq:
            print(f"Sequence found: {seq[1]} (ID: {seq[0]})")
        else:
            print("Sequence not found!")

finally:
    connection.close()
