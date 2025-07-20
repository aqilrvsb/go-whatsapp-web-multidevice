import psycopg2

# Database connection
DB_URI = "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"

def update_lead_name():
    phone = input("Enter the phone number of the lead to update (e.g., 60123456789): ").strip()
    name = input("Enter the customer's actual name: ").strip()
    
    if not phone or not name:
        print("Phone and name cannot be empty!")
        return
    
    try:
        conn = psycopg2.connect(DB_URI)
        cur = conn.cursor()
        
        # Update the lead
        cur.execute("""
            UPDATE leads 
            SET name = %s 
            WHERE phone = %s
        """, (name, phone))
        
        rows_updated = cur.rowcount
        
        if rows_updated > 0:
            conn.commit()
            print(f"✅ Successfully updated {rows_updated} lead(s) with name '{name}'")
            
            # Also update any existing sequence contacts
            cur.execute("""
                UPDATE sequence_contacts 
                SET contact_name = %s 
                WHERE contact_phone = %s
            """, (name, phone))
            
            seq_rows = cur.rowcount
            if seq_rows > 0:
                conn.commit()
                print(f"✅ Also updated {seq_rows} sequence contact(s)")
        else:
            print(f"❌ No lead found with phone number: {phone}")
        
        cur.close()
        conn.close()
        
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    print("=== UPDATE LEAD NAME ===")
    print("This will update the lead's name so greetings show the actual name instead of 'Cik'\n")
    
    while True:
        update_lead_name()
        
        another = input("\nUpdate another lead? (y/n): ").lower()
        if another != 'y':
            break
    
    print("\nDone! The next messages will use the proper names.")
