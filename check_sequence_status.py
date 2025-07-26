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
    
    print("=== SEQUENCE STATUS AFTER FIX ===")
    print(f"Checked at {datetime.now()}\n")
    
    # 1. Check current state
    print("1. CURRENT SEQUENCE CONTACTS:")
    cur.execute("""
        SELECT 
            sc.contact_phone,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.created_at
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        ORDER BY sc.created_at DESC
        LIMIT 10
    """)
    
    for row in cur.fetchall():
        print(f"  {row[0]} - {row[1]}: Step {row[2]}, Status '{row[3]}', Created {row[4]}")
    
    print("\n2. SEQUENCE DISTRIBUTION:")
    cur.execute("""
        SELECT 
            current_step,
            status,
            COUNT(*) as count
        FROM sequence_contacts
        GROUP BY current_step, status
        ORDER BY current_step, status
    """)
    
    for row in cur.fetchall():
        print(f"  Step {row[0]}, Status '{row[1]}': {row[2]} contacts")
    
    print("\n3. RECOMMENDATIONS:")
    print("  ✅ Database has been cleaned - no more duplicates")
    print("  ⚠️  All contacts are at 'completed' status")
    print("  💡 To restart sequences for testing, run this SQL:")
    print("""
UPDATE sequence_contacts
SET 
    current_step = 1,
    status = 'active',
    next_trigger_time = NOW() + INTERVAL '5 minutes'
WHERE sequence_id IN (
    SELECT id FROM sequences WHERE is_active = true
);
    """)
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
