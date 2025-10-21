import mysql.connector
from datetime import datetime, timedelta

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()
print("Connected to MySQL successfully!")

# Get current time
cursor.execute("SELECT NOW() as db_time")
result = cursor.fetchone()
current_time = result[0]
print(f"\nCurrent Database Time: {current_time}")

# Update the campaign to 5 minutes ago so it triggers
five_minutes_ago = current_time - timedelta(minutes=5)
new_time = five_minutes_ago.strftime('%H:%M')

print(f"\nUpdating campaign 70 to trigger at {new_time} (5 minutes ago)")

# Update the campaign
cursor.execute("""
    UPDATE campaigns 
    SET time_schedule = %s
    WHERE id = 70 AND status = 'pending'
""", (new_time,))

conn.commit()
print(f"✅ Campaign updated! Rows affected: {cursor.rowcount}")

# Verify the update
cursor.execute("""
    SELECT id, title, campaign_date, time_schedule, status
    FROM campaigns
    WHERE id = 70
""")
result = cursor.fetchone()
if result:
    print(f"\nUpdated campaign:")
    print(f"  ID: {result[0]}")
    print(f"  Title: {result[1]}")
    print(f"  Date: {result[2]}")
    print(f"  Time: {result[3]}")
    print(f"  Status: {result[4]}")
    
print("\n🎯 Campaign should now trigger on the next processor run (within 5 minutes)")
print("Check your application logs for 'Processing campaign: kiki (Copy)'")

conn.close()
