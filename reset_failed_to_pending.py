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
        # First, count how many failed messages we have from Aug 5 onwards
        cursor.execute("""
            SELECT 
                DATE(scheduled_at) as scheduled_date,
                COUNT(*) as count
            FROM broadcast_messages 
            WHERE scheduled_at >= '2025-08-05 00:00:00'
            AND status = 'failed'
            GROUP BY DATE(scheduled_at)
            ORDER BY scheduled_date
        """)
        
        failed_by_date = cursor.fetchall()
        
        print("Failed messages from August 5 onwards:")
        total_to_update = 0
        for row in failed_by_date:
            print(f"  {row['scheduled_date']}: {row['count']} messages")
            total_to_update += row['count']
            
        print(f"\nTotal failed messages to update: {total_to_update}")
        print()
        
        if total_to_update > 0:
            # Update failed messages back to pending
            print("Updating failed messages to pending status...")
            
            cursor.execute("""
                UPDATE broadcast_messages 
                SET 
                    status = 'pending',
                    error_message = NULL,
                    sent_at = NULL
                WHERE scheduled_at >= '2025-08-05 00:00:00'
                AND status = 'failed'
            """)
            
            updated_count = cursor.rowcount
            
            # Commit the changes
            connection.commit()
            
            print(f"Successfully updated {updated_count} messages from 'failed' to 'pending'")
            
            # Verify the update
            print("\nVerifying update - checking status counts from Aug 5 onwards:")
            cursor.execute("""
                SELECT 
                    DATE(scheduled_at) as scheduled_date,
                    status,
                    COUNT(*) as count
                FROM broadcast_messages 
                WHERE scheduled_at >= '2025-08-05 00:00:00'
                GROUP BY DATE(scheduled_at), status
                ORDER BY scheduled_date, status
            """)
            
            results = cursor.fetchall()
            
            current_date = None
            for row in results:
                if current_date != row['scheduled_date']:
                    if current_date:
                        print()
                    print(f"\n{row['scheduled_date']}:")
                    current_date = row['scheduled_date']
                print(f"  {row['status']}: {row['count']}")
                
        else:
            print("No failed messages found to update.")
            
except Exception as e:
    print(f"Error: {e}")
    connection.rollback()
finally:
    connection.close()
