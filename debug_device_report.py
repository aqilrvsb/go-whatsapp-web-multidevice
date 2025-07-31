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
        # Check campaign device assignments
        print("=== CAMPAIGN DEVICE REPORT DEBUG ===")
        
        # Get campaigns with GRR niche
        cursor.execute("""
            SELECT id, title, niche, target_status, user_id
            FROM campaigns
            WHERE niche = 'GRR'
            ORDER BY created_at DESC
            LIMIT 5
        """)
        
        campaigns = cursor.fetchall()
        for campaign in campaigns:
            print(f"\nCampaign {campaign[0]}: {campaign[1]}")
            print(f"  Niche: {campaign[2]}, Target: {campaign[3]}")
            
            # Count leads that should receive
            cursor.execute("""
                SELECT COUNT(DISTINCT l.phone) 
                FROM leads l
                WHERE l.user_id = %s 
                AND l.niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR l.target_status = %s)
            """, (campaign[4], campaign[2], campaign[3], campaign[3]))
            
            lead_count = cursor.fetchone()[0]
            print(f"  Total Leads (Should Send): {lead_count}")
            
            # Check broadcast messages for this campaign
            cursor.execute("""
                SELECT device_id, status, COUNT(*) as count
                FROM broadcast_messages
                WHERE campaign_id = %s
                GROUP BY device_id, status
            """, (campaign[0],))
            
            device_stats = cursor.fetchall()
            if device_stats:
                print("  Broadcast Messages by Device:")
                for device_id, status, count in device_stats:
                    print(f"    Device {device_id}: {status} = {count}")
            else:
                print("  No broadcast messages created yet")
            
            # Check devices for this user
            cursor.execute("""
                SELECT id, device_name, status
                FROM user_devices
                WHERE user_id = %s
                ORDER BY created_at DESC
            """, (campaign[4],))
            
            devices = cursor.fetchall()
            print(f"  User Devices ({len(devices)} total):")
            for device in devices:
                print(f"    {device[0]}: {device[1]} (Status: {device[2]})")
            
            # Check how leads are distributed
            cursor.execute("""
                SELECT device_id, COUNT(*) as count
                FROM leads
                WHERE user_id = %s 
                AND niche LIKE CONCAT('%%', %s, '%%')
                AND (%s = 'all' OR target_status = %s)
                GROUP BY device_id
            """, (campaign[4], campaign[2], campaign[3], campaign[3]))
            
            lead_distribution = cursor.fetchall()
            if lead_distribution:
                print("  Lead Distribution by Device:")
                for device_id, count in lead_distribution:
                    print(f"    Device {device_id}: {count} leads")

finally:
    connection.close()
