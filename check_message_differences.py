import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== COMPARING CAMPAIGN VS SEQUENCE MESSAGES ===\n")

# Get a campaign message
cursor.execute("""
    SELECT * FROM broadcast_messages 
    WHERE campaign_id IS NOT NULL 
    ORDER BY created_at DESC 
    LIMIT 1
""")
campaign_msg = cursor.fetchone()

# Get a sequence message
cursor.execute("""
    SELECT * FROM broadcast_messages 
    WHERE sequence_id IS NOT NULL AND status = 'sent'
    ORDER BY created_at DESC 
    LIMIT 1
""")
sequence_msg = cursor.fetchone()

print("CAMPAIGN MESSAGE:")
print(f"  ID: {campaign_msg['id']}")
print(f"  Status: {campaign_msg['status']}")
print(f"  sequence_step_id: {campaign_msg['sequence_step_id']}")
print(f"  Type: {campaign_msg['message_type']}")
print(f"  Content: {campaign_msg['content'][:50]}...")
print(f"  Scheduled At: {campaign_msg['scheduled_at']}")

print("\nSEQUENCE MESSAGE (that was sent):")
if sequence_msg:
    print(f"  ID: {sequence_msg['id']}")
    print(f"  Status: {sequence_msg['status']}")
    print(f"  sequence_step_id: {sequence_msg['sequence_step_id']}")
    print(f"  Type: {sequence_msg['message_type']}")
    print(f"  Scheduled At: {sequence_msg['scheduled_at']}")

print("\n=== KEY DIFFERENCES ===")
print(f"Campaign sequence_step_id: {campaign_msg['sequence_step_id']}")
print(f"Sequence sequence_step_id: {sequence_msg['sequence_step_id'] if sequence_msg else 'N/A'}")

# Check what processor picks up messages
print("\n=== CHECKING MESSAGE PROCESSORS ===")
cursor.execute("""
    SELECT COUNT(*) as pending_campaigns
    FROM broadcast_messages 
    WHERE campaign_id IS NOT NULL 
    AND status = 'pending'
    AND scheduled_at <= NOW()
""")
result = cursor.fetchone()
print(f"Campaign messages ready to send: {result['pending_campaigns']}")

# Check if broadcast processor is looking for campaign messages
print("\nThe broadcast processor might be filtering by sequence_id NOT NULL")
print("This would explain why campaign messages aren't being picked up!")

conn.close()
