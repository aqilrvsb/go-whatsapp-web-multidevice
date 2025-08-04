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
    print("\nFINDING WHERE 278 LEADS COUNT COMES FROM")
    print("=" * 100)
    
    with connection.cursor() as cursor:
        # Get COLD Sequence EXSTART
        cursor.execute("""
            SELECT id, name, niche 
            FROM sequences 
            WHERE name = 'COLD Sequence' 
            AND niche = 'EXSTART'
        """)
        
        sequence = cursor.fetchone()
        sequence_id = sequence['id']
        
        print(f"\nSequence: {sequence['name']} - {sequence['niche']}")
        print(f"ID: {sequence_id}")
        
        # Check different counts
        print("\n\n1. LEADS TABLE - Leads with trigger 'cold_start' and niche 'EXSTART':")
        cursor.execute("""
            SELECT COUNT(*) as count
            FROM leads
            WHERE niche = 'EXSTART'
            AND target_status = 'cold'
        """)
        
        result = cursor.fetchone()
        print(f"   Count: {result['count']}")
        
        # Check sequence_contacts
        print("\n2. SEQUENCE_CONTACTS - Enrolled contacts:")
        cursor.execute("""
            SELECT COUNT(DISTINCT contact_phone) as count
            FROM sequence_contacts
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"   Count: {result['count']}")
        
        # Check broadcast_messages unique recipients
        print("\n3. BROADCAST_MESSAGES - Unique recipients:")
        cursor.execute("""
            SELECT COUNT(DISTINCT recipient_phone) as count
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        result = cursor.fetchone()
        print(f"   Count: {result['count']}")
        
        # Skip checking contacts_count since it doesn't exist
        print("\n4. Checking if 278 might be total leads in system:")
        
        # Check all sequences for this niche
        print("\n5. ALL SEQUENCES FOR EXSTART NICHE:")
        cursor.execute("""
            SELECT 
                s.name,
                s.trigger,
                COUNT(DISTINCT sc.contact_phone) as enrolled,
                COUNT(DISTINCT bm.recipient_phone) as has_messages
            FROM sequences s
            LEFT JOIN sequence_contacts sc ON sc.sequence_id = s.id
            LEFT JOIN broadcast_messages bm ON bm.sequence_id = s.id
            WHERE s.niche = 'EXSTART'
            GROUP BY s.id, s.name, s.trigger
        """)
        
        all_sequences = cursor.fetchall()
        
        total_enrolled = 0
        total_messages = 0
        
        for seq in all_sequences:
            print(f"   {seq['name']} ({seq['trigger']}): {seq['enrolled']} enrolled, {seq['has_messages']} with messages")
            total_enrolled += seq['enrolled']
            total_messages += seq['has_messages']
        
        print(f"\n   TOTALS: {total_enrolled} enrolled, {total_messages} with messages")
        
        if total_messages == 278:
            print("\n   ✅ FOUND IT! The 278 is the TOTAL across ALL sequences for EXSTART niche!")
        
        # Verify the calculation shown in device report
        print("\n\n6. DEVICE REPORT CALCULATION (Should Send = 250):")
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT CASE WHEN status = 'sent' THEN recipient_phone END) as done,
                COUNT(DISTINCT CASE WHEN status = 'failed' THEN recipient_phone END) as failed,
                COUNT(DISTINCT CASE WHEN status = 'pending' THEN recipient_phone END) as pending,
                COUNT(DISTINCT recipient_phone) as total_unique
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        
        stats = cursor.fetchone()
        print(f"   Total unique recipients: {stats['total_unique']}")
        print(f"   Done (sent): {stats['done']}")
        print(f"   Failed: {stats['failed']}")
        print(f"   Pending: {stats['pending']}")
        print(f"\n   Note: A recipient can be in multiple statuses if they have multiple step messages")
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
