import pymysql
import sys
sys.stdout.reconfigure(encoding='utf-8')

# Connect to MySQL
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)
cursor = conn.cursor(pymysql.cursors.DictCursor)

print("=== CHECKING MESSAGES AND NAMES ===\n")

# Check recent messages
cursor.execute("""
    SELECT id, recipient_phone, recipient_name, content, 
           status, created_at, sent_at
    FROM broadcast_messages 
    WHERE recipient_phone = '60108924904'
    ORDER BY created_at DESC
    LIMIT 5
""")
messages = cursor.fetchall()

print("Recent messages to 60108924904:")
for msg in messages:
    print(f"\nMessage ID: {msg['id']}")
    print(f"  Recipient Name: '{msg['recipient_name']}'")
    print(f"  Content: '{msg['content']}'")
    print(f"  Status: {msg['status']}")
    print(f"  Created: {msg['created_at']}")
    
    # Check if this would be considered a phone number
    name = msg['recipient_name'] or ''
    digits_only = ''.join(c for c in name if c.isdigit())
    is_phone = len(digits_only) > 5
    print(f"  Name analysis: '{name}' -> {len(digits_only)} digits -> is_phone: {is_phone}")

print("\n✅ A new test message was just created!")
print("Check your WhatsApp in about 30 seconds to see if the greeting format is correct.")
print("\nExpected format:")
print("Hello Aqil 1,")
print("")
print("asdsad")

conn.close()
