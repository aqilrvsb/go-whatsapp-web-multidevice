import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== TESTING TIMEZONE FIX ===\n")

# Reset campaign 71 to current Malaysia time
cursor.execute("""
    UPDATE campaigns 
    SET status = 'pending',
        time_schedule = TIME(DATE_ADD(NOW(), INTERVAL 8 HOUR))
    WHERE id = 71
""")
conn.commit()

# Check the result
cursor.execute("""
    SELECT 
        id, title, time_schedule,
        NOW() as server_time,
        DATE_ADD(NOW(), INTERVAL 8 HOUR) as malaysia_time
    FROM campaigns 
    WHERE id = 71
""")
result = cursor.fetchone()

print(f"Campaign: {result[1]}")
print(f"Time Schedule: {result[2]} (Malaysia time)")
print(f"Server Time: {result[3]} (UTC)")
print(f"Malaysia Time: {result[4]}")

print("\n✅ Campaign set to current Malaysia time!")
print("With the timezone fix, it should trigger immediately")
print("because the query now checks: campaign_time <= NOW() + 8 hours")

conn.close()
