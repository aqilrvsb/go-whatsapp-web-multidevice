import psycopg2

# Connect to PostgreSQL
conn = psycopg2.connect(
    "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"
)
cur = conn.cursor()

print("=== DELETING ALL SEQUENCE DATA ===\n")

try:
    # First, check how many records exist
    cur.execute("SELECT COUNT(*) FROM sequence_contacts")
    count_before = cur.fetchone()[0]
    print(f"1. Found {count_before} records in sequence_contacts")
    
    # Delete ALL sequence_contacts records
    cur.execute("DELETE FROM sequence_contacts")
    sc_deleted = cur.rowcount
    print(f"   Deleted {sc_deleted} sequence_contacts records")
    
    # Also delete ALL broadcast messages related to sequences
    cur.execute("DELETE FROM broadcast_messages WHERE sequence_id IS NOT NULL")
    bm_deleted = cur.rowcount
    print(f"   Deleted {bm_deleted} broadcast_messages with sequence_id")
    
    # Commit the changes
    conn.commit()
    print("\n2. Changes committed successfully!")
    
    # Verify deletion
    cur.execute("SELECT COUNT(*) FROM sequence_contacts")
    count_after = cur.fetchone()[0]
    print(f"\n3. Verification: {count_after} records remaining in sequence_contacts (should be 0)")
    
    # Check by sequence
    cur.execute("""
        SELECT s.name, COUNT(sc.id) as count
        FROM sequences s
        LEFT JOIN sequence_contacts sc ON sc.sequence_id = s.id
        GROUP BY s.name
        ORDER BY s.name
    """)
    results = cur.fetchall()
    print("\n4. Records per sequence:")
    for row in results:
        print(f"   {row[0]}: {row[1]} records")
    
    # Double check with a different query
    cur.execute("SELECT id FROM sequence_contacts LIMIT 5")
    remaining = cur.fetchall()
    if remaining:
        print(f"\n5. WARNING: Still found {len(remaining)} records!")
        print("   Running TRUNCATE command...")
        
        # Use TRUNCATE for more aggressive deletion
        cur.execute("TRUNCATE TABLE sequence_contacts RESTART IDENTITY CASCADE")
        conn.commit()
        print("   TRUNCATE executed successfully!")
        
        # Final check
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        final_count = cur.fetchone()[0]
        print(f"   Final count: {final_count}")
    else:
        print("\n5. SUCCESS: No records found in sequence_contacts!")
    
except Exception as e:
    conn.rollback()
    print(f"\nERROR: {e}")
    print("Trying alternative approach...")
    
    try:
        # Try without CASCADE
        cur.execute("TRUNCATE TABLE sequence_contacts RESTART IDENTITY")
        conn.commit()
        print("TRUNCATE without CASCADE successful!")
    except:
        # Last resort - delete with no conditions
        cur.execute("DELETE FROM sequence_contacts WHERE 1=1")
        conn.commit()
        print("DELETE with WHERE 1=1 successful!")

finally:
    # Final verification
    cur.execute("SELECT COUNT(*) FROM sequence_contacts")
    absolute_final = cur.fetchone()[0]
    print(f"\n=== FINAL RESULT: {absolute_final} records in sequence_contacts ===")
    
    if absolute_final == 0:
        print("✅ ALL DATA SUCCESSFULLY DELETED!")
    else:
        print("❌ Some records still remain. Manual intervention may be needed.")

cur.close()
conn.close()
