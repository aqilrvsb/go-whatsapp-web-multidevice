import psycopg2
from datetime import datetime
import sys

# Set UTF-8 encoding for output
sys.stdout.reconfigure(encoding='utf-8')

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

try:
    conn = psycopg2.connect(DB_URI)
    cur = conn.cursor()
    
    print("=== CHECKING LEAD NAMES IN DATABASE ===\n")
    
    # Check some leads
    query = """
    SELECT phone, name, device_id, niche, trigger
    FROM leads
    WHERE trigger IS NOT NULL AND trigger != ''
    ORDER BY created_at DESC
    LIMIT 10
    """
    
    cur.execute(query)
    leads = cur.fetchall()
    
    print(f"Found {len(leads)} recent leads with triggers:\n")
    
    for lead in leads:
        phone, name, device_id, niche, trigger = lead
        print(f"Phone: {phone}")
        print(f"Name: '{name}' {'<-- THIS IS THE ISSUE!' if not name or name.strip() == '' or name == phone else ''}")
        print(f"Niche: {niche}")
        print(f"Trigger: {trigger}")
        print("-" * 50)
    
    # Check sequence contacts
    print("\n=== CHECKING SEQUENCE CONTACTS ===\n")
    
    query2 = """
    SELECT contact_phone, contact_name, status, current_step
    FROM sequence_contacts
    ORDER BY created_at DESC
    LIMIT 10
    """
    
    cur.execute(query2)
    contacts = cur.fetchall()
    
    if contacts:
        print(f"Found {len(contacts)} sequence contacts:\n")
        for contact in contacts:
            phone, name, status, step = contact
            print(f"Phone: {phone}")
            print(f"Name: '{name}' {'<-- EMPTY/PHONE!' if not name or name.strip() == '' or name == phone else ''}")
            print(f"Status: {status}, Step: {step}")
            print("-" * 30)
    else:
        print("No sequence contacts found (table is empty)")
    
    # Check if names are phone numbers
    print("\n=== ANALYSIS ===")
    
    cur.execute("SELECT COUNT(*) FROM leads WHERE name IS NULL OR name = ''")
    empty_count = cur.fetchone()[0]
    
    cur.execute("SELECT COUNT(*) FROM leads WHERE name = phone")
    phone_as_name_count = cur.fetchone()[0]
    
    cur.execute("SELECT COUNT(*) FROM leads")
    total_count = cur.fetchone()[0]
    
    print(f"\nTotal leads: {total_count}")
    print(f"Leads with empty name: {empty_count}")
    print(f"Leads with phone as name: {phone_as_name_count}")
    print(f"Leads with proper names: {total_count - empty_count - phone_as_name_count}")
    
    if empty_count > 0 or phone_as_name_count > 0:
        print("\n⚠️  THIS IS WHY YOU'RE SEEING 'Cik' - The leads don't have proper names!")
        print("\nTo fix this, you need to update the leads with actual names:")
        print("UPDATE leads SET name = 'Customer Name' WHERE phone = '60123456789';")
    
    cur.close()
    conn.close()
    
except Exception as e:
    print(f"Error: {e}")
