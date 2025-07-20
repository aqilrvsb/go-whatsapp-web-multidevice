import psycopg2
from datetime import datetime

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def debug_greeting_flow():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== DEBUGGING GREETING FLOW ===")
    
    try:
        # Check the raw broadcast_messages data
        print("\n1. Raw broadcast_messages data:")
        cur.execute("""
            SELECT 
                id,
                recipient_phone,
                recipient_name,
                SUBSTRING(message, 1, 100) as message_field,
                SUBSTRING(content, 1, 100) as content_field,
                status
            FROM broadcast_messages
            WHERE sequence_id IS NOT NULL
            ORDER BY created_at DESC
            LIMIT 5
        """)
        messages = cur.fetchall()
        
        for row in messages:
            print(f"\n   ID: {row[0]}")
            print(f"   Phone: {row[1]}")
            print(f"   Recipient Name: '{row[2]}'")
            print(f"   Message field: '{row[3]}...'")
            print(f"   Content field: '{row[4]}...'")
            print(f"   Status: {row[5]}")
            
        # Check if the message field has greeting but content doesn't
        print("\n2. Comparing message vs content fields:")
        cur.execute("""
            SELECT 
                recipient_name,
                CASE 
                    WHEN message LIKE '%Hi %' OR message LIKE '%Hello%' OR message LIKE '%Hai%' 
                    THEN 'HAS GREETING'
                    ELSE 'NO GREETING'
                END as message_greeting,
                CASE 
                    WHEN content LIKE '%Hi %' OR content LIKE '%Hello%' OR content LIKE '%Hai%' 
                    THEN 'HAS GREETING'
                    ELSE 'NO GREETING'
                END as content_greeting,
                SUBSTRING(message, 1, 50) as msg_preview,
                SUBSTRING(content, 1, 50) as content_preview
            FROM broadcast_messages
            WHERE sequence_id IS NOT NULL
            AND status = 'sent'
            ORDER BY created_at DESC
            LIMIT 5
        """)
        
        comparisons = cur.fetchall()
        for row in comparisons:
            print(f"\n   Name: '{row[0]}'")
            print(f"   Message field: {row[1]} - '{row[3]}...'")
            print(f"   Content field: {row[2]} - '{row[4]}...'")
            
            if row[1] != row[2]:
                print("   ⚠️  MISMATCH: Message and content fields differ!")
                
    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    debug_greeting_flow()
