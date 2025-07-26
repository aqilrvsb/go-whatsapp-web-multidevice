import psycopg2
import sys

# Try different connection strings
conn_strings = [
    # Public URL from Railway
    "postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@viaduct.proxy.rlwy.net:20298/railway",
    "postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@junction.proxy.rlwy.net:12108/railway",
    # Try with SSL
    "postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@viaduct.proxy.rlwy.net:20298/railway?sslmode=require",
]

print("Testing PostgreSQL connections...")
for i, conn_string in enumerate(conn_strings):
    print(f"\nAttempt {i+1}: {conn_string[:50]}...")
    try:
        conn = psycopg2.connect(conn_string)
        cur = conn.cursor()
        cur.execute("SELECT version();")
        version = cur.fetchone()
        print(f"SUCCESS! Connected to: {version[0][:50]}...")
        
        # Quick test query
        cur.execute("SELECT COUNT(*) FROM sequences;")
        count = cur.fetchone()
        print(f"Found {count[0]} sequences in database")
        
        cur.close()
        conn.close()
        
        print(f"\nUsing connection string: {conn_string}")
        sys.exit(0)
        
    except Exception as e:
        print(f"FAILED: {str(e)[:100]}...")
        
print("\nAll connection attempts failed!")
