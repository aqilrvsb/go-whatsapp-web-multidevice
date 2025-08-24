import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== FIXING STUCK CAMPAIGN MESSAGE ===\n")

# Update the message to send now (subtract 8 hours)
cursor.execute("""
    UPDATE broadcast_messages 
    SET scheduled_at = DATE_SUB(scheduled_at, INTERVAL 8 HOUR)
    WHERE id = '61774152-90da-4e9d-a2bc-e71a3ef47c20'
    AND status = 'pending'
""")
conn.commit()

if cursor.rowcount > 0:
    print("✅ Fixed! Message scheduled time adjusted for timezone")
    print("The message should now be processed within the next minute")
    
    # Verify the update
    cursor.execute("""
        SELECT 
            scheduled_at,
            NOW() as server_now,
            CASE 
                WHEN scheduled_at <= NOW() THEN 'READY TO SEND'
                ELSE 'WAITING'
            END as status
        FROM broadcast_messages
        WHERE id = '61774152-90da-4e9d-a2bc-e71a3ef47c20'
    """)
    
    result = cursor.fetchone()
    print(f"\nNew scheduled time: {result[0]}")
    print(f"Server time: {result[1]}")
    print(f"Status: {result[2]}")
else:
    print("Message not updated - may already be processed")

conn.close()
