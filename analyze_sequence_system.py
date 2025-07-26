import psycopg2
import pandas as pd
from datetime import datetime

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def analyze_sequence_system():
    try:
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        print("Connected to PostgreSQL database successfully\n")
        
        # 1. Check broadcast_messages table structure
        print("=== BROADCAST_MESSAGES TABLE STRUCTURE ===")
        cur.execute("""
            SELECT column_name, data_type, is_nullable, column_default
            FROM information_schema.columns
            WHERE table_name = 'broadcast_messages'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        print("Columns in broadcast_messages:")
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]}, default: {col[3]})")
        
        # Check if scheduled_at column exists
        scheduled_at_exists = any(col[0] == 'scheduled_at' for col in columns)
        print(f"\n[CHECK] scheduled_at column exists: {scheduled_at_exists}")
        
        # 2. Check sequence_steps structure
        print("\n=== SEQUENCE_STEPS TABLE STRUCTURE ===")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequence_steps'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
            
        # 3. Check sequences table for next_trigger
        print("\n=== SEQUENCES TABLE - NEXT_TRIGGER ===")
        cur.execute("""
            SELECT column_name, data_type 
            FROM information_schema.columns
            WHERE table_name = 'sequences' AND column_name = 'next_trigger'
        """)
        next_trigger_col = cur.fetchone()
        if next_trigger_col:
            print(f"[CHECK] next_trigger column exists: {next_trigger_col[1]}")
        
        # 4. Check sequence_contacts structure
        print("\n=== SEQUENCE_CONTACTS TABLE STRUCTURE ===")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequence_contacts'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
            
        # 5. Check status values in broadcast_messages
        print("\n=== BROADCAST_MESSAGES STATUS VALUES ===")
        cur.execute("""
            SELECT DISTINCT status, COUNT(*) as count
            FROM broadcast_messages
            WHERE status IS NOT NULL
            GROUP BY status
            ORDER BY count DESC
        """)
        statuses = cur.fetchall()
        print("Current status values:")
        for status in statuses:
            print(f"  - {status[0]}: {status[1]} records")
            
        # 6. Check sequence linking
        print("\n=== SEQUENCE LINKING (next_trigger) ===")
        cur.execute("""
            SELECT s1.name as sequence_name, s1.trigger, s2.name as next_sequence_name
            FROM sequences s1
            LEFT JOIN sequences s2 ON s1.next_trigger = s2.trigger
            WHERE s1.next_trigger IS NOT NULL AND s1.next_trigger != ''
            LIMIT 10
        """)
        links = cur.fetchall()
        print("Sequence links found:")
        for link in links:
            print(f"  - {link[0]} ({link[1]}) -> {link[2]}")
            
        # 7. Sample sequence steps with delays
        print("\n=== SAMPLE SEQUENCE STEPS ===")
        cur.execute("""
            SELECT s.name, ss.day_number, ss.trigger, ss.next_trigger, 
                   ss.trigger_delay_hours, ss.min_delay_seconds, ss.max_delay_seconds
            FROM sequence_steps ss
            JOIN sequences s ON s.id = ss.sequence_id
            ORDER BY s.name, ss.day_number
            LIMIT 15
        """)
        steps = cur.fetchall()
        print("Sample steps:")
        for step in steps:
            print(f"  - {step[0]} Step {step[1]}: trigger='{step[2]}', delay={step[4]}hrs, min/max={step[5]}/{step[6]}s")
            
        # 8. Check current sequence enrollment process
        print("\n=== CURRENT SEQUENCE ENROLLMENT ===")
        cur.execute("""
            SELECT COUNT(*) as total_contacts,
                   COUNT(DISTINCT sequence_id) as sequences,
                   COUNT(DISTINCT contact_phone) as unique_contacts,
                   MIN(created_at) as oldest,
                   MAX(created_at) as newest
            FROM sequence_contacts
        """)
        stats = cur.fetchone()
        print(f"Total records: {stats[0]}")
        print(f"Unique sequences: {stats[1]}")
        print(f"Unique contacts: {stats[2]}")
        print(f"Date range: {stats[3]} to {stats[4]}")
        
        # 9. Check how messages are created
        print("\n=== BROADCAST_MESSAGES WITH SEQUENCE INFO ===")
        cur.execute("""
            SELECT COUNT(*) as total,
                   COUNT(sequence_id) as from_sequences,
                   COUNT(sequence_stepid) as with_step_id
            FROM broadcast_messages
        """)
        msg_stats = cur.fetchone()
        print(f"Total messages: {msg_stats[0]}")
        print(f"From sequences: {msg_stats[1]}")
        print(f"With step ID: {msg_stats[2]}")
        
        conn.close()
        print("\n[DONE] Analysis complete!")
        
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    analyze_sequence_system()
