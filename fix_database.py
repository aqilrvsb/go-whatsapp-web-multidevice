import pymysql

# Fix the database issues
conn = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    charset='utf8mb4'
)
cursor = conn.cursor()

# 1. Fix the lead trigger
print("Fixing lead trigger...")
cursor.execute("UPDATE leads SET `trigger` = 'meow' WHERE phone = '60108924904'")
print(f"Updated {cursor.rowcount} lead(s)")

# 2. Fix the sequence
print("\nFixing sequence...")
cursor.execute("""
UPDATE sequences 
SET `trigger` = 'meow',
    device_id = '315e4f8e-6868-4808-a3df-f75e9fce331f',
    min_delay_seconds = 10,
    max_delay_seconds = 30
WHERE name = 'meow'
""")
print(f"Updated {cursor.rowcount} sequence(s)")

# Commit changes
conn.commit()
print("\nDatabase fixes committed successfully!")

# Verify the fixes
cursor.execute("SELECT `trigger` FROM leads WHERE phone = '60108924904'")
lead_trigger = cursor.fetchone()[0]
print(f"\nLead trigger is now: '{lead_trigger}'")

cursor.execute("SELECT `trigger`, device_id, min_delay_seconds FROM sequences WHERE name = 'meow'")
seq_data = cursor.fetchone()
print(f"Sequence trigger is now: '{seq_data[0]}'")
print(f"Sequence device_id: {seq_data[1]}")
print(f"Sequence min_delay: {seq_data[2]}")

conn.close()
print("\nDatabase fixes completed!")
