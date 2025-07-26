import psycopg2
import sys
from datetime import datetime, timedelta

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== FIX SEQUENCE LINKING AND MESSAGE SENDING ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Find sequences with linking issues
    print("1. CHECKING SEQUENCE LINKING:")
    cur.execute("""
        SELECT 
            ss1.sequence_id,
            s1.name as sequence_name,
            ss1.day_number,
            ss1.trigger,
            ss1.next_trigger,
            s2.id as linked_sequence_id,
            s2.name as linked_sequence_name
        FROM sequence_steps ss1
        LEFT JOIN sequences s1 ON s1.id = ss1.sequence_id
        LEFT JOIN sequence_steps ss2 ON ss2.trigger = ss1.next_trigger AND ss2.is_entry_point = true
        LEFT JOIN sequences s2 ON s2.id = ss2.sequence_id
        WHERE ss1.next_trigger IS NOT NULL
        AND ss1.next_trigger != ''
        AND NOT ss1.next_trigger LIKE '%_day%'
        ORDER BY s1.name, ss1.day_number
    """)
    
    links = cur.fetchall()
    print("Found sequence links:")
    for link in links:
        print(f"  {link[1]} Step {link[2]} -> {link[6] or 'NOT FOUND'} (trigger: {link[4]})")
    
    # 2. Fix stuck processing
    print("\n2. FIXING STUCK PROCESSING:")
    cur.execute("""
        UPDATE sequence_contacts
        SET processing_device_id = NULL,
            status = 'pending'
        WHERE processing_device_id IS NOT NULL
        AND (processing_started_at < NOW() - INTERVAL '10 minutes'
             OR processing_started_at IS NULL)
    """)
    fixed = cur.rowcount
    print(f"Fixed {fixed} stuck contacts")
    
    # 3. Create missing linked sequence enrollments
    print("\n3. CREATING MISSING LINKED ENROLLMENTS:")
    
    # Find contacts that should have linked sequences but don't
    cur.execute("""
        WITH linked_sequences AS (
            SELECT DISTINCT
                sc.contact_phone,
                sc.contact_name,
                sc.user_id,
                sc.assigned_device_id,
                ss1.next_trigger,
                s2.id as linked_sequence_id
            FROM sequence_contacts sc
            JOIN sequence_steps ss1 ON ss1.id = sc.sequence_stepid
            JOIN sequence_steps ss2 ON ss2.trigger = ss1.next_trigger AND ss2.is_entry_point = true
            JOIN sequences s2 ON s2.id = ss2.sequence_id
            WHERE ss1.next_trigger IS NOT NULL
            AND NOT ss1.next_trigger LIKE '%_day%'
            AND s2.is_active = true
            AND NOT EXISTS (
                SELECT 1 FROM sequence_contacts sc2
                WHERE sc2.contact_phone = sc.contact_phone
                AND sc2.sequence_id = s2.id
            )
        )
        INSERT INTO sequence_contacts (
            sequence_id, contact_phone, contact_name,
            current_step, status, current_trigger,
            next_trigger_time, sequence_stepid,
            assigned_device_id, user_id
        )
        SELECT 
            ls.linked_sequence_id,
            ls.contact_phone,
            ls.contact_name,
            1,
            'pending',
            ls.next_trigger,
            NOW() + INTERVAL '1 hour',
            (SELECT id FROM sequence_steps 
             WHERE sequence_id = ls.linked_sequence_id 
             AND day_number = 1 LIMIT 1),
            ls.assigned_device_id,
            ls.user_id
        FROM linked_sequences ls
        ON CONFLICT (sequence_id, contact_phone, sequence_stepid) DO NOTHING
    """)
    
    linked_created = cur.rowcount
    print(f"Created {linked_created} linked sequence enrollments")
    
    # 4. Force create broadcast messages for ready contacts
    print("\n4. CREATING BROADCAST MESSAGES:")
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
            COALESCE(ss.message_type, 'text'),
            ss.content,
            'pending',
            NOW(),
            NOW()
        FROM sequence_contacts sc
        JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
        WHERE sc.status = 'pending'
        AND sc.next_trigger_time <= NOW()
        AND sc.processing_device_id IS NULL
        AND NOT EXISTS (
            SELECT 1 FROM broadcast_messages bm
            WHERE bm.recipient_phone = sc.contact_phone
            AND bm.sequence_stepid = sc.sequence_stepid
            AND bm.status IN ('pending', 'sent', 'delivered')
        )
        LIMIT 50
    """)
    
    messages_created = cur.rowcount
    print(f"Created {messages_created} broadcast messages")
    
    # 5. Show current status
    print("\n5. CURRENT STATUS:")
    cur.execute("""
        SELECT 
            s.name as sequence_name,
            COUNT(DISTINCT sc.contact_phone) as contacts,
            SUM(CASE WHEN sc.status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN sc.status = 'sent' THEN 1 ELSE 0 END) as sent,
            SUM(CASE WHEN sc.status = 'completed' THEN 1 ELSE 0 END) as completed
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        GROUP BY s.name
    """)
    
    for row in cur.fetchall():
        print(f"  {row[0]}: {row[1]} contacts, {row[2]} pending, {row[3]} sent, {row[4]} completed")
    
    # Commit all changes
    conn.commit()
    print("\n✅ All fixes applied!")
    
    print("\n💡 RECOMMENDATIONS:")
    print("1. Monitor if broadcast messages start sending")
    print("2. Check if linked sequences are now being created")
    print("3. Verify the sequence processor is running")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
