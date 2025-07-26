import psycopg2
import pandas as pd
from datetime import datetime

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def analyze_database():
    try:
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        print("Connected to database successfully\n")
        
        # Get all tables
        cur.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'public'
            ORDER BY table_name
        """)
        tables = cur.fetchall()
        print("=== ALL TABLES ===")
        for table in tables:
            print(f"- {table[0]}")
        
        print("\n=== SEQUENCE-RELATED TABLES ===")
        
        # 1. Analyze sequences table
        print("\n1. SEQUENCES TABLE:")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequences'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
        
        # Sample data
        cur.execute("SELECT * FROM sequences LIMIT 3")
        sequences = cur.fetchall()
        print("\n  Sample sequences:")
        for seq in sequences:
            print(f"    {seq}")
        
        # 2. Analyze sequence_steps table
        print("\n2. SEQUENCE_STEPS TABLE:")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequence_steps'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
        
        # Sample data
        cur.execute("SELECT * FROM sequence_steps LIMIT 5")
        steps = cur.fetchall()
        print("\n  Sample steps:")
        for step in steps:
            print(f"    {step}")
        
        # 3. Analyze sequence_contacts table
        print("\n3. SEQUENCE_CONTACTS TABLE:")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'sequence_contacts'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
        
        # Count records
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        count = cur.fetchone()[0]
        print(f"\n  Total records: {count}")
        
        # 4. Analyze broadcast_messages table
        print("\n4. BROADCAST_MESSAGES TABLE:")
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'broadcast_messages'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        for col in columns:
            print(f"  - {col[0]}: {col[1]} (nullable: {col[2]})")
        
        # Check for scheduled_at field
        cur.execute("""
            SELECT column_name 
            FROM information_schema.columns
            WHERE table_name = 'broadcast_messages' 
            AND column_name = 'scheduled_at'
        """)
        has_scheduled = cur.fetchone()
        print(f"\n  Has scheduled_at column: {has_scheduled is not None}")
        
        # Check status values
        cur.execute("""
            SELECT DISTINCT status 
            FROM broadcast_messages 
            WHERE status IS NOT NULL
            LIMIT 10
        """)
        statuses = cur.fetchall()
        print("\n  Status values found:")
        for status in statuses:
            print(f"    - {status[0]}")
        
        # 5. Check sequence linking
        print("\n5. SEQUENCE LINKING:")
        cur.execute("""
            SELECT id, name, next_trigger 
            FROM sequences 
            WHERE next_trigger IS NOT NULL
            LIMIT 5
        """)
        linked = cur.fetchall()
        print("  Sequences with next_trigger:")
        for link in linked:
            print(f"    - {link[0]}: {link[1]} -> {link[2]}")
        
        conn.close()
        print("\nAnalysis complete!")
        
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    analyze_database()
