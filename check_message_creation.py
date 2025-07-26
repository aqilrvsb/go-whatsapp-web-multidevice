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
    
    print("=== CHECKING BROADCAST MESSAGE CREATION ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Check recent broadcast messages for sequences
    print("1. RECENT SEQUENCE MESSAGES IN BROADCAST_MESSAGES:")
    cur.execute("""
        SELECT 
            bm.id,
            bm.recipient_phone,
            bm.status,
            bm.sequence_id,
            bm.sequence_stepid,
            ss.day_number as step_num,
            bm.created_at,
            bm.scheduled_at,
            bm.sent_at,
            LEFT(bm.content, 50) as message_preview
        FROM broadcast_messages bm
        LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
        WHERE bm.sequence_id IS NOT NULL
        ORDER BY bm.created_at DESC
        LIMIT 20
    """)
    
    messages = cur.fetchall()
    for msg in messages:
        print(f"\nPhone: {msg[1]}, Status: {msg[2]}, Step: {msg[5]}")
        print(f"  Created: {msg[6]}, Scheduled: {msg[7]}, Sent: {msg[8]}")
        print(f"  Message: {msg[9]}...")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Check if processContactWithNewLogic is creating messages
    print("2. MESSAGES CREATED IN LAST HOUR:")
    cur.execute("""
        SELECT 
            DATE_TRUNC('minute', created_at) as minute,
            COUNT(*) as messages_created,
            SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
            SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
        AND created_at > NOW() - INTERVAL '1 hour'
        GROUP BY DATE_TRUNC('minute', created_at)
        ORDER BY minute DESC
        LIMIT 10
    """)
    
    creation_stats = cur.fetchall()
    if creation_stats:
        print("Minute | Created | Sent | Pending | Failed")
        print("-" * 50)
        for stat in creation_stats:
            print(f"{stat[0].strftime('%H:%M')} | {stat[1]:7} | {stat[2]:4} | {stat[3]:7} | {stat[4]:6}")
    else:
        print("No messages created in the last hour")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Check sequence_contacts that should be processed
    print("3. SEQUENCE CONTACTS READY FOR PROCESSING:")
    cur.execute("""
        SELECT 
            sc.contact_phone,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.next_trigger_time,
            CASE 
                WHEN sc.next_trigger_time <= NOW() THEN 'READY'
                ELSE 'WAITING'
            END as ready_status,
            sc.processing_device_id
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        WHERE sc.status IN ('active', 'pending')
        AND s.is_active = true
        ORDER BY sc.next_trigger_time
        LIMIT 10
    """)
    
    ready_contacts = cur.fetchall()
    for contact in ready_contacts:
        print(f"\nPhone: {contact[0]} - {contact[1]}")
        print(f"  Step: {contact[2]}, Status: {contact[3]}, Ready: {contact[5]}")
        print(f"  Next trigger: {contact[4]}, Processing device: {contact[6]}")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Check if there's a mismatch
    print("4. CHECKING FOR PROCESSING ISSUES:")
    
    # Contacts ready but no messages
    cur.execute("""
        SELECT COUNT(*)
        FROM sequence_contacts sc
        WHERE sc.status = 'active'
        AND sc.next_trigger_time <= NOW()
        AND NOT EXISTS (
            SELECT 1 FROM broadcast_messages bm
            WHERE bm.recipient_phone = sc.contact_phone
            AND bm.sequence_id = sc.sequence_id
            AND bm.created_at > NOW() - INTERVAL '5 minutes'
        )
    """)
    
    no_messages = cur.fetchone()[0]
    print(f"Contacts ready but no recent messages created: {no_messages}")
    
    # Check if processor is running
    cur.execute("""
        SELECT 
            MAX(created_at) as last_message_created,
            NOW() - MAX(created_at) as time_since_last
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
    """)
    
    last_created = cur.fetchone()
    if last_created[0]:
        print(f"Last sequence message created: {last_created[0]} ({last_created[1]} ago)")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
