import psycopg2
import time
import sys

# Database connection string
conn_string = "postgresql://whatsapp_user:Cahaya123!@autorack.proxy.rlwy.net:24400/railway"

def check_database():
    max_retries = 3
    retry_count = 0
    
    while retry_count < max_retries:
        try:
            print(f"Connecting to PostgreSQL (attempt {retry_count + 1})...")
            conn = psycopg2.connect(conn_string, connect_timeout=10)
            cur = conn.cursor()
            
            print("Connected successfully!\n")
            
            # First, let's see ALL columns in broadcast_messages
            print("=== ALL COLUMNS IN broadcast_messages ===")
            cur.execute("""
                SELECT column_name, data_type, is_nullable
                FROM information_schema.columns 
                WHERE table_name = 'broadcast_messages'
                ORDER BY ordinal_position
            """)
            
            all_columns = cur.fetchall()
            for col in all_columns:
                print(f"  {col[0]:<25} {col[1]:<20} {'NULL' if col[2] == 'YES' else 'NOT NULL'}")
            
            # Check specific columns
            print("\n=== CHECKING FOR sequence_stepid ===")
            has_stepid = any(col[0] == 'sequence_stepid' for col in all_columns)
            print(f"Has sequence_stepid: {has_stepid}")
            
            if not has_stepid:
                print("\n⚠️ MISSING sequence_stepid - Adding it...")
                try:
                    cur.execute("""
                        ALTER TABLE broadcast_messages 
                        ADD COLUMN sequence_stepid UUID REFERENCES sequence_steps(id) ON DELETE SET NULL
                    """)
                    conn.commit()
                    print("✅ Added sequence_stepid column!")
                except Exception as e:
                    print(f"Error adding column: {e}")
                    conn.rollback()
            
            # Check sequence_steps structure
            print("\n=== SEQUENCE_STEPS DELAY COLUMNS ===")
            cur.execute("""
                SELECT column_name, data_type
                FROM information_schema.columns 
                WHERE table_name = 'sequence_steps'
                AND column_name LIKE '%delay%'
                ORDER BY column_name
            """)
            
            for col in cur.fetchall():
                print(f"  {col[0]}: {col[1]}")
            
            # Show how broadcast messages are currently structured
            print("\n=== SAMPLE BROADCAST MESSAGES ===")
            cur.execute("""
                SELECT 
                    id,
                    CASE 
                        WHEN campaign_id IS NOT NULL THEN 'Campaign'
                        WHEN sequence_id IS NOT NULL THEN 'Sequence'
                        ELSE 'Unknown'
                    END as type,
                    campaign_id,
                    sequence_id,
                    sequence_stepid
                FROM broadcast_messages
                WHERE sequence_id IS NOT NULL
                LIMIT 5
            """)
            
            rows = cur.fetchall()
            if rows:
                for row in rows:
                    print(f"  ID: {row[0][:8]}... Type: {row[1]}, Campaign: {row[2]}, Sequence: {row[3]}, StepID: {row[4]}")
            else:
                print("  No sequence messages found")
            
            cur.close()
            conn.close()
            print("\n✅ Database check complete!")
            return
            
        except psycopg2.OperationalError as e:
            retry_count += 1
            print(f"Connection failed: {e}")
            if retry_count < max_retries:
                print(f"Retrying in 3 seconds...")
                time.sleep(3)
            else:
                print("Max retries reached. Connection failed.")
                sys.exit(1)
        except Exception as e:
            print(f"Unexpected error: {e}")
            sys.exit(1)

if __name__ == "__main__":
    check_database()
