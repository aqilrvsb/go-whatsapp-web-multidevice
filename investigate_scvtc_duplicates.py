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
    print("\nINVESTIGATING SCVTC-S21 DUPLICATE ISSUE")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Check duplicates for SCVTC-S21 specifically
        print("\nChecking SCVTC-S21 pending messages...")
        
        cursor.execute("""
            SELECT 
                bm.recipient_phone,
                bm.recipient_name,
                COUNT(*) as count,
                GROUP_CONCAT(ss.day) as days,
                GROUP_CONCAT(ss.message_type) as types,
                GROUP_CONCAT(s.name) as sequences
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            LEFT JOIN sequences s ON s.id = ss.sequence_id
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCVTC-S21'
            GROUP BY bm.recipient_phone, bm.recipient_name
            HAVING COUNT(*) > 1
            ORDER BY COUNT(*) DESC
            LIMIT 20
        """)
        
        duplicates = cursor.fetchall()
        
        if duplicates:
            print(f"\nFound {len(duplicates)} phone numbers with multiple pending messages:")
            for dup in duplicates[:10]:
                print(f"\n  {dup['recipient_name'] or 'No name'} ({dup['recipient_phone']})")
                print(f"    Count: {dup['count']} messages")
                print(f"    Days: {dup['days']}")
                print(f"    Types: {dup['types']}")
                print(f"    Sequences: {dup['sequences']}")
        
        # The issue is: same phone is in MULTIPLE sequences or steps!
        # Let's check duplicates within the SAME sequence step
        print("\n\nChecking for duplicates within SAME sequence step...")
        
        cursor.execute("""
            SELECT 
                bm.sequence_stepid,
                bm.recipient_phone,
                COUNT(*) as duplicate_count,
                GROUP_CONCAT(bm.id) as message_ids,
                ss.day,
                s.name as sequence_name
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            LEFT JOIN sequences s ON s.id = ss.sequence_id
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCVTC-S21'
            GROUP BY bm.sequence_stepid, bm.recipient_phone
            HAVING COUNT(*) > 1
        """)
        
        real_duplicates = cursor.fetchall()
        
        if real_duplicates:
            print(f"\n❌ Found {len(real_duplicates)} REAL duplicates (same step, same phone):")
            
            # Delete these duplicates
            total_deleted = 0
            for dup in real_duplicates:
                ids = dup['message_ids'].split(',')
                delete_ids = ids[1:]  # Keep first, delete rest
                
                if delete_ids:
                    placeholders = ','.join(['%s'] * len(delete_ids))
                    cursor.execute(f"""
                        DELETE FROM broadcast_messages 
                        WHERE id IN ({placeholders})
                    """, delete_ids)
                    
                    total_deleted += len(delete_ids)
                    
                print(f"  Deleted {len(delete_ids)} duplicates for {dup['recipient_phone']} in {dup['sequence_name']} Day {dup['day']}")
            
            connection.commit()
            print(f"\n✅ Deleted {total_deleted} duplicate messages!")
        else:
            print("\n✅ No duplicates within same sequence step!")
            print("\nThe 'duplicates' you see are because:")
            print("- Same phone number is enrolled in MULTIPLE sequences")
            print("- OR same phone is in different steps of the same sequence")
            print("- This is NORMAL and expected behavior")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
