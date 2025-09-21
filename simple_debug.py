import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

print("=== WHY CAMPAIGN ISN'T PROCESSING ===\n")

# Simple check
cursor.execute("SELECT id, title, status, campaign_date, time_schedule FROM campaigns WHERE id = 70")
result = cursor.fetchone()
print(f"Campaign: {result[1]}")
print(f"Status: {result[2]}")
print(f"Date: {result[3]}")
print(f"Time: {result[4]}")

cursor.execute("SELECT NOW()")
now = cursor.fetchone()[0]
print(f"\nCurrent DB Time: {now}")

print("\n=== THE PROBLEM ===")
print("Your logs show 'GetCampaignSummary' but NO campaign processing logs!")
print("This means:")
print("\n1. The unified processor is NOT running")
print("2. You need to deploy and restart the application")
print("\n=== WHAT YOU SHOULD SEE ===")
print("On startup:")
print("  'Starting Direct Broadcast Processor (Sequences + Campaigns)...'")
print("\nEvery 5 minutes:")
print("  'Processing campaign: kiki (Copy) (ID: 70)'")
print("  '✅ Campaigns: Processed 1 campaigns'")
print("\nYou're NOT seeing these, which means the processor isn't running!")

conn.close()
