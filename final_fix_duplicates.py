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
    print("\nFINAL FIX: ENSURE NO DUPLICATES FOR DEVICE+STEP+PHONE")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Find ALL duplicates for same device+step+phone
        print("\nFinding duplicates for same device+step+phone combination...")
        
        cursor.execute("""
            SELECT 
                device_id,
                sequence_stepid,
                recipient_phone,
                COUNT(*) as duplicate_count,
                GROUP_CONCAT(id ORDER BY created_at) as message_ids,
                GROUP_CONCAT(status) as statuses
            FROM broadcast_messages
            WHERE sequence_stepid IS NOT NULL
            GROUP BY device_id, sequence_stepid, recipient_phone
            HAVING COUNT(*) > 1
            ORDER BY duplicate_count DESC
        """)
        
        duplicates = cursor.fetchall()
        
        if duplicates:
            print(f"\n❌ Found {len(duplicates)} duplicate combinations")
            
            # Show examples
            print("\nExamples:")
            for dup in duplicates[:5]:
                print(f"  Phone: {dup['recipient_phone']} - {dup['duplicate_count']} duplicates - Statuses: {dup['statuses']}")
            
            # Delete duplicates (keep oldest)
            total_deleted = 0
            print("\n\nDeleting duplicates...")
            
            for dup in duplicates:
                ids = dup['message_ids'].split(',')
                delete_ids = ids[1:]  # Keep first, delete rest
                
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
            
            # Add unique constraint to prevent future duplicates
            print("\n\nAdding unique constraint to prevent future duplicates...")
            try:
                cursor.execute("""
                    ALTER TABLE broadcast_messages 
                    ADD UNIQUE KEY uk_device_step_phone (device_id, sequence_stepid, recipient_phone)
                """)
                connection.commit()
                print("✅ Unique constraint added successfully!")
            except Exception as e:
                if "Duplicate key name" in str(e):
                    print("✅ Unique constraint already exists!")
                else:
                    print(f"⚠️  Could not add constraint: {e}")
        else:
            print("✅ No duplicates found!")
        
        # Verify the fix
        print("\n\nVerifying the fix...")
        
        # Check a few devices
        cursor.execute("""
            SELECT 
                ud.device_name,
                COUNT(DISTINCT bm.id) as total_messages,
                COUNT(DISTINCT CONCAT(bm.device_id, '|', bm.sequence_stepid, '|', bm.recipient_phone)) as unique_combinations
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.status = 'pending'
            AND bm.sequence_stepid IS NOT NULL
            AND ud.device_name IN ('SCVTC-S21', 'SCAS-S74', 'SCAST-S30')
            GROUP BY ud.device_name
        """)
        
        results = cursor.fetchall()
        
        print("\nDevice verification:")
        all_good = True
        for r in results:
            if r['total_messages'] == r['unique_combinations']:
                print(f"  {r['device_name']}: ✅ No duplicates ({r['total_messages']} messages)")
            else:
                print(f"  {r['device_name']}: ❌ Has duplicates ({r['total_messages']} messages, {r['unique_combinations']} unique)")
                all_good = False
        
        if all_good:
            print("\n✅ ALL FIXED! Counts should now match displayed leads!")
            
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
