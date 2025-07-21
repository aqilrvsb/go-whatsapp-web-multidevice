#!/usr/bin/env python3
"""
Run SQL file against the database
"""

import psycopg2
import sys

# Railway database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def run_sql_file(filename):
    """Execute SQL file"""
    conn = None
    try:
        # Read SQL file
        print(f"Reading SQL file: {filename}")
        with open(filename, 'r') as f:
            sql_content = f.read()
        
        # Connect to database
        print("Connecting to Railway database...")
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        
        # Execute SQL
        print("Executing SQL...")
        cur.execute(sql_content)
        
        # If there are results (like the SELECT statements at the end), show them
        try:
            results = cur.fetchall()
            if results:
                print("\n=== Results ===")
                for row in results:
                    print(f"{row[0]}: {row[1]}")
        except psycopg2.ProgrammingError:
            # No results to fetch (DELETE/UPDATE statements)
            pass
        
        # Commit changes
        conn.commit()
        print("\nSQL execution completed successfully!")
        
        # Close connection
        cur.close()
        conn.close()
        
    except Exception as e:
        print(f"\nError: {e}")
        if conn:
            conn.rollback()
            conn.close()

if __name__ == "__main__":
    sql_file = "delete_all_sequence_data.sql"
    
    print("=== SQL File Executor ===")
    print(f"File: {sql_file}")
    print("Database: Railway PostgreSQL")
    print("\nWARNING: This will delete all sequence data!")
    
    confirm = input("\nContinue? Type 'yes' to confirm: ")
    if confirm.lower() == 'yes':
        run_sql_file(sql_file)
    else:
        print("Operation cancelled.")
    
    print("\nPress Enter to exit...")
    input()
