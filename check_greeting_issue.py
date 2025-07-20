import psycopg2
from datetime import datetime

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def check_greeting_issue():
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== CHECKING GREETING NAME ISSUE ===")
    
    try:
        # Check sent messages with actual content
        print("\n1. Checking actual sent messages:")
        cur.execute("""
            SELECT 
                recipient_phone, 
                recipient_name,
                SUBSTRING(content, 1, 150) as content_preview,
                status
            FROM broadcast_messages
            WHERE sequence_id IS NOT NULL
            AND status = 'sent'
            AND content IS NOT NULL
            ORDER BY created_at DESC
            LIMIT 5
        """)
        messages = cur.fetchall()
        
        if not messages:
            print("   No sent messages found!")
        else:
            for phone, name, content, status in messages:
                print(f"\n   To: {phone}")
                print(f"   Recipient Name: '{name}'")
                print(f"   Content Preview: '{content}'")
                print(f"   Status: {status}")
                
                # Check if greeting is working
                if content:
                    lines = content.split('\n')
                    if lines:
                        print(f"   First Line (Greeting): '{lines[0]}'")
                        if 'name' in lines[0].lower():
                            print("   ⚠️  WARNING: Greeting contains literal 'name' - template not processed!")
        
        # Check if the greeting processor template is being replaced
        print("\n2. Analyzing greeting patterns:")
        cur.execute("""
            SELECT 
                recipient_name,
                SUBSTRING(content, 1, 30) as greeting_part
            FROM broadcast_messages
            WHERE sequence_id IS NOT NULL
            AND content IS NOT NULL
            ORDER BY created_at DESC
            LIMIT 10
        """)
        patterns = cur.fetchall()
        
        print("\n   Name -> Greeting mapping:")
        for name, greeting in patterns:
            print(f"   '{name}' -> '{greeting}'")
            
            # Check for issues
            if '{name}' in greeting:
                print("      ❌ ERROR: Template {name} not replaced!")
            elif 'name' in greeting.lower() and name.lower() not in greeting.lower():
                print("      ⚠️  WARNING: 'name' appears but recipient name not found")
                
    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    check_greeting_issue()
