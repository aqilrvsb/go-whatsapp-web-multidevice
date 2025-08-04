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
        # Find COLD Sequence
        cursor.execute("SELECT id FROM sequences WHERE name = 'COLD Sequence' LIMIT 1")
        sequence = cursor.fetchone()
        
        if sequence:
            sequence_id = sequence['id']
            print(f"Testing query for sequence: {sequence_id}")
            
            # Test the exact query used in device report
            query = """
                SELECT 
                    COUNT(DISTINCT CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id)) as total,
                    COUNT(DISTINCT CASE WHEN status = 'sent' AND (error_message IS NULL OR error_message = '') 
                        THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as done_send,
                    COUNT(DISTINCT CASE WHEN status = 'failed' 
                        THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as failed_send,
                    COUNT(DISTINCT CASE WHEN status IN ('pending', 'queued') 
                        THEN CONCAT(sequence_stepid, '|', recipient_phone, '|', device_id) END) as remaining_send,
                    COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
                FROM broadcast_messages
                WHERE sequence_id = %s
                AND DATE(scheduled_at) = '2025-08-03'
            """
            
            cursor.execute(query, (sequence_id,))
            result = cursor.fetchone()
            
            print("\nQuery result:")
            print(f"Total messages: {result['total']}")
            print(f"Done send: {result['done_send']}")
            print(f"Failed send: {result['failed_send']}")
            print(f"Remaining send: {result['remaining_send']}")
            print(f"Total leads: {result['total_leads']}")
            
            # Also test without date filter
            query_no_date = """
                SELECT COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as total_leads
                FROM broadcast_messages
                WHERE sequence_id = %s
            """
            
            cursor.execute(query_no_date, (sequence_id,))
            result_no_date = cursor.fetchone()
            print(f"\nTotal leads without date filter: {result_no_date['total_leads']}")
            
except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
