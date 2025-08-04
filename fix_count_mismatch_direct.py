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
    print("\nFIXING COUNT MISMATCH ISSUE DIRECTLY")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Step 1: Update NULL day values based on day_number
        print("\nStep 1: Fixing NULL day values in sequence_steps...")
        
        cursor.execute("""
            UPDATE sequence_steps
            SET day = day_number
            WHERE day IS NULL
            AND day_number IS NOT NULL
        """)
        
        rows_updated = cursor.rowcount
        connection.commit()
        
        if rows_updated > 0:
            print(f"✅ Updated {rows_updated} sequence steps with proper day values")
        else:
            print("✅ No NULL day values found to update")
        
        # Step 2: Remove ALL duplicate pending messages
        print("\n\nStep 2: Removing duplicate pending messages...")
        
        # Find duplicates
        cursor.execute("""
            SELECT 
                device_id,
                IFNULL(sequence_stepid, 'NULL_STEP') as sequence_stepid,
                recipient_phone,
                COUNT(*) as duplicate_count,
                GROUP_CONCAT(id ORDER BY created_at) as message_ids
            FROM broadcast_messages
            WHERE status = 'pending'
            GROUP BY device_id, IFNULL(sequence_stepid, 'NULL_STEP'), recipient_phone
            HAVING COUNT(*) > 1
        """)
        
        duplicates = cursor.fetchall()
        
        if duplicates:
            print(f"\nFound {len(duplicates)} sets of duplicate messages")
            
            total_deleted = 0
            for dup in duplicates:
                ids = dup['message_ids'].split(',')
                # Keep first (oldest), delete rest
                delete_ids = ids[1:]
                
                if delete_ids:
                    placeholders = ','.join(['%s'] * len(delete_ids))
                    cursor.execute(f"""
                        DELETE FROM broadcast_messages 
                        WHERE id IN ({placeholders})
                    """, delete_ids)
                    
                    total_deleted += len(delete_ids)
            
            connection.commit()
            print(f"✅ Deleted {total_deleted} duplicate messages")
        else:
            print("✅ No duplicate messages found")
        
        # Step 3: Check specific device (SCAS-S74) status
        print("\n\nStep 3: Checking SCAS-S74 device status...")
        
        cursor.execute("""
            SELECT 
                ss.day,
                ss.day_number,
                ss.message_type,
                COUNT(DISTINCT bm.recipient_phone) as unique_recipients,
                COUNT(*) as total_messages,
                ss.id as step_id
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCAS-S74'
            GROUP BY ss.id, ss.day, ss.day_number, ss.message_type
            ORDER BY COALESCE(ss.day, ss.day_number, 999)
        """)
        
        results = cursor.fetchall()
        
        if results:
            print("\nSCAS-S74 pending messages by step:")
            for r in results:
                day_info = f"Day {r['day'] or r['day_number'] or 'Unknown'}"
                print(f"\n  {day_info} - {r['message_type'] or 'Unknown type'}")
                print(f"    Step ID: {r['step_id'] or 'No step'}")
                print(f"    Unique recipients: {r['unique_recipients']}")
                print(f"    Total messages: {r['total_messages']}")
                
                if r['unique_recipients'] == r['total_messages']:
                    print(f"    ✅ Count matches!")
                else:
                    print(f"    ❌ Has duplicates!")
        
        # Step 4: Fix orphaned messages (no sequence_stepid)
        print("\n\nStep 4: Checking for orphaned messages...")
        
        cursor.execute("""
            SELECT 
                COUNT(*) as count,
                COUNT(DISTINCT device_id) as devices,
                COUNT(DISTINCT recipient_phone) as unique_recipients
            FROM broadcast_messages
            WHERE status = 'pending'
            AND sequence_stepid IS NULL
        """)
        
        orphaned = cursor.fetchone()
        
        if orphaned['count'] > 0:
            print(f"\n⚠️  Found {orphaned['count']} orphaned messages")
            print(f"   Across {orphaned['devices']} devices")
            print(f"   {orphaned['unique_recipients']} unique recipients")
            
            # Delete orphaned messages
            response = input("\nDelete these orphaned messages? (yes/no): ")
            if response.lower() == 'yes':
                cursor.execute("""
                    DELETE FROM broadcast_messages
                    WHERE status = 'pending'
                    AND sequence_stepid IS NULL
                """)
                
                deleted = cursor.rowcount
                connection.commit()
                print(f"✅ Deleted {deleted} orphaned messages")
        else:
            print("✅ No orphaned messages found")
        
        # Final verification
        print("\n\n" + "=" * 100)
        print("FINAL VERIFICATION")
        print("=" * 100)
        
        cursor.execute("""
            SELECT 
                ud.device_name,
                COUNT(DISTINCT bm.recipient_phone) as unique_pending,
                COUNT(bm.id) as total_pending
            FROM user_devices ud
            LEFT JOIN broadcast_messages bm ON bm.device_id = ud.id AND bm.status = 'pending'
            WHERE ud.device_name IN ('SCAS-S74', 'SCAST-S30', 'SCARS-S46', 'SCRY-S08')
            GROUP BY ud.device_name
            ORDER BY ud.device_name
        """)
        
        final_results = cursor.fetchall()
        
        print("\nDevice Status Summary:")
        all_match = True
        for r in final_results:
            status = "✅ Match" if r['unique_pending'] == r['total_pending'] else "❌ Mismatch"
            print(f"{r['device_name']}: {r['unique_pending']} unique / {r['total_pending']} total - {status}")
            if r['unique_pending'] != r['total_pending']:
                all_match = False
        
        if all_match:
            print("\n✅ ALL COUNTS NOW MATCH! The display should show correct counts.")
        else:
            print("\n⚠️  Some devices still have mismatches. May need manual investigation.")
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
