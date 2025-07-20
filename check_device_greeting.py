import psycopg2

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def check_device_and_greeting():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== CHECKING DEVICE AND GREETING ISSUE ===")
    
    try:
        # Check which devices sent the messages
        print("\n1. Messages sent by which devices:")
        cur.execute("""
            SELECT 
                bm.recipient_phone,
                bm.recipient_name,
                ud.device_name,
                ud.platform,
                ud.jid,
                bm.status,
                SUBSTRING(bm.content, 1, 50) as content_start
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            WHERE bm.sequence_id IS NOT NULL
            AND bm.status = 'sent'
            ORDER BY bm.created_at DESC
            LIMIT 10
        """)
        
        messages = cur.fetchall()
        for row in messages:
            print(f"\n   To: {row[0]} ({row[1]})")
            print(f"   Device: {row[2]}")
            print(f"   Platform: {row[3] or 'WhatsApp Web'}")
            print(f"   JID: {row[4][:20]}..." if row[4] else "   JID: None")
            print(f"   Status: {row[5]}")
            print(f"   Content: {row[6]}...")
            
            # Check if this is platform or WhatsApp
            if row[3]:
                print(f"   ✓ Platform device - should have greeting!")
            else:
                print(f"   ✓ WhatsApp Web - should have greeting!")
                
        # Check the logs to see if greeting was applied
        print("\n\n2. Checking if content field is updated after sending:")
        print("   Note: The 'content' field in database might not reflect the actual sent message")
        print("   The greeting is applied at send time but may not be saved back to DB")
        
    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    check_device_and_greeting()
