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
    print("\nFIXING THE COUNT CALCULATION FOR COLD Sequence EXSTART")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Get sequence
        cursor.execute("""
            SELECT id FROM sequences 
            WHERE name = 'COLD Sequence' AND niche = 'EXSTART'
        """)
        
        sequence = cursor.fetchone()
        sequence_id = sequence['id']
        
        # Get correct counts - count RECIPIENTS not messages
        print("\nCORRECT CALCULATION (by unique recipients):")
        
        # For "Should Send" - count unique recipients
        cursor.execute("""
            SELECT COUNT(DISTINCT recipient_phone) as should_send
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        should_send = result['should_send']
        print(f"\nContacts Should Send: {should_send}")
        
        # For "Done Send" - recipients who have at least one 'sent' message
        cursor.execute("""
            SELECT COUNT(DISTINCT recipient_phone) as done_send
            FROM broadcast_messages
            WHERE sequence_id = %s
            AND recipient_phone IN (
                SELECT DISTINCT recipient_phone
                FROM broadcast_messages
                WHERE sequence_id = %s
                AND status = 'sent'
            )
        """, (sequence_id, sequence_id))
        
        result = cursor.fetchone()
        done_send = result['done_send']
        print(f"Contacts Done Send Message: {done_send}")
        
        # For "Failed" - recipients who have ONLY failed messages (no sent)
        cursor.execute("""
            SELECT COUNT(DISTINCT recipient_phone) as failed_only
            FROM broadcast_messages bm1
            WHERE sequence_id = %s
            AND NOT EXISTS (
                SELECT 1 FROM broadcast_messages bm2
                WHERE bm2.sequence_id = %s
                AND bm2.recipient_phone = bm1.recipient_phone
                AND bm2.status = 'sent'
            )
            AND EXISTS (
                SELECT 1 FROM broadcast_messages bm3
                WHERE bm3.sequence_id = %s
                AND bm3.recipient_phone = bm1.recipient_phone
                AND bm3.status = 'failed'
            )
        """, (sequence_id, sequence_id, sequence_id))
        
        result = cursor.fetchone()
        failed_only = result['failed_only']
        print(f"Contacts Failed Send Message: {failed_only}")
        
        # Remaining = Should Send - Done Send
        remaining = should_send - done_send
        print(f"Contacts Remaining Send Message: {remaining}")
        
        print("\n" + "-" * 50)
        print("SUMMARY:")
        print(f"Should Send: {should_send}")
        print(f"Done: {done_send}")
        print(f"Failed (no success): {failed_only}")
        print(f"Remaining: {remaining}")
        
        # Verify the calculation
        print("\n\nVERIFICATION - Message breakdown:")
        cursor.execute("""
            SELECT 
                status,
                COUNT(*) as message_count,
                COUNT(DISTINCT recipient_phone) as unique_recipients
            FROM broadcast_messages
            WHERE sequence_id = %s
            GROUP BY status
        """, (sequence_id,))
        
        statuses = cursor.fetchall()
        for status in statuses:
            print(f"{status['status']}: {status['message_count']} messages, {status['unique_recipients']} unique recipients")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
