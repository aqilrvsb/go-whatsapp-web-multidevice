import pymysql
from datetime import datetime
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
        # Test 1: Check campaigns
        print("=== CAMPAIGNS ===")
        cursor.execute("SELECT id, title, niche, target_status, status FROM campaigns LIMIT 5")
        campaigns = cursor.fetchall()
        for campaign in campaigns:
            print(f"Campaign {campaign[0]}: {campaign[1]} - Niche: {campaign[2]}, Target: {campaign[3]}, Status: {campaign[4]}")
        
        if campaigns:
            # Test 2: Check leads matching for first campaign
            campaign_id = campaigns[0][0]
            niche = campaigns[0][2]
            target_status = campaigns[0][3]
            
            print(f"\n=== TESTING CAMPAIGN {campaign_id} ===")
            print(f"Niche: {niche}, Target Status: {target_status}")
            
            # Get user_id for the campaign
            cursor.execute("SELECT user_id FROM campaigns WHERE id = %s", (campaign_id,))
            user_id = cursor.fetchone()[0]
            
            # Test the exact query used in GetCampaignBroadcastStats
            query = """
                SELECT COUNT(l.phone) 
                FROM leads l
                WHERE l.user_id = %s 
                AND l.niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR l.target_status = %s)
            """
            cursor.execute(query, (user_id, niche, target_status, target_status))
            should_send = cursor.fetchone()[0]
            print(f"Should Send Count: {should_send}")
            
            # Debug: Show some matching leads
            debug_query = """
                SELECT l.phone, l.niche, l.target_status 
                FROM leads l
                WHERE l.user_id = %s 
                AND l.niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR l.target_status = %s)
                LIMIT 5
            """
            cursor.execute(debug_query, (user_id, niche, target_status, target_status))
            matching_leads = cursor.fetchall()
            print("\nMatching Leads (first 5):")
            for lead in matching_leads:
                print(f"  Phone: {lead[0]}, Niche: {lead[1]}, Target: {lead[2]}")
            
            # Test 3: Check broadcast messages
            print("\n=== BROADCAST MESSAGES ===")
            cursor.execute("""
                SELECT COUNT(CASE WHEN status = 'success' THEN 1 END) AS done_send,
                    COUNT(CASE WHEN status = 'failed' THEN 1 END) AS failed_send
                FROM broadcast_messages
                WHERE campaign_id = %s
            """, (campaign_id,))
            result = cursor.fetchone()
            print(f"Done Send: {result[0]}, Failed Send: {result[1]}")
            
            # Check what statuses exist in broadcast_messages
            cursor.execute("""
                SELECT status, COUNT(*) as count
                FROM broadcast_messages
                WHERE campaign_id = %s
                GROUP BY status
            """, (campaign_id,))
            status_counts = cursor.fetchall()
            print("\nBroadcast Message Status Breakdown:")
            for status, count in status_counts:
                print(f"  {status}: {count}")
        
        # Test 4: Check distinct niches in leads
        print("\n=== DISTINCT NICHES IN LEADS ===")
        cursor.execute("SELECT DISTINCT niche FROM leads WHERE niche IS NOT NULL LIMIT 10")
        niches = cursor.fetchall()
        for niche in niches:
            print(f"  Niche: {niche[0]}")
        
        # Test 5: Check distinct target_status in leads
        print("\n=== DISTINCT TARGET STATUS IN LEADS ===")
        cursor.execute("SELECT DISTINCT target_status FROM leads WHERE target_status IS NOT NULL")
        statuses = cursor.fetchall()
        for status in statuses:
            print(f"  Target Status: {status[0]}")

finally:
    connection.close()
