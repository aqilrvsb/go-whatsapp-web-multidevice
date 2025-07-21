import psycopg2
import sys

# Database connection from your Railway setup
conn_string = "postgresql://whatsapp_user:Cahaya123!@autorack.proxy.rlwy.net:24400/railway"

try:
    # Connect to PostgreSQL
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== CHECKING broadcast_messages TABLE ===")
    # Check if sequence_stepid column exists
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns 
        WHERE table_name = 'broadcast_messages'
        AND column_name IN ('sequence_stepid', 'sequence_id', 'campaign_id', 'recipient_name')
        ORDER BY ordinal_position
    """)
    
    print("\nRelevant columns in broadcast_messages:")
    for row in cur.fetchall():
        print(f"  - {row[0]}: {row[1]} (nullable: {row[2]})")
    
    # Check if sequence_stepid exists
    cur.execute("""
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'broadcast_messages' 
            AND column_name = 'sequence_stepid'
        )
    """)
    has_stepid = cur.fetchone()[0]
    print(f"\nHas sequence_stepid column: {has_stepid}")
    
    if not has_stepid:
        print("\n⚠️  sequence_stepid column is MISSING! Adding it now...")
        
        # Add the column
        cur.execute("""
            ALTER TABLE broadcast_messages 
            ADD COLUMN sequence_stepid UUID REFERENCES sequence_steps(id) ON DELETE SET NULL
        """)
        conn.commit()
        print("✅ Added sequence_stepid column successfully!")
        
        # Create index
        cur.execute("""
            CREATE INDEX IF NOT EXISTS idx_broadcast_messages_sequence_stepid 
            ON broadcast_messages(sequence_stepid)
        """)
        conn.commit()
        print("✅ Created index on sequence_stepid")
    
    # Check sequence_steps delay columns
    print("\n=== CHECKING sequence_steps TABLE ===")
    cur.execute("""
        SELECT column_name, data_type
        FROM information_schema.columns 
        WHERE table_name = 'sequence_steps'
        AND column_name IN ('min_delay_seconds', 'max_delay_seconds')
    """)
    
    print("\nDelay columns in sequence_steps:")
    for row in cur.fetchall():
        print(f"  - {row[0]}: {row[1]}")
    
    # Show sample data
    print("\n=== SAMPLE SEQUENCE STEPS WITH DELAYS ===")
    cur.execute("""
        SELECT id, sequence_id, day_number, min_delay_seconds, max_delay_seconds
        FROM sequence_steps
        LIMIT 5
    """)
    
    print("\nSample sequence steps:")
    for row in cur.fetchall():
        print(f"  Step {row[2]}: min_delay={row[3]}s, max_delay={row[4]}s")
    
    cur.close()
    conn.close()
    
    print("\n✅ Database check complete!")
    
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)
