import pymysql
import datetime

# Database connection
connection = pymysql.connect(
    host='159.89.198.71',
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    port=3306,
    charset='utf8mb4',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

print("="*60)
print("WhatsApp Broadcast System - Fix Failed Messages")
print("Date: 24/09/2025")
print("="*60)

# Step 1: Check current status of messages for today
print("\n1. Checking today's message status...")
cursor.execute("""
    SELECT status, COUNT(*) as count 
    FROM broadcast_messages 
    WHERE DATE(created_at) = '2025-09-24' 
    GROUP BY status
""")
results = cursor.fetchall()
print("\nCurrent Status Summary:")
for row in results:
    print(f"  - {row['status']}: {row['count']} messages")

# Step 2: Get details of failed messages
print("\n2. Getting details of failed messages...")
cursor.execute("""
    SELECT id, recipient_phone, device_id, error_message, created_at
    FROM broadcast_messages 
    WHERE DATE(created_at) = '2025-09-24' 
    AND status = 'failed'
    LIMIT 10
""")
failed_messages = cursor.fetchall()
print(f"\nFound {cursor.rowcount} failed messages (showing first 10):")
for msg in failed_messages[:5]:
    print(f"  - ID: {msg['id']}, Phone: {msg['recipient_phone']}, Error: {msg['error_message'][:50] if msg['error_message'] else 'No error message'}")

# Step 3: Revert failed messages to pending
print("\n3. Reverting failed messages to pending status...")
cursor.execute("""
    UPDATE broadcast_messages 
    SET status = 'pending',
        error_message = NULL,
        processing_started_at = NULL,
        processing_worker_id = NULL,
        sent_at = NULL,
        scheduled_at = NOW()
    WHERE DATE(created_at) = '2025-09-24' 
    AND status = 'failed'
""")
updated_count = cursor.rowcount
connection.commit()
print(f"✅ Successfully reverted {updated_count} failed messages to pending status")

# Step 4: Verify the update
print("\n4. Verifying the update...")
cursor.execute("""
    SELECT status, COUNT(*) as count 
    FROM broadcast_messages 
    WHERE DATE(created_at) = '2025-09-24' 
    GROUP BY status
""")
results = cursor.fetchall()
print("\nUpdated Status Summary:")
for row in results:
    print(f"  - {row['status']}: {row['count']} messages")

print("\n" + "="*60)
print("✅ PART 1 COMPLETE: Failed messages reverted to pending")
print("="*60)

cursor.close()
connection.close()
