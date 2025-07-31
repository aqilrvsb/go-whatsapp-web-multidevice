import pymysql
from datetime import datetime, date, timedelta
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
        # Check what dates are being used
        print("=== DATE DEBUGGING ===")
        
        # Get current date in different formats
        today_python = date.today()
        today_iso = today_python.isoformat()  # YYYY-MM-DD
        tomorrow_python = today_python + timedelta(days=1)
        tomorrow_iso = tomorrow_python.isoformat()
        
        print(f"Python Today: {today_iso}")
        print(f"Python Tomorrow: {tomorrow_iso}")
        
        # Check MySQL's current date
        cursor.execute("SELECT CURDATE(), NOW(), @@time_zone")
        mysql_date, mysql_now, mysql_tz = cursor.fetchone()
        print(f"\nMySQL CURDATE(): {mysql_date}")
        print(f"MySQL NOW(): {mysql_now}")
        print(f"MySQL Timezone: {mysql_tz}")
        
        # Check campaign dates
        print("\n=== CAMPAIGN DATES ===")
        cursor.execute("""
            SELECT id, title, campaign_date, created_at
            FROM campaigns
            ORDER BY created_at DESC
            LIMIT 10
        """)
        
        campaigns = cursor.fetchall()
        for camp in campaigns:
            print(f"Campaign {camp[0]}: {camp[1]}")
            print(f"  campaign_date: {camp[2]} (type: {type(camp[2])})")
            print(f"  created_at: {camp[3]}")
            print(f"  Is today? {camp[2] == mysql_date}")
        
        # Test the date range query that the API uses
        print(f"\n=== TESTING DATE RANGE QUERY ===")
        print(f"Testing with start_date='{today_iso}' and end_date='{tomorrow_iso}'")
        
        # Test query 1: Direct date comparison
        cursor.execute("""
            SELECT COUNT(*) 
            FROM campaigns 
            WHERE campaign_date >= %s AND campaign_date <= %s
        """, (today_iso, tomorrow_iso))
        count1 = cursor.fetchone()[0]
        print(f"Query 1 (>= and <=): {count1} campaigns")
        
        # Test query 2: Using DATE() function
        cursor.execute("""
            SELECT COUNT(*) 
            FROM campaigns 
            WHERE DATE(campaign_date) >= %s AND DATE(campaign_date) <= %s
        """, (today_iso, tomorrow_iso))
        count2 = cursor.fetchone()[0]
        print(f"Query 2 (DATE() function): {count2} campaigns")
        
        # Test query 3: Using BETWEEN
        cursor.execute("""
            SELECT COUNT(*) 
            FROM campaigns 
            WHERE campaign_date BETWEEN %s AND %s
        """, (today_iso, tomorrow_iso))
        count3 = cursor.fetchone()[0]
        print(f"Query 3 (BETWEEN): {count3} campaigns")
        
        # Test query 4: Just today
        cursor.execute("""
            SELECT COUNT(*) 
            FROM campaigns 
            WHERE DATE(campaign_date) = CURDATE()
        """)
        count4 = cursor.fetchone()[0]
        print(f"Query 4 (= CURDATE()): {count4} campaigns")
        
        # Show exact campaign dates for debugging
        print("\n=== CAMPAIGN DATE VALUES ===")
        cursor.execute("""
            SELECT id, title, campaign_date, 
                   DATE(campaign_date) as date_only,
                   campaign_date = %s as matches_today
            FROM campaigns
            WHERE campaign_date >= DATE_SUB(CURDATE(), INTERVAL 1 DAY)
            ORDER BY campaign_date DESC
        """, (today_iso,))
        
        results = cursor.fetchall()
        for r in results:
            print(f"ID: {r[0]}, Title: {r[1]}")
            print(f"  campaign_date: {r[2]}")
            print(f"  DATE(campaign_date): {r[3]}")
            print(f"  Matches '{today_iso}': {r[4]}")

finally:
    connection.close()
