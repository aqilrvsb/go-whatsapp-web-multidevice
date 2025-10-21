import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== CHECKING BROADCAST MESSAGES FOR CAMPAIGN 70 ===\n")

# Check ALL broadcast messages for campaign 70
cursor.execute("""
    SELECT 
        id,
        recipient_phone,
        status,
        created_at,
        sent_at,
        updated_at,
        error_message
    FROM broadcast_messages
    WHERE campaign_id = 70
    ORDER BY created_at DESC
""")

messages = cursor.fetchall()
print(f"Total broadcast messages for Campaign 70: {len(messages)}")

if messages:
    for msg in messages:
        print(f"\nMessage ID: {msg['id']}")
        print(f"  Phone: {msg['recipient_phone']}")
        print(f"  Status: {msg['status']}")
        print(f"  Created: {msg['created_at']}")
        print(f"  Updated: {msg['updated_at']}")
        print(f"  Sent: {msg['sent_at']}")
        if msg['error_message']:
            print(f"  Error: {msg['error_message']}")
else:
    print("\nNO BROADCAST MESSAGES WERE EVER CREATED FOR THIS CAMPAIGN!")
    print("This explains why it shows '1 remaining' - the campaign was marked 'finished'")
    print("but never actually created any broadcast_messages records.")

# Let's also check the campaign history
print("\n\n=== CAMPAIGN 70 HISTORY ===")
cursor.execute("""
    SELECT 
        id,
        title,
        status,
        created_at,
        updated_at,
        campaign_date,
        time_schedule
    FROM campaigns
    WHERE id = 70
""")

campaign = cursor.fetchone()
print(f"Campaign: {campaign['title']}")
print(f"Current Status: {campaign['status']}")
print(f"Created: {campaign['created_at']}")
print(f"Last Updated: {campaign['updated_at']}")
print(f"Scheduled: {campaign['campaign_date']} {campaign['time_schedule']}")

# Check if it's been processed again after reset
time_5_min_ago = campaign['updated_at'] - timedelta(minutes=5)
cursor.execute("""
    SELECT COUNT(*) as new_messages
    FROM broadcast_messages
    WHERE campaign_id = 70
    AND created_at > %s
""", (time_5_min_ago,))

result = cursor.fetchone()
if result['new_messages'] > 0:
    print(f"\n✅ {result['new_messages']} NEW MESSAGES CREATED AFTER RESET!")
else:
    print(f"\n❌ Still no messages created after reset")

conn.close()
