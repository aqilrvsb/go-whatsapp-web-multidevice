#!/usr/bin/env python3
"""
Delete all records from sequence_contacts and broadcast_messages tables
"""

import psycopg2

# Railway database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def delete_contacts_and_messages():
    """Delete all records from sequence_contacts and broadcast_messages"""
    conn = None
    try:
        # Connect to database
        print("Connecting to Railway database...")
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        
        # Get current counts
        print("\n=== Current Record Counts ===")
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        contacts_count = cur.fetchone()[0]
        print(f"Sequence contacts: {contacts_count}")
        
        cur.execute("SELECT COUNT(*) FROM broadcast_messages")
        messages_count = cur.fetchone()[0]
        print(f"Broadcast messages: {messages_count}")
        
        total = contacts_count + messages_count
        if total == 0:
            print("\nNo records to delete.")
            return
        
        # Confirm deletion
        print(f"\nWARNING: This will delete {total} records!")
        print("  - ALL sequence contacts")
        print("  - ALL broadcast messages")
        
        confirm = input("\nType 'yes' to confirm deletion: ")
        if confirm.lower() != 'yes':
            print("Operation cancelled.")
            return
        
        # Delete records
        print("\nDeleting records...")
        
        # Delete sequence contacts
        cur.execute("DELETE FROM sequence_contacts")
        deleted_contacts = cur.rowcount
        print(f"Deleted {deleted_contacts} sequence contacts")
        
        # Delete broadcast messages
        cur.execute("DELETE FROM broadcast_messages")
        deleted_messages = cur.rowcount
        print(f"Deleted {deleted_messages} broadcast messages")
        
        # Commit changes
        conn.commit()
        
        # Verify deletion
        print("\n=== Verification ===")
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        print(f"Sequence contacts remaining: {cur.fetchone()[0]}")
        
        cur.execute("SELECT COUNT(*) FROM broadcast_messages")
        print(f"Broadcast messages remaining: {cur.fetchone()[0]}")
        
        print("\nDeletion completed successfully!")
        
        # Close connection
        cur.close()
        conn.close()
        
    except Exception as e:
        print(f"\nError: {e}")
        if conn:
            conn.rollback()
            conn.close()

if __name__ == "__main__":
    print("=== Delete Sequence Contacts and Broadcast Messages ===")
    delete_contacts_and_messages()
    
    print("\nPress Enter to exit...")
    input()
