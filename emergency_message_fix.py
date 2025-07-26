import psycopg2
import sys
from datetime import datetime

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== EMERGENCY: FORCE PROCESS STUCK MESSAGES ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Create missing broadcast messages for sequences
    print("1. CREATING MISSING SEQUENCE MESSAGES:")
    cur.execute("""
        INSERT INTO broadcast_messages (
            user_id, device_id, sequence_id, sequence_stepid,
            recipient_phone, recipient_name, message_type, content,
            status, scheduled_at, created_at
        )
        SELECT 
            sc.user_id,
            COALESCE(sc.assigned_device_id, (
                SELECT id FROM user_devices 
                WHERE user_id = sc.user_id 
                AND status IN ('online', 'connected')
                LIMIT 1
            )),
            sc.sequence_id,
            sc.sequence_stepid,
            sc.contact_phone,
            sc.contact_name,
            COALESCE(ss.message_type, 'text'),
            ss.content,
            'pending',
            NOW(),
            NOW()
        FROM sequence_contacts sc
        JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
        WHERE sc.status = 'pending'
        AND sc.next_trigger_time <= NOW()
        AND NOT EXISTS (
            SELECT 1 FROM broadcast_messages bm
            WHERE bm.recipient_phone = sc.contact_phone
            AND bm.sequence_stepid = sc.sequence_stepid
        )
    """)
    
    created = cur.rowcount
    print(f"Created {created} missing broadcast messages")
    
    # 2. Force update stuck pending messages
    print("\n2. RESETTING STUCK MESSAGES:")
    cur.execute("""
        UPDATE broadcast_messages
        SET 
            scheduled_at = NOW(),
            status = 'pending'
        WHERE status = 'pending'
        AND scheduled_at < NOW() - INTERVAL '1 hour'
        AND sent_at IS NULL
    """)
    
    reset = cur.rowcount
    print(f"Reset {reset} stuck messages")
    
    # 3. Check device availability
    print("\n3. DEVICE STATUS:")
    cur.execute("""
        SELECT 
            id,
            phone,
            status,
            platform,
            updated_at
        FROM user_devices
        WHERE platform IS NULL
        ORDER BY status DESC
    """)
    
    devices = cur.fetchall()
    for dev in devices:
        print(f"  Device {dev[0][:8]}... ({dev[1]}): {dev[2]}, updated: {dev[4]}")
    
    # 4. Force mark one device as online for testing
    print("\n4. FORCING DEVICE ONLINE:")
    cur.execute("""
        UPDATE user_devices
        SET status = 'online'
        WHERE id = (
            SELECT id FROM user_devices 
            WHERE platform IS NULL 
            ORDER BY updated_at DESC 
            LIMIT 1
        )
        RETURNING id, phone
    """)
    
    updated_device = cur.fetchone()
    if updated_device:
        print(f"Marked device {updated_device[0]} as online")
    
    # 5. Assign all pending messages to online device
    print("\n5. ASSIGNING MESSAGES TO ONLINE DEVICE:")
    cur.execute("""
        UPDATE broadcast_messages
        SET device_id = (
            SELECT id FROM user_devices 
            WHERE status = 'online' 
            AND platform IS NULL
            LIMIT 1
        )
        WHERE status = 'pending'
        AND device_id IN (
            SELECT id FROM user_devices WHERE status = 'offline'
        )
    """)
    
    reassigned = cur.rowcount
    print(f"Reassigned {reassigned} messages to online device")
    
    # Commit all changes
    conn.commit()
    print("\n✅ Emergency fixes applied!")
    
    # 6. Show current state
    print("\n6. CURRENT STATE:")
    cur.execute("""
        SELECT 
            'Total pending messages' as metric,
            COUNT(*) as value
        FROM broadcast_messages
        WHERE status = 'pending'
        UNION ALL
        SELECT 
            'Messages ready to send' as metric,
            COUNT(*) as value
        FROM broadcast_messages bm
        JOIN user_devices ud ON ud.id = bm.device_id
        WHERE bm.status = 'pending'
        AND ud.status IN ('online', 'connected')
    """)
    
    for row in cur.fetchall():
        print(f"  {row[0]}: {row[1]}")
    
    print("\n💡 NEXT STEPS:")
    print("1. Ensure UltraOptimizedBroadcastProcessor is running")
    print("2. Check Redis connection in the app")
    print("3. Monitor if messages start sending")
    print("4. Connect more WhatsApp devices for better throughput")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
