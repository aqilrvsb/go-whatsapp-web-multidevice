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
    print("\nUNDERSTANDING THE EXACT ISSUE")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Find what the UI might be showing as "Step 1: Day 4"
        print("\nLooking for SCAS-S74 messages that could be 'Step 1: Day 4'...\n")
        
        # Check all pending messages for SCAS-S74
        cursor.execute("""
            SELECT 
                bm.recipient_phone,
                bm.recipient_name,
                ss.day,
                ss.day_number,
                s.name as sequence_name,
                ss.message_type
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            LEFT JOIN sequences s ON s.id = ss.sequence_id
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCAS-S74'
            AND (ss.day = 4 OR ss.day_number = 4)
            ORDER BY bm.recipient_phone
        """)
        
        day4_messages = cursor.fetchall()
        
        print(f"Found {len(day4_messages)} pending messages for Day 4:\n")
        
        # Group by sequence
        sequences = {}
        for msg in day4_messages:
            seq = msg['sequence_name'] or 'Unknown'
            if seq not in sequences:
                sequences[seq] = []
            sequences[seq].append(msg)
        
        for seq, msgs in sequences.items():
            print(f"\n{seq}:")
            unique_phones = set()
            for msg in msgs:
                print(f"  - {msg['recipient_name'] or 'No name'} ({msg['recipient_phone']}) - {msg['message_type']}")
                unique_phones.add(msg['recipient_phone'])
            print(f"  Total: {len(msgs)} messages, {len(unique_phones)} unique phones")
        
        # Now check what might show as "remaining 4"
        print("\n\n" + "=" * 80)
        print("CHECKING WHAT MIGHT SHOW AS 'REMAINING 4'")
        print("=" * 80)
        
        # This might be counting unique phones across all Day 4 steps
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT bm.recipient_phone) as unique_count,
                COUNT(*) as total_count
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.status = 'pending'
            AND ud.device_name = 'SCAS-S74'
            AND (ss.day = 4 OR ss.day_number = 4)
        """)
        
        result = cursor.fetchone()
        print(f"\nDay 4 totals: {result['unique_count']} unique phones, {result['total_count']} total messages")
        
        if result['unique_count'] == 4:
            print("\n✅ This matches the '4 remaining' shown in UI!")
            print("The issue is: UI counts UNIQUE phones but displays ALL messages")
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
