import pymysql
from datetime import datetime
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Parse MySQL URI
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')
uri_parts = mysql_uri.replace('mysql://', '').split('@')
user_pass = uri_parts[0].split(':')
host_db = uri_parts[1].split('/')
host_port = host_db[0].split(':')

# Connect to MySQL
connection = pymysql.connect(
    host=host_port[0],
    port=int(host_port[1]),
    user=user_pass[0],
    password=user_pass[1],
    database=host_db[1],
    charset='utf8mb4'
)

try:
    with connection.cursor() as cursor:
        # Test 1: Check sequences
        print("=== SEQUENCES ===")
        cursor.execute("SELECT id, name, niche, status, total_contacts FROM sequences LIMIT 5")
        sequences = cursor.fetchall()
        for seq in sequences:
            print(f"Sequence {seq[0]}: {seq[1]} - Niche: {seq[2]}, Status: {seq[3]}, Contacts: {seq[4]}")
        
        # Test 2: Check sequence_contacts
        print("\n=== SEQUENCE CONTACTS ===")
        cursor.execute("""
            SELECT sequence_id, COUNT(*) as contact_count, 
                   COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
                   COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed
            FROM sequence_contacts 
            GROUP BY sequence_id
        """)
        seq_contacts = cursor.fetchall()
        for seq_id, count, active, completed in seq_contacts:
            print(f"Sequence {seq_id}: {count} contacts (Active: {active}, Completed: {completed})")
        
        # Test 3: Check sequence_steps
        print("\n=== SEQUENCE STEPS ===")
        cursor.execute("""
            SELECT sequence_id, COUNT(*) as step_count 
            FROM sequence_steps 
            GROUP BY sequence_id
        """)
        seq_steps = cursor.fetchall()
        for seq_id, count in seq_steps:
            print(f"Sequence {seq_id}: {count} steps")
        
        # Test 4: Check sequence_contact statuses
        print("\n=== SEQUENCE CONTACT STATUSES ===")
        cursor.execute("""
            SELECT status, COUNT(*) as count 
            FROM sequence_contacts 
            GROUP BY status
        """)
        statuses = cursor.fetchall()
        for status, count in statuses:
            print(f"Status '{status}': {count} contacts")
        
        # Test 5: Check broadcast messages for sequences
        print("\n=== SEQUENCE BROADCAST MESSAGES ===")
        cursor.execute("""
            SELECT 
                sequence_id,
                COUNT(*) as total,
                COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
                COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending
            FROM broadcast_messages 
            WHERE sequence_id IS NOT NULL
            GROUP BY sequence_id
        """)
        seq_broadcasts = cursor.fetchall()
        for seq_id, total, success, failed, pending in seq_broadcasts:
            print(f"Sequence {seq_id}: Total={total}, Success={success}, Failed={failed}, Pending={pending}")
        
        # Test 6: Check if there are any sequence flows in sequence_steps
        print("\n=== SEQUENCE FLOWS COUNT ===")
        cursor.execute("SELECT COUNT(*) FROM sequence_steps")
        total_flows = cursor.fetchone()[0]
        print(f"Total sequence flows: {total_flows}")
        
        # Test 7: Check actual broadcast message statuses
        print("\n=== ALL BROADCAST MESSAGE STATUSES ===")
        cursor.execute("""
            SELECT status, COUNT(*) as count 
            FROM broadcast_messages 
            GROUP BY status
        """)
        all_statuses = cursor.fetchall()
        for status, count in all_statuses:
            print(f"Status '{status}': {count} messages")

finally:
    connection.close()
