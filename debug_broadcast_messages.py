import psycopg2

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def check_broadcast_messages():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== CHECKING BROADCAST MESSAGES ===")
    
    try:
        # Check pending messages
        cur.execute("""
            SELECT id, recipient_phone, recipient_name, content, status, created_at
            FROM broadcast_messages 
            WHERE status = 'pending'
            ORDER BY created_at DESC
            LIMIT 10
        """)
        
        rows = cur.fetchall()
        
        if rows:
            print(f"\nFound {len(rows)} pending messages:")
            for row in rows:
                print(f"\n- ID: {row[0][:8]}...")
                print(f"  Phone: {row[1]}")
                print(f"  Name: '{row[2]}'")
                print(f"  Content preview: {row[3][:100]}...")
                print(f"  Status: {row[4]}")
                print(f"  Created: {row[5]}")
        else:
            print("\nNo pending messages found.")
            
        # Check recently sent messages
        print("\n\n=== RECENTLY SENT MESSAGES ===")
        cur.execute("""
            SELECT id, recipient_phone, recipient_name, content, status, sent_at
            FROM broadcast_messages 
            WHERE status = 'sent'
            ORDER BY sent_at DESC
            LIMIT 5
        """)
        
        rows = cur.fetchall()
        
        if rows:
            print(f"\nFound {len(rows)} recent sent messages:")
            for row in rows:
                print(f"\n- ID: {row[0][:8]}...")
                print(f"  Phone: {row[1]}")
                print(f"  Name: '{row[2]}'")
                print(f"  Content preview: {row[3][:100]}...")
                print(f"  Sent at: {row[5]}")
        
    except Exception as e:
        print(f"Error: {e}")
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    check_broadcast_messages()
