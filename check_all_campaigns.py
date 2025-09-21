import mysql.connector
from datetime import datetime

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

# Check all campaigns
cursor.execute("""
    SELECT 
        id, 
        title, 
        campaign_date, 
        time_schedule, 
        status,
        niche,
        target_status
    FROM campaigns
    ORDER BY id DESC
    LIMIT 10
""")

campaigns = cursor.fetchall()
print("Recent campaigns:")
for camp in campaigns:
    print(f"\nID: {camp['id']}")
    print(f"  Title: {camp['title']}")
    print(f"  Date: {camp['campaign_date']}")
    print(f"  Time: {camp['time_schedule']}")
    print(f"  Status: {camp['status']}")
    print(f"  Niche: {camp['niche']}")
    print(f"  Target: {camp['target_status']}")

# Check if campaign 70 created any messages
cursor.execute("""
    SELECT COUNT(*) as msg_count, MIN(created_at) as first_msg, MAX(created_at) as last_msg
    FROM broadcast_messages
    WHERE campaign_id = 70
""")
result = cursor.fetchone()
print(f"\n\nCampaign 70 messages:")
print(f"  Total messages: {result['msg_count']}")
print(f"  First message: {result['first_msg']}")
print(f"  Last message: {result['last_msg']}")

# Check for any pending campaigns
cursor.execute("""
    SELECT COUNT(*) as pending_count
    FROM campaigns
    WHERE status = 'pending'
""")
result = cursor.fetchone()
print(f"\n\nTotal pending campaigns: {result['pending_count']}")

if result['pending_count'] == 0:
    print("\nNo pending campaigns found!")
    print("All campaigns have been processed.")
    print("\nTo test the system:")
    print("1. Create a new campaign in your UI")
    print("2. Set the date to today")
    print("3. Set the time to current time or a few minutes ago")
    print("4. Make sure the niche matches some of your leads")
    print("5. Wait up to 5 minutes for it to process")

conn.close()
