import pymysql
import os
from datetime import datetime

# Get MySQL connection from environment
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
        print("=== DEBUGGING SEQUENCE TOTAL LEADS ISSUE ===\n")
        
        # First, find the COLD Sequence ID
        cursor.execute("""
            SELECT id, name, user_id 
            FROM sequences 
            WHERE name = 'COLD Sequence' 
            LIMIT 1
        """)
        sequence = cursor.fetchone()
        
        if not sequence:
            print("ERROR: Could not find COLD Sequence!")
            exit(1)
            
        sequence_id = sequence['id']
        user_id = sequence['user_id']
        print(f"Found COLD Sequence: ID={sequence_id}, User={user_id}\n")
        
        # Check total messages
        cursor.execute("""
            SELECT COUNT(*) as total_messages
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        result = cursor.fetchone()
        print(f"1. Total messages in sequence: {result['total_messages']}")
        
        # Check for NULL values
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                SUM(CASE WHEN recipient_phone IS NULL THEN 1 ELSE 0 END) as null_phones,
                SUM(CASE WHEN device_id IS NULL THEN 1 ELSE 0 END) as null_devices,
                SUM(CASE WHEN recipient_phone = '' THEN 1 ELSE 0 END) as empty_phones,
                SUM(CASE WHEN device_id = '' THEN 1 ELSE 0 END) as empty_devices
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        null_check = cursor.fetchone()
        print(f"\n2. NULL/Empty value check:")
        print(f"   - NULL phones: {null_check['null_phones']}")
        print(f"   - NULL devices: {null_check['null_devices']}")
        print(f"   - Empty phones: {null_check['empty_phones']}")
        print(f"   - Empty devices: {null_check['empty_devices']}")
        
        # Check unique leads calculation
        cursor.execute("""
            SELECT 
                COUNT(DISTINCT recipient_phone) as unique_phones,
                COUNT(DISTINCT device_id) as unique_devices,
                COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as unique_combinations
            FROM broadcast_messages
            WHERE sequence_id = %s
        """, (sequence_id,))
        unique_counts = cursor.fetchone()
        print(f"\n3. Unique counts:")
        print(f"   - Unique phones: {unique_counts['unique_phones']}")
        print(f"   - Unique devices: {unique_counts['unique_devices']}")
        print(f"   - Unique phone+device combinations: {unique_counts['unique_combinations']}")
        
        # Sample some actual data
        cursor.execute("""
            SELECT 
                recipient_phone,
                device_id,
                sequence_stepid,
                status
            FROM broadcast_messages
            WHERE sequence_id = %s
            LIMIT 5
        """, (sequence_id,))
        samples = cursor.fetchall()
        print(f"\n4. Sample data (first 5 records):")
        for sample in samples:
            print(f"   Phone: {sample['recipient_phone']}, Device: {sample['device_id']}, Step: {sample['sequence_stepid']}, Status: {sample['status']}")
        
        # Check if CONCAT is working properly
        cursor.execute("""
            SELECT 
                recipient_phone,
                device_id,
                CONCAT(recipient_phone, '|', device_id) as concatenated
            FROM broadcast_messages
            WHERE sequence_id = %s
            LIMIT 3
        """, (sequence_id,))
        concat_test = cursor.fetchall()
        print(f"\n5. Testing CONCAT function:")
        for test in concat_test:
            print(f"   Phone: '{test['recipient_phone']}', Device: '{test['device_id']}', Concat: '{test['concatenated']}'")
        
        # Check step-wise statistics
        cursor.execute("""
            SELECT 
                sequence_stepid,
                COUNT(*) as messages,
                COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as unique_leads
            FROM broadcast_messages
            WHERE sequence_id = %s
            GROUP BY sequence_stepid
        """, (sequence_id,))
        step_stats = cursor.fetchall()
        print(f"\n6. Step-wise statistics:")
        for step in step_stats:
            print(f"   Step {step['sequence_stepid']}: {step['messages']} messages, {step['unique_leads']} unique leads")
            
        # Check if it's a date filtering issue
        cursor.execute("""
            SELECT 
                DATE(scheduled_at) as date,
                COUNT(*) as messages,
                COUNT(DISTINCT CONCAT(recipient_phone, '|', device_id)) as unique_leads
            FROM broadcast_messages
            WHERE sequence_id = %s
            GROUP BY DATE(scheduled_at)
            ORDER BY date DESC
            LIMIT 5
        """, (sequence_id,))
        date_stats = cursor.fetchall()
        print(f"\n7. Date-wise breakdown (last 5 dates):")
        for date_stat in date_stats:
            print(f"   {date_stat['date']}: {date_stat['messages']} messages, {date_stat['unique_leads']} unique leads")

except Exception as e:
    print(f"Error: {e}")
finally:
    connection.close()
