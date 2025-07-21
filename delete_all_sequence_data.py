#!/usr/bin/env python3
"""
Delete all sequence-related data from the database
"""

import psycopg2
import os
import sys

# Railway database connection (from the clear_sequence_contacts.py file)
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def delete_all_sequence_data():
    """Delete all sequence-related data with confirmation"""
    try:
        # Connect to database
        print("Connecting to Railway database...")
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        
        # Get counts before deletion
        print("\n=== Current Data Counts ===")
        
        # Sequence contacts
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        sequence_contacts_count = cur.fetchone()[0]
        print(f"Sequence contacts: {sequence_contacts_count}")
        
        # Broadcast messages from sequences
        cur.execute("SELECT COUNT(*) FROM broadcast_messages WHERE sequence_id IS NOT NULL")
        broadcast_messages_count = cur.fetchone()[0]
        print(f"Broadcast messages from sequences: {broadcast_messages_count}")
        
        # Sequence steps
        cur.execute("SELECT COUNT(*) FROM sequence_steps")
        sequence_steps_count = cur.fetchone()[0]
        print(f"Sequence steps: {sequence_steps_count}")
        
        # Sequences
        cur.execute("SELECT COUNT(*) FROM sequences")
        sequences_count = cur.fetchone()[0]
        print(f"Sequences: {sequences_count}")
        
        total_records = sequence_contacts_count + broadcast_messages_count + sequence_steps_count + sequences_count
        
        if total_records == 0:
            print("\nNo sequence data to delete.")
            cur.close()
            conn.close()
            return
        
        # Ask for confirmation
        print(f"\nWARNING: This will permanently delete {total_records} records!")
        print("This includes:")
        print("  - All sequence enrollments (sequence_contacts)")
        print("  - All sequence-related broadcast messages")
        print("  - All sequence steps")
        print("  - All sequences")
        
        confirm = input("\nAre you sure you want to delete ALL sequence data? Type 'yes' to confirm: ")
        
        if confirm.lower() != 'yes':
            print("Operation cancelled.")
            cur.close()
            conn.close()
            return
        
        # Delete data in the correct order (respecting foreign key constraints)
        print("\nDeleting sequence data...")
        
        # 1. Delete sequence contacts (enrollments)
        print("Deleting sequence contacts...")
        cur.execute("DELETE FROM sequence_contacts")
        deleted_contacts = cur.rowcount
        
        # 2. Delete broadcast messages from sequences
        print("Deleting sequence broadcast messages...")
        cur.execute("DELETE FROM broadcast_messages WHERE sequence_id IS NOT NULL")
        deleted_messages = cur.rowcount
        
        # 3. Delete message analytics from sequences (if table exists)
        try:
            cur.execute("DELETE FROM message_analytics WHERE sequence_id IS NOT NULL")
            deleted_analytics = cur.rowcount
            print(f"Deleted {deleted_analytics} message analytics records")
        except:
            # Table might not exist
            pass
        
        # 4. Delete sequence steps
        print("Deleting sequence steps...")
        cur.execute("DELETE FROM sequence_steps")
        deleted_steps = cur.rowcount
        
        # 5. Delete sequences
        print("Deleting sequences...")
        cur.execute("DELETE FROM sequences")
        deleted_sequences = cur.rowcount
        
        # Commit the changes
        conn.commit()
        
        print("\nDeletion Complete!")
        print(f"  - Deleted {deleted_contacts} sequence contacts")
        print(f"  - Deleted {deleted_messages} broadcast messages")
        print(f"  - Deleted {deleted_steps} sequence steps")
        print(f"  - Deleted {deleted_sequences} sequences")
        
        # Verify deletion
        print("\n=== Verification ===")
        cur.execute("SELECT COUNT(*) FROM sequence_contacts")
        print(f"Remaining sequence contacts: {cur.fetchone()[0]}")
        
        cur.execute("SELECT COUNT(*) FROM broadcast_messages WHERE sequence_id IS NOT NULL")
        print(f"Remaining sequence broadcast messages: {cur.fetchone()[0]}")
        
        cur.execute("SELECT COUNT(*) FROM sequence_steps")
        print(f"Remaining sequence steps: {cur.fetchone()[0]}")
        
        cur.execute("SELECT COUNT(*) FROM sequences")
        print(f"Remaining sequences: {cur.fetchone()[0]}")
        
        # Close connection
        cur.close()
        conn.close()
        
    except psycopg2.Error as e:
        print(f"\nDatabase Error: {e}")
        if 'conn' in locals():
            conn.rollback()
            conn.close()
    except Exception as e:
        print(f"\nError: {e}")
        if 'conn' in locals():
            conn.close()

if __name__ == "__main__":
    print("=== Delete All Sequence Data ===")
    print("Database: Railway PostgreSQL")
    
    delete_all_sequence_data()
    
    print("\nPress Enter to exit...")
    input()
