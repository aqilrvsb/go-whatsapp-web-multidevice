import psycopg2
import sys
from datetime import datetime
import json

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    print("=== DEEP SYSTEM MONITORING - CAMPAIGNS & SEQUENCES ===")
    print(f"Time: {datetime.now()}\n")
    
    # 1. Campaign Status
    print("1. CAMPAIGN STATUS:")
    cur.execute("""
        SELECT 
            c.id,
            c.name,
            c.status,
            COUNT(DISTINCT bm.recipient_phone) as total_recipients,
            SUM(CASE WHEN bm.status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN bm.status = 'sent' THEN 1 ELSE 0 END) as sent,
            SUM(CASE WHEN bm.status = 'failed' THEN 1 ELSE 0 END) as failed,
            c.min_delay_seconds,
            c.max_delay_seconds
        FROM campaigns c
        LEFT JOIN broadcast_messages bm ON bm.campaign_id = c.id
        WHERE c.created_at > NOW() - INTERVAL '7 days'
        GROUP BY c.id, c.name, c.status, c.min_delay_seconds, c.max_delay_seconds
        ORDER BY c.created_at DESC
        LIMIT 5
    """)
    
    campaigns = cur.fetchall()
    if campaigns:
        print("ID | Name | Status | Recipients | Pending | Sent | Failed | Delays")
        print("-" * 80)
        for c in campaigns:
            print(f"{c[0]} | {c[1][:20]} | {c[2]} | {c[3]} | {c[4]} | {c[5]} | {c[6]} | {c[7]}-{c[8]}s")
    else:
        print("No recent campaigns found")
    
    print("\n" + "="*80 + "\n")
    
    # 2. Sequence Flow Analysis
    print("2. SEQUENCE FLOW ANALYSIS:")
    
    # Active sequences
    cur.execute("""
        SELECT 
            s.name,
            s.trigger,
            COUNT(DISTINCT ss.id) as total_steps,
            COUNT(DISTINCT sc.contact_phone) as enrolled_contacts,
            STRING_AGG(DISTINCT ss.next_trigger, ', ') as linked_sequences
        FROM sequences s
        LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
        LEFT JOIN sequence_contacts sc ON sc.sequence_id = s.id
        WHERE s.is_active = true
        GROUP BY s.id, s.name, s.trigger
    """)
    
    sequences = cur.fetchall()
    print("\nActive Sequences:")
    for seq in sequences:
        print(f"\n  {seq[0]}:")
        print(f"    Trigger: {seq[1]}")
        print(f"    Steps: {seq[2]}")
        print(f"    Enrolled: {seq[3]} contacts")
        if seq[4]:
            print(f"    Links to: {seq[4]}")
    
    # Sequence contact states
    cur.execute("""
        SELECT 
            status,
            COUNT(*) as count,
            MIN(next_trigger_time) as next_trigger
        FROM sequence_contacts
        GROUP BY status
        ORDER BY status
    """)
    
    print("\nSequence Contact States:")
    states = cur.fetchall()
    for state in states:
        print(f"  {state[0]}: {state[1]} contacts, next trigger: {state[2]}")
    
    print("\n" + "="*80 + "\n")
    
    # 3. Broadcast Messages Queue Status
    print("3. BROADCAST MESSAGES QUEUE:")
    cur.execute("""
        WITH queue_stats AS (
            SELECT 
                CASE 
                    WHEN campaign_id IS NOT NULL THEN 'Campaign'
                    WHEN sequence_id IS NOT NULL THEN 'Sequence'
                    ELSE 'Other'
                END as message_type,
                status,
                COUNT(*) as count,
                MIN(scheduled_at) as oldest_scheduled,
                MAX(scheduled_at) as newest_scheduled
            FROM broadcast_messages
            WHERE created_at > NOW() - INTERVAL '24 hours'
            GROUP BY message_type, status
        )
        SELECT * FROM queue_stats
        ORDER BY message_type, status
    """)
    
    queue_stats = cur.fetchall()
    print("Type | Status | Count | Oldest | Newest")
    print("-" * 60)
    for stat in queue_stats:
        print(f"{stat[0]} | {stat[1]} | {stat[2]} | {stat[3]} | {stat[4]}")
    
    print("\n" + "="*80 + "\n")
    
    # 4. Device Status and Load
    print("4. DEVICE STATUS AND LOAD:")
    cur.execute("""
        SELECT 
            ud.id,
            ud.phone,
            ud.status,
            COUNT(bm.id) as pending_messages,
            (SELECT COUNT(*) FROM broadcast_messages 
             WHERE device_id = ud.id 
             AND status = 'sent' 
             AND sent_at > NOW() - INTERVAL '1 hour') as sent_last_hour
        FROM user_devices ud
        LEFT JOIN broadcast_messages bm ON bm.device_id = ud.id AND bm.status = 'pending'
        WHERE ud.platform IS NULL
        GROUP BY ud.id, ud.phone, ud.status
        ORDER BY pending_messages DESC
        LIMIT 10
    """)
    
    devices = cur.fetchall()
    print("Device ID | Phone | Status | Pending | Sent/Hour")
    print("-" * 60)
    for dev in devices:
        print(f"{dev[0][:8]}... | {dev[1]} | {dev[2]} | {dev[3]} | {dev[4]}")
    
    print("\n" + "="*80 + "\n")
    
    # 5. Processing Flow Check
    print("5. PROCESSING FLOW CHECK:")
    
    # Check if processors are working
    cur.execute("""
        SELECT 
            'Sequence Messages Created (last hour)' as metric,
            COUNT(*) as value
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
        AND created_at > NOW() - INTERVAL '1 hour'
        UNION ALL
        SELECT 
            'Campaign Messages Created (last hour)' as metric,
            COUNT(*) as value
        FROM broadcast_messages
        WHERE campaign_id IS NOT NULL
        AND created_at > NOW() - INTERVAL '1 hour'
        UNION ALL
        SELECT 
            'Messages Sent (last hour)' as metric,
            COUNT(*) as value
        FROM broadcast_messages
        WHERE status = 'sent'
        AND sent_at > NOW() - INTERVAL '1 hour'
    """)
    
    metrics = cur.fetchall()
    for metric in metrics:
        print(f"  {metric[0]}: {metric[1]}")
    
    # Check for stuck processing
    cur.execute("""
        SELECT COUNT(*) 
        FROM sequence_contacts
        WHERE processing_device_id IS NOT NULL
        AND processing_started_at < NOW() - INTERVAL '10 minutes'
    """)
    
    stuck = cur.fetchone()[0]
    print(f"\n  Stuck sequence contacts: {stuck}")
    
    print("\n" + "="*80 + "\n")
    
    # 6. Redis Queue Simulation (what should be in Redis)
    print("6. WORKER POOL SIMULATION (What should be in Redis):")
    
    # Campaign pools
    cur.execute("""
        SELECT 
            'campaign_' || campaign_id as pool_id,
            COUNT(*) as queued_messages
        FROM broadcast_messages
        WHERE campaign_id IS NOT NULL
        AND status = 'pending'
        GROUP BY campaign_id
    """)
    
    campaign_pools = cur.fetchall()
    if campaign_pools:
        print("\nCampaign Pools:")
        for pool in campaign_pools:
            print(f"  Pool: {pool[0]}, Queued: {pool[1]}")
    
    # Sequence pools
    cur.execute("""
        SELECT 
            'seq_' || sequence_id as pool_id,
            COUNT(*) as queued_messages
        FROM broadcast_messages
        WHERE sequence_id IS NOT NULL
        AND status = 'pending'
        GROUP BY sequence_id
    """)
    
    seq_pools = cur.fetchall()
    if seq_pools:
        print("\nSequence Pools:")
        for pool in seq_pools:
            print(f"  Pool: {pool[0]}, Queued: {pool[1]}")
    
    print("\n" + "="*80 + "\n")
    
    print("💡 KEY INSIGHTS:")
    print("1. Campaigns create bulk messages → broadcast_messages → Redis queues → WhatsApp")
    print("2. Sequences create messages one-by-one as triggers activate")
    print("3. Both use same broadcast_messages table and Redis worker pools")
    print("4. Pool ID format: 'campaign_123' or 'seq_uuid'")
    print("5. Redis queues are: queue:campaign_123, queue:seq_uuid")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
