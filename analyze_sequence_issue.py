# -*- coding: utf-8 -*-
import psycopg2
import sys
from datetime import datetime

# Set UTF-8 encoding for output
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== SEQUENCE ISSUE ANALYSIS ===")
    print(f"Connected at {datetime.now()}\n")
    
    print("ISSUE IDENTIFIED: Multiple sequence_contact records per contact!")
    print("The system is creating a new sequence_contact record for EACH step,")
    print("instead of updating the same record to progress through steps.\n")
    
    # 1. Show the duplicate issue
    print("1. DUPLICATE SEQUENCE_CONTACTS (same phone, same sequence):")
    cur.execute("""
        SELECT 
            contact_phone,
            sequence_id,
            COUNT(*) as duplicate_count,
            STRING_AGG(current_step::text, ', ' ORDER BY current_step) as steps,
            STRING_AGG(status, ', ' ORDER BY current_step) as statuses
        FROM sequence_contacts
        GROUP BY contact_phone, sequence_id
        HAVING COUNT(*) > 1
        ORDER BY COUNT(*) DESC
        LIMIT 10
    """)
    
    duplicates = cur.fetchall()
    for dup in duplicates:
        print(f"  Phone {dup[0]}: {dup[2]} records, steps=[{dup[3]}], statuses=[{dup[4]}]")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Check how the system should work
    print("2. HOW IT SHOULD WORK vs CURRENT BEHAVIOR:")
    print("\nEXPECTED FLOW:")
    print("1. Create ONE sequence_contact record when enrolling")
    print("2. Set current_step = 1, status = 'active'")
    print("3. Create broadcast_message for step 1")
    print("4. After step 1 sent, UPDATE sequence_contact: current_step = 2")
    print("5. Create broadcast_message for step 2")
    print("6. Continue updating same record...")
    
    print("\nACTUAL BEHAVIOR:")
    print("1. Creating NEW sequence_contact for each step!")
    print("2. This causes:")
    print("   - Only step 2 messages being sent (since step 1 records are marked 'completed')")
    print("   - Duplicate enrollments")
    print("   - Incorrect progress tracking")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Show the problem in detail
    print("3. EXAMPLE OF THE PROBLEM:")
    cur.execute("""
        SELECT 
            sc.id,
            sc.contact_phone,
            s.name as sequence_name,
            sc.current_step,
            sc.status,
            sc.created_at,
            sc.sequence_stepid
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        WHERE sc.contact_phone = '601119241561'
        AND s.name = 'HOT Seqeunce'
        ORDER BY sc.created_at
    """)
    
    example = cur.fetchall()
    print(f"Contact 601119241561 in 'HOT Seqeunce':")
    for ex in example:
        print(f"  Record {ex[0]}: Step {ex[3]}, Status '{ex[4]}', Created {ex[5]}")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Check the code logic
    print("4. CODE ANALYSIS NEEDED:")
    print("\nThe sequence processor is likely doing:")
    print("1. When processing step transitions, it's creating NEW sequence_contact records")
    print("2. Instead of UPDATE sequence_contacts SET current_step = current_step + 1")
    print("3. This happens in the sequence monitoring/processing logic")
    
    print("\nFILES TO CHECK:")
    print("- src/usecase/sequence_monitor.go")
    print("- src/usecase/sequence_processor.go")
    print("- src/repository/sequence_repository.go")
    
    print("\n" + "="*80 + "\n")
    
    # 5. Temporary fix suggestion
    print("5. TEMPORARY FIX (Database Cleanup):")
    print("\nTo fix existing data:")
    print("1. Keep only the latest sequence_contact record per phone/sequence")
    print("2. Delete the duplicate 'completed' records")
    print("3. Reset current_step to 1 for active sequences")
    
    # Show cleanup query
    print("\nCleanup query (DO NOT RUN YET - review first):")
    print("""
-- Delete duplicate sequence_contact records, keeping only the latest
WITH duplicates AS (
    SELECT 
        id,
        ROW_NUMBER() OVER (
            PARTITION BY contact_phone, sequence_id 
            ORDER BY created_at DESC
        ) as rn
    FROM sequence_contacts
)
DELETE FROM sequence_contacts
WHERE id IN (
    SELECT id FROM duplicates WHERE rn > 1
);

-- Reset sequences to start from step 1
UPDATE sequence_contacts
SET current_step = 1, status = 'active'
WHERE status = 'completed'
AND sequence_id IN (SELECT id FROM sequences WHERE is_active = true);
    """)
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
