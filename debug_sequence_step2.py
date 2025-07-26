# -*- coding: utf-8 -*-
import psycopg2
import sys
from datetime import datetime

# Set UTF-8 encoding for output
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== SEQUENCE DEBUGGING - STEP 2 ONLY ISSUE ===")
    print(f"Connected at {datetime.now()}\n")
    
    # 1. Check sequence steps structure
    print("1. CHECKING SEQUENCE STEPS:")
    cur.execute("""
        SELECT 
            ss.id,
            ss.sequence_id,
            s.name as sequence_name,
            ss.day_number,
            ss.delay_days,
            ss.min_delay_seconds,
            ss.max_delay_seconds,
            LEFT(ss.content, 50) as message_preview
        FROM sequence_steps ss
        JOIN sequences s ON s.id = ss.sequence_id
        WHERE s.is_active = true
        ORDER BY s.name, ss.day_number
    """)
    
    steps = cur.fetchall()
    print(f"Total active sequence steps: {len(steps)}")
    current_seq = None
    for step in steps:
        if current_seq != step[2]:
            current_seq = step[2]
            print(f"\n  Sequence: {current_seq}")
        print(f"    Day {step[3]}: delay_days={step[4]}, delay_sec={step[5]}-{step[6]}, msg={step[7]}...")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Check sequence contacts and their current step
    print("2. SEQUENCE CONTACTS BY STEP:")
    cur.execute("""
        SELECT 
            current_step,
            status,
            COUNT(*) as count
        FROM sequence_contacts
        GROUP BY current_step, status
        ORDER BY current_step, status
    """)
    
    step_counts = cur.fetchall()
    print("Step distribution:")
    for sc in step_counts:
        print(f"  Step {sc[0] or 'NULL'}, Status '{sc[1]}': {sc[2]} contacts")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Check messages created per step
    print("3. MESSAGES CREATED PER STEP:")
    cur.execute("""
        SELECT 
            ss.day_number,
            bm.status,
            COUNT(*) as count
        FROM broadcast_messages bm
        JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
        WHERE bm.sequence_id IS NOT NULL
        GROUP BY ss.day_number, bm.status
        ORDER BY ss.day_number, bm.status
    """)
    
    msg_counts = cur.fetchall()
    print("Messages by step and status:")
    for mc in msg_counts:
        print(f"  Step {mc[0]}, Status '{mc[1]}': {mc[2]} messages")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Check recent sequence contact details
    print("4. RECENT SEQUENCE ENROLLMENTS (last 10):")
    cur.execute("""
        SELECT 
            sc.id,
            sc.contact_phone,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.created_at,
            sc.sequence_stepid,
            ss.day_number as step_day
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        LEFT JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
        ORDER BY sc.created_at DESC
        LIMIT 10
    """)
    
    recent = cur.fetchall()
    for r in recent:
        print(f"\n  Contact: {r[1]}")
        print(f"    Sequence: {r[2]}, Current Step: {r[3]}, Status: {r[4]}")
        print(f"    Created: {r[5]}, Step ID: {r[6]}, Step Day: {r[7]}")
        
        # Check messages for this contact
        cur.execute("""
            SELECT 
                bm.id,
                ss.day_number,
                bm.status,
                bm.created_at,
                bm.sent_at
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_stepid
            WHERE bm.recipient_phone = %s 
            AND bm.sequence_id = (SELECT sequence_id FROM sequence_contacts WHERE id = %s)
            ORDER BY bm.created_at
        """, (r[1], r[0]))
        
        msgs = cur.fetchall()
        print(f"    Messages: {len(msgs)}")
        for m in msgs:
            print(f"      - Step {m[1]}: {m[2]}, Created: {m[3]}, Sent: {m[4]}")
    
    print("\n" + "="*80 + "\n")
    
    # 5. Check if there's an issue with step 1 creation
    print("5. CHECKING STEP 1 ISSUES:")
    
    # Check if step 1 exists for all sequences
    cur.execute("""
        SELECT 
            s.id,
            s.name,
            MIN(ss.day_number) as min_day,
            COUNT(ss.id) as step_count
        FROM sequences s
        LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
        WHERE s.is_active = true
        GROUP BY s.id, s.name
        HAVING MIN(ss.day_number) > 1 OR MIN(ss.day_number) IS NULL
    """)
    
    missing_step1 = cur.fetchall()
    if missing_step1:
        print("WARNING: Sequences without step 1:")
        for ms in missing_step1:
            print(f"  - {ms[1]}: min_day={ms[2]}, steps={ms[3]}")
    else:
        print("All active sequences have step 1")
    
    # Check sequence_contacts with NULL sequence_stepid
    cur.execute("""
        SELECT COUNT(*) 
        FROM sequence_contacts 
        WHERE sequence_stepid IS NULL
    """)
    
    null_count = cur.fetchone()[0]
    print(f"\nContacts with NULL sequence_stepid: {null_count}")
    
    print("\n" + "="*80 + "\n")
    
    # 6. Check the sequence processor logic
    print("6. POTENTIAL ISSUES:")
    
    # Check for contacts stuck at step 1
    cur.execute("""
        SELECT 
            sc.id,
            sc.contact_phone,
            s.name,
            sc.created_at,
            sc.current_step,
            sc.sequence_stepid,
            CASE 
                WHEN bm.id IS NULL THEN 'No message created'
                ELSE 'Message exists'
            END as message_status
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        LEFT JOIN broadcast_messages bm ON bm.recipient_phone = sc.contact_phone 
            AND bm.sequence_id = sc.sequence_id
            AND bm.sequence_stepid = sc.sequence_stepid
        WHERE sc.current_step = 1
        AND sc.status = 'active'
        ORDER BY sc.created_at DESC
        LIMIT 5
    """)
    
    stuck = cur.fetchall()
    if stuck:
        print("Contacts stuck at step 1:")
        for s in stuck:
            print(f"  - {s[1]} in '{s[2]}': created {s[3]}, step_id={s[5]}, {s[6]}")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
