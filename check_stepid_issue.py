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
        
        print("Checking sequence_stepid population for WARM Sequence...")
        print("=" * 80)
        
        # Check how many messages have NULL sequence_stepid
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(sequence_stepid) as with_stepid,
                COUNT(*) - COUNT(sequence_stepid) as without_stepid
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"\nTotal messages: {result['total']}")
        print(f"With sequence_stepid: {result['with_stepid']}")
        print(f"Without sequence_stepid (NULL): {result['without_stepid']}")
        
        # Sample some messages to see their structure
        print("\nSample messages:")
        cursor.execute("""
            SELECT 
                id,
                sequence_id,
                sequence_stepid,
                recipient_phone,
                scheduled_at,
                status
            FROM broadcast_messages
            WHERE sequence_id = %s
            LIMIT 5
        """, (sequence_id,))
        
        messages = cursor.fetchall()
        for msg in messages:
            print(f"\nMessage ID: {msg['id'][:8]}...")
            print(f"  Sequence ID: {msg['sequence_id']}")
            print(f"  Step ID: {msg['sequence_stepid']}")
            print(f"  Phone: {msg['recipient_phone']}")
            print(f"  Scheduled: {msg['scheduled_at']}")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
