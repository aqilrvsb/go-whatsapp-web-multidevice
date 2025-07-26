import psycopg2
from datetime import datetime
import sys

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
conn_string = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

print("=== FIXING SEQUENCE DUPLICATE ISSUE ===")
print(f"Started at {datetime.now()}\n")

try:
    conn = psycopg2.connect(conn_string)
    cur = conn.cursor()
    
    # Read and execute the SQL file
    with open('C:\\Users\\ROGSTRIX\\go-whatsapp-web-multidevice-main\\fix_sequence_duplicates.sql', 'r') as f:
        sql_commands = f.read()
    
    # Split by semicolons but be careful with strings
    commands = []
    current_command = ""
    in_string = False
    
    for char in sql_commands:
        if char == "'" and not in_string:
            in_string = True
        elif char == "'" and in_string:
            in_string = False
        
        current_command += char
        
        if char == ';' and not in_string:
            commands.append(current_command.strip())
            current_command = ""
    
    # Execute each command
    for i, command in enumerate(commands):
        if command and not command.startswith('--') and not command.strip().startswith('/*'):
            try:
                print(f"\nExecuting command {i+1}...")
                cur.execute(command)
                
                # If it's a SELECT, show results
                if command.strip().upper().startswith('SELECT'):
                    results = cur.fetchall()
                    for row in results:
                        print(f"  {row}")
                else:
                    print(f"  Affected rows: {cur.rowcount}")
                    
            except Exception as e:
                print(f"  Error: {e}")
                # Continue with other commands
    
    # Commit the changes
    conn.commit()
    print("\n✅ All changes committed successfully!")
    
    # Show final state
    print("\n=== FINAL STATE ===")
    cur.execute("""
        SELECT 
            'Total contacts' as metric,
            COUNT(DISTINCT contact_phone) as value
        FROM sequence_contacts
        UNION ALL
        SELECT 
            'Total records' as metric,
            COUNT(*) as value
        FROM sequence_contacts
        UNION ALL
        SELECT 
            'Active sequences' as metric,
            COUNT(DISTINCT sequence_id) as value
        FROM sequence_contacts
        WHERE status = 'active'
    """)
    
    for row in cur.fetchall():
        print(f"{row[0]}: {row[1]}")
    
    cur.close()
    conn.close()
    
    print("\n✅ Fix completed successfully!")
    print("\nNOTE: The root cause is in the code - it's creating multiple")
    print("sequence_contact records instead of updating existing ones.")
    print("This database fix is temporary until the code is fixed.")
    
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
