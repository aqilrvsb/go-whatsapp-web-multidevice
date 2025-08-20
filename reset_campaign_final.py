import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("Resetting campaign 70 to test the fix...")

# Reset campaign to pending
cursor.execute("""
    UPDATE campaigns 
    SET status = 'pending',
        time_schedule = TIME(DATE_SUB(NOW(), INTERVAL 2 MINUTE))
    WHERE id = 70
""")
conn.commit()

print(f"Campaign reset to 'pending' with time 2 minutes ago")
print("\nDeploy the new build and the campaign should now:")
print("1. Find the lead even though device is offline")
print("2. Create broadcast message record")
print("3. Show proper counts in UI")

conn.close()
