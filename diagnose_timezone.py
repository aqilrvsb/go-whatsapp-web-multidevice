import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== TIMEZONE ISSUE DIAGNOSIS ===\n")

# Check server timezone
cursor.execute("SELECT @@global.time_zone, @@session.time_zone")
tz = cursor.fetchone()
print(f"MySQL Global Timezone: {tz[0]}")
print(f"MySQL Session Timezone: {tz[1]}")

# Check current times
cursor.execute("SELECT NOW() as server_time, CONVERT_TZ(NOW(), @@session.time_zone, '+08:00') as malaysia_time")
times = cursor.fetchone()
print(f"\nServer Time (NOW()): {times[0]}")
print(f"Malaysia Time (+8): {times[1] if times[1] else 'CONVERT_TZ not available'}")

# Check campaign 71
cursor.execute("""
    SELECT 
        campaign_date,
        time_schedule,
        STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') as parsed_time,
        NOW() as server_now,
        CASE 
            WHEN STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') <= NOW() 
            THEN 'YES' ELSE 'NO' 
        END as should_trigger
    FROM campaigns 
    WHERE id = 71
""")
result = cursor.fetchone()

print(f"\n=== Campaign 71 ===")
print(f"Date: {result[0]}")
print(f"Time: {result[1]}")
print(f"Parsed Time: {result[2]}")
print(f"Server NOW(): {result[3]}")
print(f"Should Trigger? {result[4]}")

print("\n=== THE PROBLEM ===")
print("Your campaign times are in Malaysia time (+8)")
print("But the server compares them with NOW() which is UTC or server timezone")
print("So a campaign set for 14:21 Malaysia time")
print("Is actually 06:21 UTC - 8 hours earlier!")

print("\n=== SOLUTIONS ===")
print("1. Quick fix: Set campaign times 8 hours earlier")
print("2. Better fix: Modify the query to handle timezone")
print("3. Best fix: Store all times in UTC and convert in UI")

conn.close()
