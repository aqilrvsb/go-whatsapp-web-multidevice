import psycopg2
import pandas as pd
from datetime import datetime
import json

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== SEQUENCE DEBUGGING ===")
    print(f"Connected to database at {datetime.now()}")
    print("\n" + "="*80 + "\n")
    
    # 1. Check sequence_steps table
    print("1. SEQUENCE STEPS TABLE:")
    cur.execute("""
        SELECT 
            ss.id,
            ss.sequence_id,
            s.name as sequence_name,
            ss.step_number,
            ss.message,
            ss.delay_minutes,
            ss.min_delay_seconds,
            ss.max_delay_seconds,
            ss.created_at
        FROM sequence_steps ss
        JOIN sequences s ON s.id = ss.sequence_id
        ORDER BY ss.sequence_id, ss.step_number
    """)
    
    steps = cur.fetchall()
    print(f"Total sequence steps: {len(steps)}")
    print("\nStep Details:")
    for step in steps:
        print(f"  Step ID: {step[0]}, Sequence: {step[2]}, Step #{step[3]}, Delay: {step[5]} min, Message: {step[4][:50]}...")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Check sequence_contacts table
    print("2. SEQUENCE CONTACTS TABLE:")
    cur.execute("""
        SELECT 
            sc.id,
            sc.lead_id,
            sc.sequence_id,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.enrolled_at,
            sc.last_sent_at,
            sc.next_send_at,
            sc.created_at
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        ORDER BY sc.created_at DESC
        LIMIT 20
    """)
    
    contacts = cur.fetchall()
    print(f"Recent sequence enrollments (last 20):")
    for contact in contacts:
        print(f"  Contact ID: {contact[0]}, Lead: {contact[1]}, Sequence: {contact[3]}, Step: {contact[4]}, Status: {contact[5]}")
        print(f"    Enrolled: {contact[6]}, Last Sent: {contact[7]}, Next Send: {contact[8]}")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Check broadcast_messages table for sequence messages
    print("3. BROADCAST MESSAGES TABLE (Sequence Messages):")
    cur.execute("""
        SELECT 
            bm.id,
            bm.message_id,
            bm.lead_id,
            bm.phone_number,
            bm.message,
            bm.status,
            bm.sequence_id,
            bm.sequence_step_id,
            bm.created_at,
            bm.sent_at,
            ss.step_number
        FROM broadcast_messages bm
        LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_step_id
        WHERE bm.sequence_id IS NOT NULL
        ORDER BY bm.created_at DESC
        LIMIT 30
    """)
    
    messages = cur.fetchall()
    print(f"Recent sequence messages (last 30):")
    for msg in messages:
        print(f"  Msg ID: {msg[0]}, Lead: {msg[2]}, Status: {msg[5]}, Seq ID: {msg[6]}, Step ID: {msg[7]} (Step #{msg[10]})")
        print(f"    Created: {msg[8]}, Sent: {msg[9]}")
        print(f"    Message: {msg[4][:50]}...")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Check for anomalies
    print("4. CHECKING FOR ANOMALIES:")
    
    # Check if all steps exist for sequences
    cur.execute("""
        SELECT 
            s.id,
            s.name,
            COUNT(ss.id) as step_count,
            STRING_AGG(ss.step_number::text, ', ' ORDER BY ss.step_number) as steps
        FROM sequences s
        LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
        GROUP BY s.id, s.name
    """)
    
    seq_steps = cur.fetchall()
    print("\nSteps per sequence:")
    for seq in seq_steps:
        print(f"  Sequence {seq[0]} ({seq[1]}): {seq[2]} steps - [{seq[3]}]")
    
    # Check contacts stuck on step 1
    cur.execute("""
        SELECT 
            COUNT(*) as stuck_count,
            sc.current_step,
            sc.status
        FROM sequence_contacts sc
        WHERE sc.current_step = 1
        GROUP BY sc.current_step, sc.status
    """)
    
    stuck = cur.fetchall()
    print("\nContacts on step 1:")
    for s in stuck:
        print(f"  Step {s[1]}, Status '{s[2]}': {s[0]} contacts")
    
    # Check if step 1 messages are being created
    cur.execute("""
        SELECT 
            ss.step_number,
            COUNT(bm.id) as message_count
        FROM sequence_steps ss
        LEFT JOIN broadcast_messages bm ON bm.sequence_step_id = ss.id
        GROUP BY ss.step_number
        ORDER BY ss.step_number
    """)
    
    step_msgs = cur.fetchall()
    print("\nMessages created per step:")
    for sm in step_msgs:
        print(f"  Step {sm[0]}: {sm[1]} messages")
    
    print("\n" + "="*80 + "\n")
    
    # 5. Check specific sequence flow
    print("5. DETAILED SEQUENCE FLOW CHECK:")
    
    # Get a sample contact to trace
    cur.execute("""
        SELECT 
            sc.id,
            sc.lead_id,
            sc.sequence_id,
            sc.current_step,
            sc.status,
            sc.enrolled_at,
            l.name as lead_name,
            l.phone
        FROM sequence_contacts sc
        JOIN leads_ai l ON l.id = sc.lead_id
        WHERE sc.sequence_id IS NOT NULL
        ORDER BY sc.enrolled_at DESC
        LIMIT 5
    """)
    
    sample_contacts = cur.fetchall()
    print("\nSample contacts to trace:")
    for sc in sample_contacts:
        print(f"\n  Contact {sc[0]}: Lead {sc[6]} ({sc[7]})")
        print(f"    Sequence ID: {sc[2]}, Current Step: {sc[3]}, Status: {sc[4]}")
        
        # Check messages for this contact
        cur.execute("""
            SELECT 
                bm.id,
                bm.status,
                bm.sequence_step_id,
                ss.step_number,
                bm.created_at,
                bm.sent_at
            FROM broadcast_messages bm
            LEFT JOIN sequence_steps ss ON ss.id = bm.sequence_step_id
            WHERE bm.lead_id = %s AND bm.sequence_id = %s
            ORDER BY bm.created_at
        """, (sc[1], sc[2]))
        
        contact_msgs = cur.fetchall()
        print(f"    Messages created: {len(contact_msgs)}")
        for cm in contact_msgs:
            print(f"      - Step {cm[3]}: Status={cm[1]}, Created={cm[4]}, Sent={cm[5]}")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
