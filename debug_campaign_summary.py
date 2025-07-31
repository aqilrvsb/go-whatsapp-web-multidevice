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
        # Check today's campaigns
        print("=== TODAY'S CAMPAIGNS ===")
        cursor.execute("""
            SELECT id, title, status, campaign_date, created_at, user_id
            FROM campaigns
            WHERE DATE(campaign_date) = CURDATE()
            ORDER BY created_at DESC
        """)
        campaigns = cursor.fetchall()
        
        print(f"Found {len(campaigns)} campaigns for today")
        for campaign in campaigns:
            print(f"\nCampaign ID: {campaign[0]}")
            print(f"Title: {campaign[1]}")
            print(f"Status: {campaign[2]}")
            print(f"Campaign Date: {campaign[3]}")
            print(f"Created At: {campaign[4]}")
            print(f"User ID: {campaign[5]}")
            
            # Check broadcast messages for this campaign
            cursor.execute("""
                SELECT COUNT(*) as total,
                    COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
                    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending
                FROM broadcast_messages
                WHERE campaign_id = %s
            """, (campaign[0],))
            
            stats = cursor.fetchone()
            print(f"Broadcast Messages - Total: {stats[0]}, Success: {stats[1]}, Failed: {stats[2]}, Pending: {stats[3]}")
        
        # Check all broadcast messages with campaign_id
        print("\n=== ALL CAMPAIGN BROADCAST MESSAGES ===")
        cursor.execute("""
            SELECT campaign_id, COUNT(*) as count,
                COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
                COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending
            FROM broadcast_messages
            WHERE campaign_id IS NOT NULL
            GROUP BY campaign_id
            ORDER BY campaign_id DESC
            LIMIT 10
        """)
        
        campaign_messages = cursor.fetchall()
        if campaign_messages:
            for cm in campaign_messages:
                print(f"Campaign {cm[0]}: Total={cm[1]}, Success={cm[2]}, Failed={cm[3]}, Pending={cm[4]}")
        else:
            print("No broadcast messages found for any campaigns")
        
        # Check the campaign summary query logic
        print("\n=== TESTING CAMPAIGN SUMMARY QUERY ===")
        # Get a sample user_id
        cursor.execute("SELECT DISTINCT user_id FROM campaigns LIMIT 1")
        user_id = cursor.fetchone()
        if user_id:
            user_id = user_id[0]
            print(f"Testing with user_id: {user_id}")
            
            # Get campaigns for this user
            cursor.execute("""
                SELECT id FROM campaigns 
                WHERE user_id = %s
                ORDER BY created_at DESC
                LIMIT 5
            """, (user_id,))
            
            campaign_ids = [row[0] for row in cursor.fetchall()]
            print(f"Campaign IDs for user: {campaign_ids}")
            
            if campaign_ids:
                # Build the IN clause query
                placeholders = ','.join(['%s'] * len(campaign_ids))
                query = f"""
                    SELECT 
                        COUNT(*) as total,
                        COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
                        COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                        COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending
                    FROM broadcast_messages
                    WHERE campaign_id IN ({placeholders})
                """
                
                cursor.execute(query, campaign_ids)
                result = cursor.fetchone()
                print(f"Aggregate stats: Total={result[0]}, Success={result[1]}, Failed={result[2]}, Pending={result[3]}")

finally:
    connection.close()
