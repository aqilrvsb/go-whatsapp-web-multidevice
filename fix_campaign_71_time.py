import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

# Get current time
cursor.execute("SELECT NOW() as now_time")
result = cursor.fetchone()
print(f"Current Database Time: {result[0]}")

# Update campaign 71 to trigger now
cursor.execute("""
    UPDATE campaigns 
    SET time_schedule = TIME(DATE_SUB(NOW(), INTERVAL 5 MINUTE))
    WHERE id = 71 AND status = 'pending'
""")
conn.commit()

print("\nCampaign 71 updated to 5 minutes ago")
print("It should now process within 5 minutes")
print("\nWatch for these logs:")
print("- 'Processing campaign: kiki'")
print("- 'Campaign kiki triggered: X messages queued'")

conn.close()
