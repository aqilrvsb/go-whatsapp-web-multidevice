import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

# Quick status check
cursor.execute("SELECT id, title, status, time_schedule FROM campaigns WHERE id = 70")
result = cursor.fetchone()

print(f"Campaign 70: {result[1]}")
print(f"Status: {result[2]}")
print(f"Time: {result[3]}")

if result[2] == 'pending':
    print("\nCampaign is still PENDING - waiting to be processed")
    print("The processor runs every 5 minutes")
elif result[2] == 'finished':
    print("\nCampaign is FINISHED again - but let's check for messages")
    
    cursor.execute("SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = 70")
    count = cursor.fetchone()[0]
    print(f"Broadcast messages created: {count}")
    
    if count == 0:
        print("\n⚠️ PROBLEM: Campaign marked finished but NO messages created!")
        print("This indicates an issue in the campaign processor")

conn.close()
