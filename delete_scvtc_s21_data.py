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
        # First, find the device ID for SCVTC-S21
        print("Finding device SCVTC-S21...")
        cursor.execute("SELECT id, device_name FROM user_devices WHERE device_name = 'SCVTC-S21'")
        device = cursor.fetchone()
        
        if not device:
            print("ERROR: Device SCVTC-S21 not found!")
            exit(1)
            
        device_id = device['id']
        print(f"Found device: {device['device_name']} with ID: {device_id}")
        
        # Get counts before deletion
        print("\n=== CHECKING CURRENT DATA ===")
        
        # Count leads
        cursor.execute("SELECT COUNT(*) as count FROM leads WHERE device_id = %s", (device_id,))
        lead_count = cursor.fetchone()['count']
        print(f"Leads to delete: {lead_count}")
        
        # Count broadcast messages
        cursor.execute("SELECT COUNT(*) as count FROM broadcast_messages WHERE device_id = %s", (device_id,))
        broadcast_count = cursor.fetchone()['count']
        print(f"Broadcast messages to delete: {broadcast_count}")
        
        if lead_count == 0 and broadcast_count == 0:
            print("\nNo data to delete for this device.")
            exit(0)
        
        # Confirm deletion
        print(f"\nWARNING: This will permanently delete:")
        print(f"   - {lead_count} leads")
        print(f"   - {broadcast_count} broadcast messages")
        print(f"   For device: SCVTC-S21 ({device_id})")
        
        confirm = input("\nType 'DELETE' to confirm deletion: ")
        
        if confirm != 'DELETE':
            print("Deletion cancelled.")
            exit(0)
        
        # Start transaction
        connection.begin()
        
        try:
            # Delete leads
            print("\nDeleting leads...")
            cursor.execute("DELETE FROM leads WHERE device_id = %s", (device_id,))
            deleted_leads = cursor.rowcount
            print(f"Deleted {deleted_leads} leads")
            
            # Delete broadcast messages
            print("\nDeleting broadcast messages...")
            cursor.execute("DELETE FROM broadcast_messages WHERE device_id = %s", (device_id,))
            deleted_broadcasts = cursor.rowcount
            print(f"Deleted {deleted_broadcasts} broadcast messages")
            
            # Commit transaction
            connection.commit()
            print("\nDeletion completed successfully!")
            
            # Verify deletion
            print("\n=== VERIFICATION ===")
            cursor.execute("SELECT COUNT(*) as count FROM leads WHERE device_id = %s", (device_id,))
            remaining_leads = cursor.fetchone()['count']
            
            cursor.execute("SELECT COUNT(*) as count FROM broadcast_messages WHERE device_id = %s", (device_id,))
            remaining_broadcasts = cursor.fetchone()['count']
            
            print(f"Remaining leads: {remaining_leads}")
            print(f"Remaining broadcast messages: {remaining_broadcasts}")
            
        except Exception as e:
            # Rollback on error
            connection.rollback()
            print(f"\nError during deletion: {e}")
            print("Transaction rolled back - no data was deleted.")
            
except Exception as e:
    print(f"Connection error: {e}")
finally:
    connection.close()
