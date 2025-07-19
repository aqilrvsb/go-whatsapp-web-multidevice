import psycopg2
from datetime import datetime

DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def diagnose_sequence_issue():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== SEQUENCE DIAGNOSTIC REPORT ===")
    print(f"Time: {datetime.now()}")
    
    # 1. Check current status distribution
    print("\n1. Status Distribution:")
    cur.execute("""
        SELECT status, COUNT(*) as count
        FROM sequence_contacts
        GROUP BY status
        ORDER BY status
    """)
    for status, count in cur.fetchall():
        print(f"   {status}: {count}")
    
    # 2. Check if multiple steps are active for same contact
    print("\n2. Contacts with Multiple Active Steps:")
    cur.execute("""
        SELECT contact_phone, COUNT(*) as active_count
        FROM sequence_contacts
        WHERE status = 'active'
        GROUP BY contact_phone
        HAVING COUNT(*) > 1
    """)
    multi_active = cur.fetchall()
    if multi_active:
        print("   PROBLEM FOUND! These contacts have multiple active steps:")
        for phone, count in multi_active:
            print(f"   {phone}: {count} active steps")
    else:
        print("   OK - No contacts have multiple active steps")
    
    # 3. Check timing issues
    print("\n3. Active Steps with Past Due Times:")
    cur.execute("""
        SELECT contact_phone, current_step, next_trigger_time, 
               EXTRACT(EPOCH FROM (NOW() - next_trigger_time))/60 as minutes_overdue
        FROM sequence_contacts
        WHERE status = 'active'
        AND next_trigger_time < NOW()
        ORDER BY next_trigger_time
        LIMIT 10
    """)
    overdue = cur.fetchall()
    if overdue:
        print("   These active steps are ready to process:")
        for phone, step, trigger_time, minutes in overdue:
            print(f"   {phone} - Step {step}: Due {trigger_time} ({minutes:.1f} min overdue)")
    else:
        print("   No active steps are due yet")
    
    # 4. Check future active steps (shouldn't exist)
    print("\n4. Active Steps with Future Times (SHOULD NOT EXIST):")
    cur.execute("""
        SELECT contact_phone, current_step, next_trigger_time,
               EXTRACT(EPOCH FROM (next_trigger_time - NOW()))/60 as minutes_until
        FROM sequence_contacts
        WHERE status = 'active'
        AND next_trigger_time > NOW()
        ORDER BY next_trigger_time
        LIMIT 10
    """)
    future_active = cur.fetchall()
    if future_active:
        print("   PROBLEM FOUND! These active steps are scheduled for the future:")
        for phone, step, trigger_time, minutes in future_active:
            print(f"   {phone} - Step {step}: Due {trigger_time} (in {minutes:.1f} min)")
    else:
        print("   OK - No active steps scheduled for future")
    
    # 5. Sample timeline for one contact
    print("\n5. Sample Contact Timeline:")
    cur.execute("""
        SELECT contact_phone 
        FROM sequence_contacts 
        GROUP BY contact_phone 
        LIMIT 1
    """)
    sample_phone = cur.fetchone()
    if sample_phone:
        phone = sample_phone[0]
        print(f"   Contact: {phone}")
        cur.execute("""
            SELECT current_step, status, next_trigger_time, completed_at
            FROM sequence_contacts
            WHERE contact_phone = %s
            ORDER BY current_step
        """, (phone,))
        for step, status, next_time, completed in cur.fetchall():
            print(f"   Step {step}: {status}")
            print(f"      Next trigger: {next_time}")
            print(f"      Completed: {completed}")
    
    cur.close()
    conn.close()
    print("\n=== END DIAGNOSTIC REPORT ===")

if __name__ == "__main__":
    diagnose_sequence_issue()
