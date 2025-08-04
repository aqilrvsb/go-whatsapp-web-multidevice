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
        # First, count how many we're about to delete
        cursor.execute("""
            SELECT COUNT(*) as count 
            FROM broadcast_messages 
            WHERE DATE(scheduled_at) = '2025-08-04' 
            AND status = 'pending'
        """)
        result = cursor.fetchone()
        count_to_delete = result['count']
        
        print(f"About to delete {count_to_delete} pending messages scheduled for August 4, 2025")
        print()
        
        # Delete the records
        cursor.execute("""
            DELETE FROM broadcast_messages 
            WHERE DATE(scheduled_at) = '2025-08-04' 
            AND status = 'pending'
        """)
        
        deleted_count = cursor.rowcount
        
        # Commit the deletion
        connection.commit()
        
        print(f"Successfully deleted {deleted_count} records")
        
        # Verify deletion
        cursor.execute("""
            SELECT COUNT(*) as remaining 
            FROM broadcast_messages 
            WHERE DATE(scheduled_at) = '2025-08-04' 
            AND status = 'pending'
        """)
        result = cursor.fetchone()
        
        print(f"Remaining pending messages for Aug 4: {result['remaining']}")
        
        # Check what's left for Aug 4
        cursor.execute("""
            SELECT status, COUNT(*) as count 
            FROM broadcast_messages 
            WHERE DATE(scheduled_at) = '2025-08-04' 
            GROUP BY status
        """)
        
        print("\nRemaining messages for August 4, 2025:")
        for row in cursor.fetchall():
            print(f"  {row['status']}: {row['count']}")
            
except Exception as e:
    print(f"Error: {e}")
    connection.rollback()
finally:
    connection.close()
