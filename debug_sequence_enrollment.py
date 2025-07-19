import psycopg2

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    # Connect to database
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("Connected to PostgreSQL database...")
    print("=" * 60)
    
    # 1. Check the sequence enrollment query - what leads should be enrolled
    print("\n1. Checking leads that should be enrolled in sequences...")
    cur.execute("""
        SELECT DISTINCT 
            l.id, l.phone, l.name, l.device_id, l.user_id, 
            s.id as sequence_id, s.name as sequence_name,
            ss.trigger as entry_trigger, ss.id as step_id
        FROM leads l
        CROSS JOIN sequences s
        INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
        WHERE s.is_active = true 
            AND ss.is_entry_point = true
            AND l.trigger IS NOT NULL 
            AND l.trigger != ''
            AND position(ss.trigger in l.trigger) > 0
            AND NOT EXISTS (
                SELECT 1 FROM sequence_contacts sc
                WHERE sc.sequence_id = s.id 
                AND sc.contact_phone = l.phone
                AND sc.current_step = 1
            )
        LIMIT 10
    """)
    
    enrollable_leads = cur.fetchall()
    print(f"Found {len(enrollable_leads)} leads ready for enrollment:")
    for lead in enrollable_leads:
        print(f"  Phone: {lead[1]}, Name: {lead[2]}, Trigger: {lead[6]}")
        print(f"  Sequence: {lead[6]}, Entry Step: {lead[8]}")
    
    # 2. Check sequence steps for WARM Sequence
    print("\n2. Checking steps for WARM Sequence...")
    cur.execute("""
        SELECT id, day_number, trigger, next_trigger, trigger_delay_hours, is_entry_point
        FROM sequence_steps
        WHERE sequence_id = (SELECT id FROM sequences WHERE name = 'WARM Sequence')
        ORDER BY day_number
    """)
    
    steps = cur.fetchall()
    print(f"Found {len(steps)} steps:")
    for step in steps:
        print(f"  Day {step[1]}: {step[2]} -> {step[3]} (delay: {step[4]}h, entry: {step[5]})")
        print(f"    Step ID: {step[0]}")
    
    # 3. Check existing enrollments
    print("\n3. Checking existing sequence_contacts...")
    cur.execute("""
        SELECT sequence_id, contact_phone, sequence_stepid, status, current_step
        FROM sequence_contacts
        WHERE contact_phone IN (
            SELECT phone FROM leads WHERE trigger = 'WARMEXAMA' LIMIT 5
        )
    """)
    
    existing = cur.fetchall()
    if existing:
        print(f"Found {len(existing)} existing enrollments:")
        for e in existing:
            print(f"  Phone: {e[1]}, Step: {e[4]}, Status: {e[3]}")
    else:
        print("No existing enrollments found for WARMEXAMA leads")
    
    # 4. Try manual enrollment to test
    print("\n4. Testing manual enrollment...")
    
    # Get first WARMEXAMA lead and sequence info
    cur.execute("""
        SELECT l.phone, l.name, l.device_id, l.user_id,
               s.id as seq_id, ss.id as step_id
        FROM leads l, sequences s, sequence_steps ss
        WHERE l.trigger = 'WARMEXAMA'
        AND s.name = 'WARM Sequence'
        AND ss.sequence_id = s.id
        AND ss.day_number = 1
        LIMIT 1
    """)
    
    test_data = cur.fetchone()
    if test_data:
        phone, name, device_id, user_id, seq_id, step_id = test_data
        
        print(f"Testing with phone: {phone}")
        
        # Try the insert
        try:
            cur.execute("""
                INSERT INTO sequence_contacts (
                    sequence_id, contact_phone, contact_name, 
                    current_step, status, completed_at, current_trigger,
                    next_trigger_time, sequence_stepid, assigned_device_id,
                    user_id
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (sequence_id, contact_phone, sequence_stepid) DO NOTHING
                RETURNING id
            """, (
                seq_id,           # sequence_id
                phone + '_test',  # contact_phone (adding _test to avoid real conflict)
                name,             # contact_name
                1,                # current_step
                'active',         # status
                'NOW()',          # completed_at
                'WARMEXAMA',      # current_trigger
                'NOW()',          # next_trigger_time
                step_id,          # sequence_stepid
                device_id,        # assigned_device_id
                user_id           # user_id
            ))
            
            result = cur.fetchone()
            if result:
                print(f"SUCCESS: Inserted test enrollment with ID: {result[0]}")
                # Clean up
                cur.execute("DELETE FROM sequence_contacts WHERE id = %s", (result[0],))
            else:
                print("Insert was skipped due to ON CONFLICT")
                
        except Exception as e:
            print(f"ERROR: {e}")
    
    # 5. Check constraints one more time
    print("\n5. Final constraint check...")
    cur.execute("""
        SELECT conname, pg_get_constraintdef(oid) as definition
        FROM pg_constraint 
        WHERE conrelid = 'sequence_contacts'::regclass
        AND contype = 'u'
    """)
    
    constraints = cur.fetchall()
    for name, definition in constraints:
        print(f"  {name}: {definition}")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
    if conn:
        conn.close()
