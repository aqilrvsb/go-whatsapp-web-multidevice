import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== CAMPAIGNS NOW WORK EXACTLY LIKE SEQUENCES ===\n")

print("WHAT CHANGED:")
print("1. ScheduledAt: time.Now().Add(5 * time.Minute)")
print("   - Messages scheduled 5 minutes in future (like sequences)")
print("2. Status: 'pending' explicitly set")
print("3. Added Message field (sequences have both Message and Content)")
print("4. Added MinDelay and MaxDelay from campaign settings")

print("\nHOW IT WORKS NOW:")
print("1. Campaign triggers when scheduled time is reached")
print("2. Creates broadcast_messages with 5 minute delay")
print("3. Broadcast processor picks up messages after 5 minutes")
print("4. Messages are sent with random delay between min/max")

print("\nEXACTLY LIKE SEQUENCES!")

# Update any existing pending campaign messages to have proper scheduled time
cursor.execute("""
    UPDATE broadcast_messages 
    SET scheduled_at = DATE_ADD(NOW(), INTERVAL 5 MINUTE)
    WHERE campaign_id IS NOT NULL 
    AND status = 'pending'
    AND scheduled_at <= NOW()
""")
affected = cursor.rowcount
conn.commit()

if affected > 0:
    print(f"\nUpdated {affected} existing campaign messages to schedule 5 minutes from now")

conn.close()
