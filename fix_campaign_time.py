import mysql.connector

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("Let me update your campaign to trigger now...")

# Update campaign to 1 minute ago
cursor.execute("""
    UPDATE campaigns 
    SET time_schedule = TIME(DATE_SUB(NOW(), INTERVAL 1 MINUTE))
    WHERE id = 70 AND status = 'pending'
""")
conn.commit()

print(f"Updated {cursor.rowcount} campaign(s)")

# Check the result
cursor.execute("""
    SELECT 
        id, 
        title, 
        campaign_date, 
        time_schedule, 
        status
    FROM campaigns
    WHERE id = 70
""")

result = cursor.fetchone()
if result:
    print(f"\nCampaign Updated:")
    print(f"  Title: {result[1]}")
    print(f"  Date: {result[2]}")
    print(f"  Time: {result[3]} (should be 1 minute ago)")
    print(f"  Status: {result[4]}")
    
    print("\n>>> Campaign should now trigger within 5 minutes!")
    print(">>> Check your application logs for:")
    print("    - 'Processing campaign: kiki (Copy)'")
    print("    - 'Campaigns: Processed 1 campaigns'")

conn.close()
