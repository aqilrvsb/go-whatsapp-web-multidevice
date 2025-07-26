import psycopg2
import pandas as pd
from datetime import datetime

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def analyze_sequence_details():
    try:
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        print("Connected to PostgreSQL database successfully\n")
        
        # 1. Check sequences table structure
        print("=== SEQUENCES TABLE FULL STRUCTURE ===")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequences'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
            
        # 2. Check if sequences link through steps' next_trigger
        print("\n=== SEQUENCE LINKING VIA STEPS ===")
        cur.execute("""
            SELECT DISTINCT 
                s1.name as current_sequence,
                ss.trigger as current_trigger,
                ss.next_trigger,
                s2.name as next_sequence_name
            FROM sequences s1
            JOIN sequence_steps ss ON s1.id = ss.sequence_id
            LEFT JOIN sequences s2 ON ss.next_trigger = s2.trigger
            WHERE ss.next_trigger IS NOT NULL AND ss.next_trigger != ''
            ORDER BY s1.name
            LIMIT 20
        """)
        links = cur.fetchall()
        print("Sequence links through steps:")
        for link in links:
            print(f"  - {link[0]} ({link[1]}) -> next_trigger: '{link[2]}' -> {link[3]}")
            
        # 3. Show complete flow for one sequence
        print("\n=== EXAMPLE SEQUENCE FLOW ===")
        cur.execute("""
            SELECT s.name, s.trigger, ss.day_number, ss.trigger, ss.next_trigger, 
                   ss.trigger_delay_hours, ss.is_entry_point
            FROM sequences s
            JOIN sequence_steps ss ON s.id = ss.sequence_id
            WHERE s.name LIKE '%COLD%'
            ORDER BY s.name, ss.day_number
            LIMIT 10
        """)
        steps = cur.fetchall()
        current_seq = None
        for step in steps:
            if current_seq != step[0]:
                current_seq = step[0]
                print(f"\nSequence: {step[0]} (trigger: {step[1]})")
            print(f"  Step {step[2]}: trigger='{step[3]}', next='{step[4]}', delay={step[5]}hrs, entry={step[6]}")
            
        # 4. Check how sequence contacts are created
        print("\n=== SEQUENCE CONTACTS SAMPLE ===")
        cur.execute("""
            SELECT sc.sequence_id, s.name, sc.contact_phone, sc.current_step, 
                   sc.status, sc.next_trigger_time, sc.sequence_stepid
            FROM sequence_contacts sc
            JOIN sequences s ON s.id = sc.sequence_id
            ORDER BY sc.contact_phone, sc.current_step
            LIMIT 20
        """)
        contacts = cur.fetchall()
        print("Sample sequence contacts:")
        current_phone = None
        for contact in contacts:
            if current_phone != contact[2]:
                current_phone = contact[2]
                print(f"\nContact: {contact[2]}")
            print(f"  - {contact[1]} Step {contact[3]}: status={contact[4]}, trigger_time={contact[5]}")
            
        # 5. Check broadcast messages with sequence info
        print("\n=== BROADCAST MESSAGES FROM SEQUENCES ===")
        cur.execute("""
            SELECT bm.recipient_phone, s.name, bm.status, bm.scheduled_at, 
                   bm.sequence_stepid, bm.created_at
            FROM broadcast_messages bm
            JOIN sequences s ON s.id = bm.sequence_id
            WHERE bm.sequence_id IS NOT NULL
            ORDER BY bm.created_at DESC
            LIMIT 15
        """)
        messages = cur.fetchall()
        print("Recent sequence messages:")
        for msg in messages:
            print(f"  - {msg[0]} | {msg[1]} | {msg[2]} | scheduled: {msg[3]} | created: {msg[5]}")
            
        # 6. Check if there's a constraint on sequence_contacts
        print("\n=== SEQUENCE_CONTACTS CONSTRAINTS ===")
        cur.execute("""
            SELECT constraint_name, constraint_type
            FROM information_schema.table_constraints
            WHERE table_name = 'sequence_contacts'
        """)
        constraints = cur.fetchall()
        for constraint in constraints:
            print(f"  - {constraint[0]}: {constraint[1]}")
            
        conn.close()
        print("\n[DONE] Analysis complete!")
        
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    analyze_sequence_details()
