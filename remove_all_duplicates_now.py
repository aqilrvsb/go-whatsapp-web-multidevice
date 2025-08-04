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
    print("\nFIXING DUPLICATE LEADS - REMOVING ALL DUPLICATES NOW")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Find ALL duplicate pending messages
        print("\nFinding ALL duplicate pending messages...")
        
        cursor.execute("""
            SELECT 
                device_id,
                sequence_stepid,
                recipient_phone,
                COUNT(*) as duplicate_count,
                GROUP_CONCAT(id ORDER BY created_at) as message_ids
            FROM broadcast_messages
            WHERE status = 'pending'
            GROUP BY device_id, sequence_stepid, recipient_phone
            HAVING COUNT(*) > 1
            ORDER BY duplicate_count DESC
        """)
        
        duplicates = cursor.fetchall()
        
        if duplicates:
            print(f"\n❌ Found {len(duplicates)} sets of duplicate messages")
            
            # Show examples
            print("\nExamples of duplicates found:")
            for dup in duplicates[:10]:
                print(f"  Phone: {dup['recipient_phone']} - {dup['duplicate_count']} copies")
            
            if len(duplicates) > 10:
                print(f"  ... and {len(duplicates) - 10} more sets")
            
            # Delete ALL duplicates
            total_deleted = 0
            print("\n\nDeleting duplicates (keeping oldest copy)...")
            
            for dup in duplicates:
                ids = dup['message_ids'].split(',')
                # Keep first (oldest), delete all others
                delete_ids = ids[1:]
                
                if delete_ids:
                    placeholders = ','.join(['%s'] * len(delete_ids))
                    cursor.execute(f"""
                        DELETE FROM broadcast_messages 
                        WHERE id IN ({placeholders})
                    """, delete_ids)
                    
                    total_deleted += len(delete_ids)
                
                if total_deleted % 100 == 0 and total_deleted > 0:
                    connection.commit()
                    print(f"  Deleted {total_deleted} messages...")
            
            connection.commit()
            print(f"\n✅ Successfully deleted {total_deleted} duplicate messages!")
        else:
            print("✅ No duplicate messages found!")
        
        # Verify the fix
        print("\n\nVerifying the fix...")
        
        # Check SCVTC-S21 Step 2 Day 2 specifically
        cursor.execute("""
            SELECT 
                ud.device_name,
                COUNT(DISTINCT bm.recipient_phone) as unique_leads,
                COUNT(*) as total_messages,
                GROUP_CONCAT(DISTINCT bm.recipient_phone) as phone_list
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCVTC-S21'
            AND ss.day = 2
            GROUP BY ud.device_name
        """)
        
        result = cursor.fetchone()
        
        if result:
            print(f"\nSCVTC-S21 Day 2 Status:")
            print(f"  Unique leads: {result['unique_leads']}")
            print(f"  Total messages: {result['total_messages']}")
            
            if result['unique_leads'] == result['total_messages']:
                print("  ✅ NO DUPLICATES! Count should now match displayed leads!")
            else:
                print("  ❌ Still has duplicates!")
        
        # Check overall status
        print("\n\nOverall Status:")
        cursor.execute("""
            SELECT 
                COUNT(*) as total_sets,
                SUM(duplicate_count) as total_duplicates
            FROM (
                SELECT 
                    device_id,
                    sequence_stepid,
                    recipient_phone,
                    COUNT(*) as duplicate_count
                FROM broadcast_messages
                WHERE status = 'pending'
                GROUP BY device_id, sequence_stepid, recipient_phone
                HAVING COUNT(*) > 1
            ) as dup_check
        """)
        
        final_check = cursor.fetchone()
        
        if final_check['total_sets'] == 0:
            print("✅ ALL DUPLICATES REMOVED! Counts should now match displayed leads!")
        else:
            print(f"❌ Still have {final_check['total_sets']} duplicate sets")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
