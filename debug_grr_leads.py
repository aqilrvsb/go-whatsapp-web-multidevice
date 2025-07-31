import pymysql
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Parse MySQL URI
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')
uri_parts = mysql_uri.replace('mysql://', '').split('@')
user_pass = uri_parts[0].split(':')
host_db = uri_parts[1].split('/')
host_port = host_db[0].split(':')

# Connect to MySQL
connection = pymysql.connect(
    host=host_port[0],
    port=int(host_port[1]),
    user=user_pass[0],
    password=user_pass[1],
    database=host_db[1],
    charset='utf8mb4'
)

try:
    with connection.cursor() as cursor:
        # Check GRR campaigns
        print("=== GRR CAMPAIGNS ===")
        cursor.execute("""
            SELECT id, title, niche, target_status, user_id
            FROM campaigns
            WHERE niche = 'GRR'
            ORDER BY created_at DESC
        """)
        
        campaigns = cursor.fetchall()
        for camp in campaigns:
            print(f"\nCampaign {camp[0]}: {camp[1]}")
            print(f"  Niche: {camp[2]}")
            print(f"  Target Status: {camp[3]}")
            print(f"  User ID: {camp[4]}")
            
            # Check leads for this campaign
            cursor.execute("""
                SELECT COUNT(DISTINCT phone) 
                FROM leads
                WHERE user_id = %s 
                AND niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR target_status = %s)
            """, (camp[4], camp[2], camp[3], camp[3]))
            
            lead_count = cursor.fetchone()[0]
            print(f"  Matching Leads: {lead_count}")
            
            # Show some sample leads
            cursor.execute("""
                SELECT phone, name, niche, target_status
                FROM leads
                WHERE user_id = %s 
                AND niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR target_status = %s)
                LIMIT 5
            """, (camp[4], camp[2], camp[3], camp[3]))
            
            sample_leads = cursor.fetchall()
            if sample_leads:
                print("  Sample matching leads:")
                for lead in sample_leads:
                    print(f"    - {lead[1]} ({lead[0]}), Niche: {lead[2]}, Target: {lead[3]}")
        
        # Check all leads with GRR niche
        print("\n=== ALL GRR LEADS ===")
        cursor.execute("""
            SELECT user_id, COUNT(*) as count, GROUP_CONCAT(DISTINCT target_status) as statuses
            FROM leads
            WHERE niche LIKE '%GRR%'
            GROUP BY user_id
        """)
        
        grr_leads = cursor.fetchall()
        for user_id, count, statuses in grr_leads:
            print(f"User {user_id}: {count} leads with GRR niche (Target statuses: {statuses})")

finally:
    connection.close()
