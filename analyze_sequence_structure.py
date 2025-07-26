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
    
    print("=== UNDERSTANDING SEQUENCE STRUCTURE ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. broadcast_messages structure
    print("1. BROADCAST_MESSAGES TABLE STRUCTURE:")
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns
        WHERE table_name = 'broadcast_messages'
        ORDER BY ordinal_position
    """)
    
    print("\nColumns:")
    for col in cur.fetchall():
        print(f"  - {col[0]:<25} {col[1]:<20} {col[2]}")
    
    # 2. sequences structure
    print("\n\n2. SEQUENCES TABLE STRUCTURE:")
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns
        WHERE table_name = 'sequences'
        ORDER BY ordinal_position
        LIMIT 15
    """)
    
    print("\nKey columns:")
    for col in cur.fetchall():
        print(f"  - {col[0]:<25} {col[1]:<20} {col[2]}")
    
    # 3. sequence_steps structure
    print("\n\n3. SEQUENCE_STEPS TABLE STRUCTURE:")
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns
        WHERE table_name = 'sequence_steps'
        ORDER BY ordinal_position
    """)
    
    print("\nColumns:")
    for col in cur.fetchall():
        print(f"  - {col[0]:<25} {col[1]:<20} {col[2]}")
    
    # 4. sequence_contacts structure (to be replaced)
    print("\n\n4. SEQUENCE_CONTACTS TABLE STRUCTURE (to skip):")
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns
        WHERE table_name = 'sequence_contacts'
        ORDER BY ordinal_position
        LIMIT 10
    """)
    
    print("\nColumns:")
    for col in cur.fetchall():
        print(f"  - {col[0]:<25} {col[1]:<20} {col[2]}")
    
    # 5. Check sequence linking
    print("\n\n5. SEQUENCE LINKING ANALYSIS:")
    cur.execute("""
        SELECT 
            s1.name as sequence_name,
            ss1.day_number,
            ss1.trigger,
            ss1.next_trigger,
            s2.name as linked_sequence_name,
            ss1.trigger_delay_hours
        FROM sequence_steps ss1
        LEFT JOIN sequences s1 ON s1.id = ss1.sequence_id
        LEFT JOIN sequence_steps ss2 ON ss2.trigger = ss1.next_trigger AND ss2.is_entry_point = true
        LEFT JOIN sequences s2 ON s2.id = ss2.sequence_id
        WHERE ss1.next_trigger IS NOT NULL
        AND ss1.next_trigger != ''
        ORDER BY s1.name, ss1.day_number
    """)
    
    print("\nSequence Links Found:")
    for link in cur.fetchall():
        print(f"  {link[0]} Step {link[1]}: {link[2]} -> {link[3]} (links to: {link[4] or 'NOT FOUND'})")
        print(f"    Delay: {link[5]} hours")
    
    # 6. Sample sequence flow
    print("\n\n6. SAMPLE SEQUENCE FLOW:")
    cur.execute("""
        SELECT 
            s.name,
            ss.day_number,
            ss.trigger,
            ss.next_trigger,
            ss.trigger_delay_hours,
            LEFT(ss.content, 50) as message_preview
        FROM sequence_steps ss
        JOIN sequences s ON s.id = ss.sequence_id
        WHERE s.is_active = true
        ORDER BY s.name, ss.day_number
        LIMIT 10
    """)
    
    print("\nSequence Steps:")
    for step in cur.fetchall():
        print(f"\n  {step[0]} - Step {step[1]}:")
        print(f"    Trigger: {step[2]}")
        print(f"    Next: {step[3] or 'END'}")
        print(f"    Delay: {step[4]} hours")
        print(f"    Message: {step[5]}...")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
