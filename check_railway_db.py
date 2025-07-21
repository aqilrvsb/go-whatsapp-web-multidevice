import psycopg2
import sys

# Direct Railway PostgreSQL connection
conn_params = {
    "host": "autorack.proxy.rlwy.net",
    "port": 24400,
    "database": "railway",
    "user": "postgres",
    "password": "CNFPbgfjsIVirTuqLMoObNMvoYobDDTU"
}

try:
    print("Connecting to Railway PostgreSQL...")
    conn = psycopg2.connect(**conn_params)
    cur = conn.cursor()
    print("✅ Connected successfully!\n")
    
    # Check broadcast_messages structure
    print("=== BROADCAST_MESSAGES TABLE STRUCTURE ===")
    cur.execute("""
        SELECT column_name, data_type, is_nullable
        FROM information_schema.columns 
        WHERE table_name = 'broadcast_messages'
        ORDER BY ordinal_position
    """)
    
    columns = cur.fetchall()
    print(f"Total columns: {len(columns)}\n")
    
    # Look for specific columns
    has_sequence_stepid = False
    for col_name, data_type, nullable in columns:
        if col_name in ['id', 'campaign_id', 'sequence_id', 'sequence_stepid', 'recipient_name', 'recipient_phone']:
            print(f"  {col_name:<20} {data_type:<15} {'NULL' if nullable == 'YES' else 'NOT NULL'}")
            if col_name == 'sequence_stepid':
                has_sequence_stepid = True
    
    print(f"\n❓ Has sequence_stepid column: {has_sequence_stepid}")
    
    if not has_sequence_stepid:
        print("\n🔧 Adding sequence_stepid column...")
        try:
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
                WHERE sequence_stepid IS NOT NULL
            """)
            conn.commit()
            print("✅ Created index on sequence_stepid")
            
        except psycopg2.errors.DuplicateColumn:
            print("⚠️  Column already exists (this is fine)")
            conn.rollback()
        except Exception as e:
            print(f"❌ Error: {e}")
            conn.rollback()
    
    # Check sequence_steps delay columns
    print("\n=== SEQUENCE_STEPS DELAY COLUMNS ===")
    cur.execute("""
        SELECT column_name, data_type
        FROM information_schema.columns 
        WHERE table_name = 'sequence_steps'
        AND column_name LIKE '%delay%'
        ORDER BY column_name
    """)
    
    delay_cols = cur.fetchall()
    for col_name, data_type in delay_cols:
        print(f"  {col_name}: {data_type}")
    
    # Check sample sequence steps with delays
    print("\n=== SAMPLE SEQUENCE STEPS WITH DELAYS ===")
    cur.execute("""
        SELECT 
            ss.day_number,
            ss.min_delay_seconds,
            ss.max_delay_seconds,
            s.name as sequence_name
        FROM sequence_steps ss
        JOIN sequences s ON s.id = ss.sequence_id
        ORDER BY s.name, ss.day_number
        LIMIT 10
    """)
    
    steps = cur.fetchall()
    if steps:
        current_seq = None
        for day, min_delay, max_delay, seq_name in steps:
            if seq_name != current_seq:
                print(f"\n  Sequence: {seq_name}")
                current_seq = seq_name
            print(f"    Step {day}: {min_delay}-{max_delay} seconds")
    else:
        print("  No sequence steps found")
    
    # Check if there are any sequence messages without stepid
    print("\n=== SEQUENCE MESSAGES STATUS ===")
    cur.execute("""
        SELECT 
            COUNT(*) as total_sequence_messages,
            COUNT(sequence_stepid) as with_stepid,
            COUNT(*) - COUNT(sequence_stepid) as missing_stepid
        FROM broadcast_messages 
        WHERE sequence_id IS NOT NULL
    """)
    
    result = cur.fetchone()
    if result:
        print(f"  Total sequence messages: {result[0]}")
        print(f"  With sequence_stepid: {result[1]}")
        print(f"  Missing sequence_stepid: {result[2]}")
    
    cur.close()
    conn.close()
    
    print("\n✅ Database check complete!")
    print("\n📝 Summary:")
    print("- The broadcast processor has been updated to use sequence_steps delays")
    print("- Each sequence step can now have different min/max delays")
    print("- The system will respect per-step delays instead of global sequence delays")
    
except Exception as e:
    print(f"❌ Error: {e}")
    sys.exit(1)
