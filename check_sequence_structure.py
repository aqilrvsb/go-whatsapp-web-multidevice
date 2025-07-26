import psycopg2
import pandas as pd
from datetime import datetime

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== SEQUENCE TABLE STRUCTURE ANALYSIS ===")
    print(f"Connected at {datetime.now()}\n")
    
    # 1. Get table structures
    tables = ['sequences', 'sequence_steps', 'sequence_contacts', 'broadcast_messages']
    
    for table in tables:
        print(f"\n{table.upper()} TABLE COLUMNS:")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = %s
            ORDER BY ordinal_position
        """, (table,))
        
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]:<30} {col[1]:<20} {col[2]}")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Check sequence_steps data
    print("SEQUENCE STEPS DATA:")
    cur.execute("""
        SELECT * FROM sequence_steps 
        ORDER BY sequence_id, id 
        LIMIT 10
    """)
    
    steps = cur.fetchall()
    col_names = [desc[0] for desc in cur.description]
    print(f"Columns: {col_names}")
    print(f"\nFirst 10 steps:")
    for step in steps:
        print(f"  {step}")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Check sequence_contacts data
    print("SEQUENCE CONTACTS DATA:")
    cur.execute("""
        SELECT * FROM sequence_contacts 
        ORDER BY created_at DESC 
        LIMIT 10
    """)
    
    contacts = cur.fetchall()
    col_names = [desc[0] for desc in cur.description]
    print(f"Columns: {col_names}")
    print(f"\nLast 10 contacts:")
    for contact in contacts:
        print(f"  {contact}")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Check broadcast_messages for sequences
    print("BROADCAST MESSAGES (Sequence-related):")
    cur.execute("""
        SELECT * FROM broadcast_messages 
        WHERE sequence_id IS NOT NULL 
        ORDER BY created_at DESC 
        LIMIT 10
    """)
    
    messages = cur.fetchall()
    col_names = [desc[0] for desc in cur.description]
    print(f"Columns: {col_names}")
    print(f"\nLast 10 sequence messages:")
    for msg in messages:
        print(f"  {msg}")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
