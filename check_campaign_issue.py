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

print("=== CAMPAIGN 70 ISSUE ===\n")

# Simple check
cursor.execute("""
    SELECT 
        (SELECT COUNT(*) FROM leads l 
         INNER JOIN user_devices ud ON l.device_id = ud.id
         WHERE ud.user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
         AND l.niche LIKE '%kiki%'
         AND l.target_status = 'prospect') as matching_leads,
        (SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = 70) as messages_created,
        (SELECT COUNT(*) FROM user_devices 
         WHERE user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
         AND (status = 'connected' OR status = 'online' OR platform IS NOT NULL)) as connected_devices
""")

result = cursor.fetchone()
print(f"Matching Leads: {result[0]}")
print(f"Messages Created: {result[1]}")
print(f"Connected Devices: {result[2]}")

print("\n=== DIAGNOSIS ===")
if result[1] == 0:
    print("NO MESSAGES WERE CREATED!")
    
    if result[0] > 0 and result[2] == 0:
        print("Reason: No connected devices when campaign processed")
    elif result[0] == 0:
        print("Reason: No leads match the criteria")
    else:
        print("Reason: Campaign was marked 'finished' without processing")
        print("This happens when:")
        print("1. Campaign runs but finds no valid leads/devices")
        print("2. All messages were blocked by duplicate prevention")
        
# Check when campaign was updated
cursor.execute("SELECT updated_at FROM campaigns WHERE id = 70")
updated = cursor.fetchone()
print(f"\nCampaign last updated: {updated[0]}")

# Reset campaign to test again?
print("\n=== SOLUTION ===")
print("To reprocess this campaign:")
print("1. Create a new campaign (recommended)")
print("2. OR reset this campaign to 'pending' status")
print("\nWould you like me to reset it to 'pending'? (Not recommended)")

conn.close()
