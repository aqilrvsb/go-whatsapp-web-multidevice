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
        # Check sequence steps structure
        print("=== SEQUENCE STEPS TABLE STRUCTURE ===")
        cursor.execute("DESCRIBE sequence_steps")
        columns = cursor.fetchall()
        for col in columns:
            print(f"{col[0]}: {col[1]}")
        
        # Check broadcast_messages structure for sequence fields
        print("\n=== BROADCAST_MESSAGES SEQUENCE FIELDS ===")
        cursor.execute("DESCRIBE broadcast_messages")
        columns = cursor.fetchall()
        for col in columns:
            if 'sequence' in col[0] or 'step' in col[0]:
                print(f"{col[0]}: {col[1]}")
        
        # Check sample sequence with steps
        print("\n=== SAMPLE SEQUENCE DATA ===")
        cursor.execute("""
            SELECT s.id, s.name, COUNT(ss.id) as step_count
            FROM sequences s
            LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
            WHERE s.name LIKE '%HOI%'
            GROUP BY s.id, s.name
            LIMIT 5
        """)
        sequences = cursor.fetchall()
        for seq in sequences:
            print(f"\nSequence {seq[0]}: {seq[1]} ({seq[2]} steps)")
            
            # Get steps for this sequence
            cursor.execute("""
                SELECT id, step_order, message_type, message_content
                FROM sequence_steps
                WHERE sequence_id = %s
                ORDER BY step_order
                LIMIT 5
            """, (seq[0],))
            steps = cursor.fetchall()
            for step in steps:
                print(f"  Step {step[1]}: {step[2]} - {step[3][:50]}...")
        
        # Check broadcast messages with sequence data
        print("\n=== BROADCAST MESSAGES WITH SEQUENCE ===")
        cursor.execute("""
            SELECT bm.sequence_id, bm.device_id, COUNT(*) as count,
                   GROUP_CONCAT(DISTINCT bm.status) as statuses
            FROM broadcast_messages bm
            WHERE bm.sequence_id IS NOT NULL
            GROUP BY bm.sequence_id, bm.device_id
            LIMIT 10
        """)
        results = cursor.fetchall()
        for r in results:
            print(f"Sequence {r[0]}, Device {r[1]}: {r[2]} messages (Status: {r[3]})")

finally:
    connection.close()
