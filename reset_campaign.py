import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

# Get the lead that matches
cursor.execute("""
    SELECT l.phone, l.name, l.niche, l.target_status
    FROM leads l
    INNER JOIN user_devices ud ON l.device_id = ud.id
    WHERE ud.user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
    AND l.niche LIKE '%kiki%'
    AND l.target_status = 'prospect'
    LIMIT 1
""")
lead = cursor.fetchone()

if lead:
    print(f"Matching Lead: {lead['phone']}")
    print(f"Niche: {lead['niche']}")
    
    # Check all messages for this phone number
    cursor.execute("""
        SELECT 
            bm.id,
            bm.campaign_id,
            bm.status,
            bm.created_at,
            c.title as campaign_title
        FROM broadcast_messages bm
        LEFT JOIN campaigns c ON bm.campaign_id = c.id
        WHERE bm.recipient_phone = %s
        AND bm.campaign_id IS NOT NULL
        ORDER BY bm.created_at DESC
    """, (lead['phone'],))
    
    messages = cursor.fetchall()
    print(f"\nMessages for this phone: {len(messages)}")
    
    for msg in messages:
        print(f"\n- Campaign: {msg['campaign_title']} (ID: {msg['campaign_id']})")
        print(f"  Status: {msg['status']}")
        print(f"  Created: {msg['created_at']}")

# Let me reset the campaign to pending so it can be reprocessed
print("\n\nResetting campaign 70 to 'pending'...")
cursor.execute("""
    UPDATE campaigns 
    SET status = 'pending',
        time_schedule = TIME(DATE_SUB(NOW(), INTERVAL 5 MINUTE))
    WHERE id = 70
""")
conn.commit()

print("Campaign reset to 'pending' with time 5 minutes ago")
print("\nThe campaign should reprocess within 5 minutes!")
print("Watch for these logs:")
print("- Processing campaign: kiki (Copy)")
print("- Campaign kiki (Copy) enrolled X leads")

conn.close()
