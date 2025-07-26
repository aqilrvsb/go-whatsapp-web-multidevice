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
    
    print("=== AGGRESSIVE SEQUENCE FIX ===")
    print(f"Started at {datetime.now()}\n")
    
    # 1. Show the problem
    print("1. SHOWING DUPLICATES:")
    cur.execute("""
        SELECT 
            contact_phone,
            COUNT(*) as duplicate_count
        FROM sequence_contacts
        GROUP BY contact_phone
        HAVING COUNT(*) > 1
        ORDER BY COUNT(*) DESC
    """)
    
    duplicates = cur.fetchall()
    print(f"Found {len(duplicates)} contacts with duplicates")
    
    # 2. Delete ALL duplicates, keeping only the record with highest step
    print("\n2. DELETING DUPLICATES (keeping highest step per contact):")
    cur.execute("""
        WITH keep_records AS (
            SELECT DISTINCT ON (contact_phone) 
                id
            FROM sequence_contacts
            ORDER BY contact_phone, current_step DESC, created_at DESC
        )
        DELETE FROM sequence_contacts
        WHERE id NOT IN (SELECT id FROM keep_records)
    """)
    
    deleted = cur.rowcount
    print(f"Deleted {deleted} duplicate records")
    
    # 3. Verify no more duplicates
    cur.execute("""
        SELECT COUNT(*) 
        FROM (
            SELECT contact_phone
            FROM sequence_contacts
            GROUP BY contact_phone
            HAVING COUNT(*) > 1
        ) dups
    """)
    
    remaining_dups = cur.fetchone()[0]
    print(f"Remaining duplicates: {remaining_dups} (should be 0)")
    
    # 4. Reset all sequences to active for testing
    print("\n3. RESETTING SEQUENCES FOR TESTING:")
    cur.execute("""
        UPDATE sequence_contacts
        SET 
            current_step = 1,
            status = 'active',
            next_trigger_time = NOW() + INTERVAL '2 minutes'
        WHERE sequence_id IN (
            SELECT id FROM sequences WHERE is_active = true
        )
    """)
    
    reset_count = cur.rowcount
    print(f"Reset {reset_count} contacts to step 1, active status")
    
    # 5. Show final state
    print("\n4. FINAL STATE:")
    cur.execute("""
        SELECT 
            COUNT(DISTINCT contact_phone) as unique_contacts,
            COUNT(*) as total_records,
            SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_contacts
        FROM sequence_contacts
    """)
    
    final = cur.fetchone()
    print(f"Unique contacts: {final[0]}")
    print(f"Total records: {final[1]} (should equal unique contacts)")
    print(f"Active contacts: {final[2]}")
    
    # Commit
    conn.commit()
    print("\n✅ All changes committed!")
    
    # Show some sample data
    print("\n5. SAMPLE DATA (first 5 contacts):")
    cur.execute("""
        SELECT 
            sc.contact_phone,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.next_trigger_time
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        ORDER BY sc.next_trigger_time
        LIMIT 5
    """)
    
    for row in cur.fetchall():
        print(f"  {row[0]} - {row[1]}: Step {row[2]}, {row[3]}, Next: {row[4]}")
    
    cur.close()
    conn.close()
    
    print("\n✅ Fix completed! Sequences should start processing in 2 minutes.")
    print("\n⚠️  IMPORTANT: The code still needs to be fixed to prevent creating duplicates!")
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
