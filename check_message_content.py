import pymysql
import sys

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

# Connect to MySQL
connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

# Check a sample of pending messages
query = """
SELECT 
    id,
    recipient_phone,
    recipient_name,
    content,
    message_type,
    status,
    device_id
FROM broadcast_messages
WHERE status = 'pending'
AND content IS NOT NULL
LIMIT 5
"""

cursor.execute(query)
results = cursor.fetchall()

print("=== CHECKING PENDING MESSAGES ===")
print(f"Found {len(results)} pending messages\n")

for i, msg in enumerate(results):
    print(f"--- Message {i+1} ---")
    print(f"ID: {msg['id']}")
    print(f"Phone: {msg['recipient_phone']}")
    print(f"Recipient Name: '{msg['recipient_name']}'")
    print(f"Type: {msg['message_type']}")
    print(f"Device: {msg['device_id']}")
    print(f"\nContent (raw):")
    print(repr(msg['content']))  # Using repr to see escape characters
    print(f"\nContent (formatted):")
    print(msg['content'])
    print("\n" + "="*50 + "\n")

# Check if recipient_name is populated in sequences
print("\n=== CHECKING SEQUENCE MESSAGES ===")
seq_query = """
SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN recipient_name IS NULL OR recipient_name = '' THEN 1 ELSE 0 END) as missing_names,
    SUM(CASE WHEN recipient_name = recipient_phone THEN 1 ELSE 0 END) as phone_as_name
FROM broadcast_messages
WHERE sequence_id IS NOT NULL
AND status = 'pending'
"""

cursor.execute(seq_query)
stats = cursor.fetchone()

print(f"Total sequence messages: {stats['total']}")
print(f"Missing recipient names: {stats['missing_names']}")
print(f"Phone number as name: {stats['phone_as_name']}")

cursor.close()
connection.close()
