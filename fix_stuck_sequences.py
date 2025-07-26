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
    
    print("=== FIXING STUCK SEQUENCE PROCESSING ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Clear stuck processing
    print("1. CLEARING STUCK PROCESSING FLAGS:")
    cur.execute("""
        UPDATE sequence_contacts
        SET 
            processing_device_id = NULL,
            processing_started_at = NULL
        WHERE processing_device_id IS NOT NULL
        AND status = 'active'
    """)
    
    cleared = cur.rowcount
    print(f"Cleared {cleared} stuck processing flags")
    
    # 2. Reset trigger times to NOW for immediate processing
    print("\n2. RESETTING TRIGGER TIMES:")
    cur.execute("""
        UPDATE sequence_contacts
        SET next_trigger_time = NOW()
        WHERE status = 'active'
        AND next_trigger_time > NOW()
    """)
    
    reset = cur.rowcount
    print(f"Reset {reset} trigger times to NOW")
    
    # 3. Check if sequence processor is needed
    print("\n3. CHECKING SEQUENCE STATUS:")
    cur.execute("""
        SELECT 
            s.name,
            COUNT(sc.id) as active_contacts,
            MIN(sc.next_trigger_time) as earliest_trigger
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        WHERE sc.status = 'active'
        GROUP BY s.name
    """)
    
    sequences = cur.fetchall()
    for seq in sequences:
        print(f"  {seq[0]}: {seq[1]} active contacts, next trigger: {seq[2]}")
    
    # 4. Create test broadcast messages manually
    print("\n4. CREATING TEST BROADCAST MESSAGES:")
    cur.execute("""
        INSERT INTO broadcast_messages (
            user_id, device_id, sequence_id, sequence_stepid,
            recipient_phone, recipient_name, message_type, content,
            status, scheduled_at, created_at
        )
        SELECT 
            sc.user_id,
            sc.assigned_device_id,
            sc.sequence_id,
            sc.sequence_stepid,
            sc.contact_phone,
            sc.contact_name,
            'text',
            ss.content,
            'pending',
            NOW(),
            NOW()
        FROM sequence_contacts sc
        JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
        WHERE sc.status = 'active'
        AND sc.next_trigger_time <= NOW()
        AND NOT EXISTS (
            SELECT 1 FROM broadcast_messages bm
            WHERE bm.recipient_phone = sc.contact_phone
            AND bm.sequence_id = sc.sequence_id
            AND bm.created_at > NOW() - INTERVAL '1 hour'
        )
        LIMIT 5
    """)
    
    created = cur.rowcount
    print(f"Created {created} test broadcast messages")
    
    # Commit changes
    conn.commit()
    print("\n✅ All changes committed!")
    
    # 5. Show current state
    print("\n5. CURRENT STATE:")
    cur.execute("""
        SELECT 
            'Active sequence contacts' as metric,
            COUNT(*) as count
        FROM sequence_contacts
        WHERE status = 'active'
        UNION ALL
        SELECT 
            'Pending broadcast messages' as metric,
            COUNT(*) as count
        FROM broadcast_messages
        WHERE status = 'pending'
    """)
    
    for row in cur.fetchall():
        print(f"  {row[0]}: {row[1]}")
    
    cur.close()
    conn.close()
    
    print("\n💡 NEXT STEPS:")
    print("1. Check if broadcast processor is running")
    print("2. Monitor if messages start sending")
    print("3. Verify sequence trigger processor is active")
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
