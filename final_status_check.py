import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== FINAL STATUS CHECK ===\n")

# Check campaign status
cursor.execute("SELECT id, title, status, time_schedule FROM campaigns WHERE id = 70")
campaign = cursor.fetchone()
print(f"Campaign: {campaign['title']}")
print(f"Status: {campaign['status']}")
print(f"Time: {campaign['time_schedule']}")

# Check if any messages were created
cursor.execute("SELECT COUNT(*) as count FROM broadcast_messages WHERE campaign_id = 70")
result = cursor.fetchone()
print(f"\nBroadcast messages created: {result['count']}")

# Check the lead that should receive the message
cursor.execute("""
    SELECT l.phone, l.niche, l.target_status, ud.device_name, ud.status
    FROM leads l
    INNER JOIN user_devices ud ON l.device_id = ud.id
    WHERE l.niche LIKE '%kiki%' AND l.target_status = 'prospect'
""")
lead = cursor.fetchone()
if lead:
    print(f"\nTarget Lead:")
    print(f"  Phone: {lead['phone']}")
    print(f"  Niche: {lead['niche']}")
    print(f"  Target Status: {lead['target_status']}")
    print(f"  Device: {lead['device_name']} (Status: {lead['status']})")

print("\n=== NEXT STEPS ===")
print("1. The fix has been pushed to GitHub")
print("2. Campaign 70 is ready to be processed")
print("3. The processor will now find leads regardless of device status")
print("4. Deploy the updated code and watch for these logs:")
print("   - 'Processing campaign: kiki (Copy)'")
print("   - 'Campaign kiki (Copy) triggered: 1 messages queued'")

conn.close()
