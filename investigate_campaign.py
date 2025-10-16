import mysql.connector

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== INVESTIGATING CAMPAIGN 70 (kiki Copy) ===\n")

# 1. Check campaign details
cursor.execute("""
    SELECT * FROM campaigns WHERE id = 70
""")
campaign = cursor.fetchone()
print(f"Campaign Status: {campaign['status']}")
print(f"Target Status: {campaign['target_status']}")
print(f"Niche: {campaign['niche']}")
print(f"User ID: {campaign['user_id']}")

# 2. Check broadcast messages for this campaign
print("\n=== BROADCAST MESSAGES ===")
cursor.execute("""
    SELECT 
        id,
        recipient_phone,
        status,
        created_at,
        sent_at,
        error_message
    FROM broadcast_messages
    WHERE campaign_id = 70
    ORDER BY created_at DESC
""")
messages = cursor.fetchall()
print(f"Total messages created: {len(messages)}")

if messages:
    for msg in messages:
        print(f"\nMessage ID: {msg['id']}")
        print(f"  Phone: {msg['recipient_phone']}")
        print(f"  Status: {msg['status']}")
        print(f"  Created: {msg['created_at']}")
        print(f"  Sent: {msg['sent_at']}")
        if msg['error_message']:
            print(f"  Error: {msg['error_message']}")

# 3. Check leads that match the campaign criteria
print("\n=== MATCHING LEADS ===")
cursor.execute("""
    SELECT 
        l.id,
        l.phone,
        l.name,
        l.niche,
        l.target_status,
        ud.device_name,
        ud.status as device_status
    FROM leads l
    INNER JOIN user_devices ud ON l.device_id = ud.id
    WHERE ud.user_id = %s
    AND l.niche LIKE CONCAT('%%', %s, '%%')
    AND (%s = 'all' OR l.target_status = %s)
    LIMIT 10
""", (campaign['user_id'], campaign['niche'], campaign['target_status'], campaign['target_status']))

leads = cursor.fetchall()
print(f"Leads matching criteria: {len(leads)}")

for lead in leads:
    print(f"\nLead: {lead['name']} ({lead['phone']})")
    print(f"  Niche: {lead['niche']}")
    print(f"  Target Status: {lead['target_status']}")
    print(f"  Device: {lead['device_name']} (Status: {lead['device_status']})")
    
    # Check if this lead already has a message
    cursor.execute("""
        SELECT id, status FROM broadcast_messages 
        WHERE campaign_id = 70 AND recipient_phone = %s
    """, (lead['phone'],))
    msg = cursor.fetchone()
    if msg:
        print(f"  Has Message: YES (Status: {msg['status']})")
    else:
        print(f"  Has Message: NO")

# 4. Check the campaign summary calculation
print("\n=== CAMPAIGN SUMMARY CALCULATION ===")

# This is likely how your UI calculates the numbers
cursor.execute("""
    SELECT 
        COUNT(DISTINCT l.phone) as should_send
    FROM leads l
    INNER JOIN user_devices ud ON l.device_id = ud.id
    WHERE ud.user_id = %s
    AND l.niche LIKE CONCAT('%%', %s, '%%')
    AND (%s = 'all' OR l.target_status = %s)
""", (campaign['user_id'], campaign['niche'], campaign['target_status'], campaign['target_status']))
result = cursor.fetchone()
print(f"Contacts Should Send: {result['should_send']}")

# Messages sent
cursor.execute("""
    SELECT 
        COUNT(*) as total,
        SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
        SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
        SUM(CASE WHEN status IN ('pending', 'processing', 'queued') THEN 1 ELSE 0 END) as in_progress
    FROM broadcast_messages
    WHERE campaign_id = 70
""")
result = cursor.fetchone()
print(f"Messages Created: {result['total']}")
print(f"Messages Sent: {result['sent']}")
print(f"Messages Failed: {result['failed']}")
print(f"Messages In Progress: {result['in_progress']}")

print("\n=== CONCLUSION ===")
if result['total'] == 0:
    print("NO MESSAGES WERE CREATED for this campaign!")
    print("Possible reasons:")
    print("1. No connected devices when campaign ran")
    print("2. Duplicate prevention blocked all messages")
    print("3. Campaign processor didn't pick it up properly")
else:
    print(f"Campaign created {result['total']} messages")

conn.close()
