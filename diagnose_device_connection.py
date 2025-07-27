import psycopg2
import sys

sys.stdout.reconfigure(encoding='utf-8')

conn = psycopg2.connect(
    "postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway"
)
cursor = conn.cursor()

print("=== DEVICE CONNECTION DIAGNOSTICS ===\n")

device_id = "d409cadc-75e2-4004-a789-c2bad0b31393"

# 1. Check device info
cursor.execute("""
    SELECT device_name, status, platform, phone, jid, last_seen, created_at
    FROM user_devices
    WHERE id = %s
""", (device_id,))

device = cursor.fetchone()
if device:
    print(f"Device: {device[0]}")
    print(f"  Status: {device[1]}")
    print(f"  Platform: {device[2] or 'WhatsApp Web'}")
    print(f"  Phone: {device[3]}")
    print(f"  JID: {device[4]}")
    print(f"  Last seen: {device[5]}")
    print(f"  Created: {device[6]}")

# 2. Check for duplicate devices with same phone
if device[3]:
    cursor.execute("""
        SELECT id, device_name, status, last_seen
        FROM user_devices
        WHERE phone = %s
        ORDER BY last_seen DESC
    """, (device[3],))
    
    duplicates = cursor.fetchall()
    if len(duplicates) > 1:
        print(f"\n⚠️ WARNING: Found {len(duplicates)} devices with same phone number!")
        for dup in duplicates:
            print(f"  - {dup[0]}: {dup[1]} (Status: {dup[2]}, Last seen: {dup[3]})")

# 3. Check whatsmeow store
cursor.execute("""
    SELECT COUNT(*) 
    FROM whatsmeow_device
    WHERE jid LIKE %s
""", (f"%{device_id}%",))
count = cursor.fetchone()[0]
print(f"\n📱 WhatsApp Store: {count} records found")

# 4. Check recent broadcast attempts
print("\n=== RECENT BROADCAST ATTEMPTS ===")
cursor.execute("""
    SELECT 
        bm.created_at,
        bm.status,
        bm.error_message,
        c.title as campaign
    FROM broadcast_messages bm
    LEFT JOIN campaigns c ON c.id = bm.campaign_id
    WHERE bm.device_id = %s
    ORDER BY bm.created_at DESC
    LIMIT 10
""", (device_id,))

for msg in cursor.fetchall():
    print(f"\n{msg[0]}")
    print(f"  Campaign: {msg[3]}")
    print(f"  Status: {msg[1]}")
    if msg[2]:
        print(f"  Error: {msg[2]}")

conn.close()

print("\n=== RECOMMENDATIONS ===")
print("1. If duplicate devices exist, remove old ones")
print("2. Try disconnecting and reconnecting the device")
print("3. Check if device shows in dashboard as 'online'")
print("4. Consider using a platform device (Wablas/Whacenter) for stability")
