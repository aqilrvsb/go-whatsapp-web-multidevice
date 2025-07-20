import psycopg2

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def check_broadcast_schema():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== BROADCAST_MESSAGES SCHEMA ===")
    
    try:
        # Get all columns
        cur.execute("""
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'broadcast_messages'
            ORDER BY ordinal_position
        """)
        columns = cur.fetchall()
        
        print("\nColumns:")
        for name, dtype, nullable in columns:
            print(f"   {name} ({dtype}) - Nullable: {nullable}")
            
        # Check a sample record
        print("\n\nSample record:")
        cur.execute("""
            SELECT * FROM broadcast_messages
            WHERE sequence_id IS NOT NULL
            LIMIT 1
        """)
        
        # Get column names
        col_names = [desc[0] for desc in cur.description]
        record = cur.fetchone()
        
        if record:
            for i, (col, val) in enumerate(zip(col_names, record)):
                if val and isinstance(val, str) and len(str(val)) > 100:
                    print(f"   {col}: {str(val)[:100]}...")
                else:
                    print(f"   {col}: {val}")
                    
    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    check_broadcast_schema()
