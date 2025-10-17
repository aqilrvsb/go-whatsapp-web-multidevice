import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== CHECKING ALL CAMPAIGNS ===\n")

cursor.execute("""
    SELECT id, title, status, campaign_date, time_schedule 
    FROM campaigns 
    WHERE campaign_date >= CURDATE()
    ORDER BY id DESC
""")

campaigns = cursor.fetchall()
print(f"Found {len(campaigns)} campaigns for today or later:\n")

for camp in campaigns:
    print(f"ID: {camp[0]}")
    print(f"  Title: {camp[1]}")
    print(f"  Status: {camp[2]}")
    print(f"  Date: {camp[3]}")
    print(f"  Time: {camp[4]}")
    print()

if len(campaigns) == 0:
    print("NO CAMPAIGNS FOUND!")
    print("Campaign 70 might have been deleted.")
    print("\nCreate a new campaign to test:")
    print("1. Create campaign in UI")
    print("2. Set date to today")
    print("3. Set time to current time or past")
    print("4. Match niche with your leads")

conn.close()
