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
    
    print("=== SYSTEM STATE ANALYSIS - CAMPAIGNS & SEQUENCES ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Check campaigns table structure
    print("1. CAMPAIGNS TABLE:")
    cur.execute("""
        SELECT column_name, data_type 
        FROM information_schema.columns 
        WHERE table_name = 'campaigns'
        LIMIT 10
    """)
    
    cols = cur.fetchall()
    if cols:
        print("Columns found:", [c[0] for c in cols])
    else:
        print("No campaigns table found")
    
    # 2. Recent campaigns
    cur.execute("""
        SELECT 
            c.id,
            c.status,
            COUNT(DISTINCT bm.recipient_phone) as recipients
        FROM campaigns c
        LEFT JOIN broadcast_messages bm ON bm.campaign_id = c.id
        GROUP BY c.id, c.status
        ORDER BY c.id DESC
        LIMIT 5
    """)
    
    campaigns = cur.fetchall()
    if campaigns:
        print("\nRecent Campaigns:")
        for c in campaigns:
            print(f"  Campaign {c[0]}: {c[2]} recipients, status: {c[1]}")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Sequence Flow Status
    print("2. SEQUENCE PROCESSING STATUS:")
    
    # Messages by step
    cur.execute("""
        SELECT 
            ss.day_number,
            COUNT(DISTINCT sc.contact_phone) as contacts,
            SUM(CASE WHEN sc.status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN sc.status = 'sent' THEN 1 ELSE 0 END) as sent,
            SUM(CASE WHEN sc.status = 'completed' THEN 1 ELSE 0 END) as completed
        FROM sequence_contacts sc
        JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
        GROUP BY ss.day_number
        ORDER BY ss.day_number
    """)
    
    steps = cur.fetchall()
    print("\nContacts by Step:")
    print("Step | Contacts | Pending | Sent | Completed")
    print("-" * 50)
    for step in steps:
        print(f"{step[0]:4} | {step[1]:8} | {step[2]:7} | {step[3]:4} | {step[4]:9}")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Broadcast Messages Flow
    print("3. BROADCAST MESSAGES FLOW:")
    
    # Last hour activity
    cur.execute("""
        SELECT 
            DATE_TRUNC('minute', created_at) as minute,
            COUNT(*) as created,
            SUM(CASE WHEN campaign_id IS NOT NULL THEN 1 ELSE 0 END) as campaign_msgs,
            SUM(CASE WHEN sequence_id IS NOT NULL THEN 1 ELSE 0 END) as sequence_msgs
        FROM broadcast_messages
        WHERE created_at > NOW() - INTERVAL '1 hour'
        GROUP BY DATE_TRUNC('minute', created_at)
        ORDER BY minute DESC
        LIMIT 10
    """)
    
    activity = cur.fetchall()
    if activity:
        print("Recent Activity (by minute):")
        print("Time  | Total | Campaign | Sequence")
        print("-" * 40)
        for a in activity:
            print(f"{a[0].strftime('%H:%M')} | {a[1]:5} | {a[2]:8} | {a[3]:8}")
    else:
        print("No activity in last hour")
    
    # Current queue
    cur.execute("""
        SELECT 
            status,
            COUNT(*) as count,
            MIN(scheduled_at) as oldest
        FROM broadcast_messages
        WHERE created_at > NOW() - INTERVAL '24 hours'
        GROUP BY status
    """)
    
    print("\nCurrent Queue Status:")
    queue = cur.fetchall()
    for q in queue:
        print(f"  {q[0]}: {q[1]} messages, oldest: {q[2]}")
    
    print("\n" + "="*80 + "\n")
    
    # 5. Device and Processing
    print("4. DEVICE PROCESSING:")
    
    cur.execute("""
        SELECT 
            ud.status,
            COUNT(DISTINCT ud.id) as device_count,
            COUNT(bm.id) as pending_messages
        FROM user_devices ud
        LEFT JOIN broadcast_messages bm ON bm.device_id = ud.id AND bm.status = 'pending'
        WHERE ud.platform IS NULL
        GROUP BY ud.status
    """)
    
    device_stats = cur.fetchall()
    for ds in device_stats:
        print(f"  {ds[0]} devices: {ds[1]}, pending messages: {ds[2]}")
    
    print("\n" + "="*80 + "\n")
    
    # 6. Troubleshooting
    print("5. TROUBLESHOOTING:")
    
    # Check if sequence processor is creating messages
    cur.execute("""
        SELECT 
            COUNT(*) as ready_contacts
        FROM sequence_contacts sc
        WHERE sc.status = 'pending'
        AND sc.next_trigger_time <= NOW()
        AND NOT EXISTS (
            SELECT 1 FROM broadcast_messages bm
            WHERE bm.recipient_phone = sc.contact_phone
            AND bm.sequence_stepid = sc.sequence_stepid
        )
    """)
    
    ready = cur.fetchone()[0]
    print(f"  Sequence contacts ready but no message: {ready}")
    
    # Check stuck processing
    cur.execute("""
        SELECT COUNT(*) 
        FROM sequence_contacts
        WHERE processing_device_id IS NOT NULL
    """)
    
    stuck = cur.fetchone()[0]
    print(f"  Contacts stuck in processing: {stuck}")
    
    # Check last message sent
    cur.execute("""
        SELECT MAX(sent_at) as last_sent,
               NOW() - MAX(sent_at) as time_ago
        FROM broadcast_messages
        WHERE status = 'sent'
    """)
    
    last_sent = cur.fetchone()
    if last_sent[0]:
        print(f"  Last message sent: {last_sent[0]} ({last_sent[1]} ago)")
    else:
        print("  No messages sent yet")
    
    print("\n" + "="*80 + "\n")
    
    print("📋 FLOW SUMMARY:")
    print("1. CAMPAIGNS: Trigger → Create all messages in broadcast_messages → Redis → Send")
    print("2. SEQUENCES: Enrollment → Create contact records → Process triggers → Create messages → Redis → Send")
    print("3. REDIS QUEUES: 'queue:campaign_X' and 'queue:seq_UUID'")
    print("4. WORKER POOLS: Auto-created per campaign/sequence, cleaned after 5min idle")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
