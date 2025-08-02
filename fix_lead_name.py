import pymysql

# Connect to MySQL
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)
cursor = conn.cursor()

print("=== FIXING LEAD NAME ===\n")

# Update the lead name from phone number to actual name
print("Updating lead name from phone number to 'Aqil 1'...")
cursor.execute("""
    UPDATE leads 
    SET name = 'Aqil 1'
    WHERE phone = '60108924904'
""")
updated = cursor.rowcount
conn.commit()

print(f"Updated {updated} lead(s)")

# Set the trigger back to 'meow' for testing
print("\nSetting trigger back to 'meow'...")
cursor.execute("""
    UPDATE leads 
    SET `trigger` = 'meow'
    WHERE phone = '60108924904'
""")
conn.commit()

# Verify the changes
cursor.execute("""
    SELECT name, phone, `trigger`
    FROM leads 
    WHERE phone = '60108924904'
""")
result = cursor.fetchone()
print(f"\nLead updated:")
print(f"  Name: {result[0]}")
print(f"  Phone: {result[1]}")
print(f"  Trigger: {result[2]}")

print("\n=== READY FOR NEW TEST ===")
print("The lead name is now 'Aqil 1' instead of the phone number.")
print("The sequence processor will run again in a few minutes and should create")
print("a new message with the proper greeting: 'Hello Aqil 1,'")

conn.close()
