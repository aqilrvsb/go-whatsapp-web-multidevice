import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== DEBUGGING LEAD SEARCH ISSUE ===\n")

# 1. The query that SHOULD work (from direct_broadcast_processor.go)
print("1. Query used in ProcessCampaigns (simplified):")
cursor.execute("""
    SELECT COUNT(*) as count
    FROM leads l
    INNER JOIN user_devices ud ON l.device_id = ud.id
    WHERE ud.user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
    AND (ud.status = 'connected' OR ud.status = 'online' OR ud.platform IS NOT NULL)
    AND l.niche LIKE CONCAT('%', 'kiki', '%')
    AND ('prospect' = 'all' OR l.target_status = 'prospect')
""")
result = cursor.fetchone()
print(f"   Result: {result['count']} leads found\n")

# 2. Let's check each condition separately
print("2. Breaking down the conditions:")

# Check leads with this niche
cursor.execute("""
    SELECT COUNT(*) as count
    FROM leads l
    WHERE l.niche LIKE '%kiki%'
""")
result = cursor.fetchone()
print(f"   - Leads with niche 'kiki': {result['count']}")

# Check leads with this niche and target status
cursor.execute("""
    SELECT COUNT(*) as count
    FROM leads l
    WHERE l.niche LIKE '%kiki%'
    AND l.target_status = 'prospect'
""")
result = cursor.fetchone()
print(f"   - Leads with niche 'kiki' AND status 'prospect': {result['count']}")

# Check which devices these leads belong to
cursor.execute("""
    SELECT 
        l.id,
        l.phone,
        l.niche,
        l.target_status,
        l.device_id,
        ud.device_name,
        ud.user_id,
        ud.status as device_status,
        ud.platform
    FROM leads l
    LEFT JOIN user_devices ud ON l.device_id = ud.id
    WHERE l.niche LIKE '%kiki%'
    AND l.target_status = 'prospect'
""")
leads = cursor.fetchall()

print(f"\n3. Lead details:")
for lead in leads:
    print(f"   Lead ID: {lead['id']}")
    print(f"   Phone: {lead['phone']}")
    print(f"   Device ID: {lead['device_id']}")
    print(f"   Device Name: {lead['device_name']}")
    print(f"   Device User ID: {lead['user_id']}")
    print(f"   Device Status: {lead['device_status']}")
    print(f"   Platform: {lead['platform']}")
    
    # The issue might be here!
    if lead['user_id'] != 'de078f16-3266-4ab3-8153-a248b015228f':
        print(f"   >>> PROBLEM: This lead belongs to user {lead['user_id']}")
        print(f"   >>> But campaign is for user de078f16-3266-4ab3-8153-a248b015228f")

# 4. Let's see what the old campaign processor was doing
print("\n4. What the OLD campaign processor would find:")
cursor.execute("""
    SELECT COUNT(*) as count
    FROM leads l
    WHERE l.device_id IN (
        SELECT id FROM user_devices 
        WHERE user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
    )
    AND l.niche LIKE '%kiki%'
    AND l.target_status = 'prospect'
""")
result = cursor.fetchone()
print(f"   Leads for this user's devices: {result['count']}")

conn.close()
