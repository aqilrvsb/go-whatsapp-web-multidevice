import pymysql
import os
import sys
from datetime import datetime
from tabulate import tabulate

# Set UTF-8 encoding for Windows
if sys.platform == 'win32':
    sys.stdout.reconfigure(encoding='utf-8')

# Get MySQL connection from environment
mysql_uri = os.getenv('MYSQL_URI', 'mysql://admin_aqil:admin_aqil@159.89.198.71:3306/admin_railway')

# Parse MySQL URI
if mysql_uri.startswith('mysql://'):
    mysql_uri = mysql_uri[8:]  # Remove mysql://
    
parts = mysql_uri.split('@')
user_pass = parts[0].split(':')
host_db = parts[1].split('/')

user = user_pass[0]
password = user_pass[1]
host_port = host_db[0].split(':')
host = host_port[0]
port = int(host_port[1]) if len(host_port) > 1 else 3306
database = host_db[1].split('?')[0]

# List of devices
DEVICE_NAMES = [
    'SCAST-S30', 'SCARS-S46', 'SCRY-S08', 'SCAS-S74', 'SCARR-S39',
    'SCSHQ-S05', 'SCARS-S35', 'SCAS-S05', 'SMHQ-S05', 'SCAST-S59',
    'SCTTN-S77', 'SCAS-S40', 'SCHQ-S105', 'SCHQ-S02', 'SCHQ-S09'
]

try:
    # Connect to MySQL
    connection = pymysql.connect(
        host=host,
        port=port,
        user=user,
        password=password,
        database=database,
        cursorclass=pymysql.cursors.DictCursor
    )
    
    print("Connected to MySQL database")
    print("=" * 120)
    print("\nRESCHEDULED MESSAGES SUMMARY REPORT")
    print("=" * 120)
    print(f"Report generated at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("\n")
    
    with connection.cursor() as cursor:
        # Get summary for each device
        device_data = []
        total_pending = 0
        
        for device_name in DEVICE_NAMES:
            cursor.execute("""
                SELECT 
                    ud.id,
                    ud.platform,
                    COUNT(CASE WHEN bm.status = 'pending' AND DATE(bm.scheduled_at) >= CURDATE() THEN 1 END) as pending_count,
                    COUNT(CASE WHEN bm.status = 'failed' THEN 1 END) as remaining_failed,
                    MIN(CASE WHEN bm.status = 'pending' AND bm.scheduled_at >= NOW() THEN bm.scheduled_at END) as next_send,
                    MAX(CASE WHEN bm.status = 'pending' THEN bm.scheduled_at END) as last_send
                FROM user_devices ud
                LEFT JOIN broadcast_messages bm ON bm.device_id = ud.id
                WHERE ud.device_name = %s
                GROUP BY ud.id, ud.platform
            """, (device_name,))
            
            result = cursor.fetchone()
            if result:
                device_data.append([
                    device_name,
                    result['platform'] or 'whatsapp',
                    result['pending_count'],
                    result['remaining_failed'],
                    result['next_send'].strftime('%d %b %H:%M') if result['next_send'] else 'None',
                    result['last_send'].strftime('%d %b %H:%M') if result['last_send'] else 'None'
                ])
                total_pending += result['pending_count']
        
        # Display table
        headers = ['Device Name', 'Platform', 'Pending', 'Still Failed', 'Next Send', 'Last Send']
        print(tabulate(device_data, headers=headers, tablefmt='grid'))
        
        print(f"\n\nTOTAL MESSAGES RESCHEDULED: {total_pending}")
        
        # Show hourly distribution
        print("\n\nMESSAGES SCHEDULED BY HOUR (Next 24 hours):")
        print("-" * 60)
        
        cursor.execute("""
            SELECT 
                DATE_FORMAT(bm.scheduled_at, '%Y-%m-%d %H:00') as hour,
                COUNT(*) as message_count
            FROM broadcast_messages bm
            JOIN user_devices ud ON ud.id = bm.device_id
            WHERE ud.device_name IN (%s)
            AND bm.status = 'pending'
            AND bm.scheduled_at BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 24 HOUR)
            GROUP BY hour
            ORDER BY hour
            LIMIT 24
        """ % ','.join(['%s'] * len(DEVICE_NAMES)), DEVICE_NAMES)
        
        hourly = cursor.fetchall()
        
        for hour_data in hourly:
            print(f"{hour_data['hour']}: {hour_data['message_count']} messages")
            
        # Create summary file
        with open('reschedule_summary.txt', 'w', encoding='utf-8') as f:
            f.write("RESCHEDULED MESSAGES SUMMARY\n")
            f.write("=" * 60 + "\n")
            f.write(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
            
            for device in device_data:
                f.write(f"{device[0]}: {device[2]} messages pending\n")
                if device[4] != 'None':
                    f.write(f"  Next: {device[4]}, Last: {device[5]}\n")
                f.write("\n")
                
            f.write(f"\nTOTAL: {total_pending} messages rescheduled\n")
            
        print("\n\nSummary saved to: reschedule_summary.txt")
        
except Exception as e:
    print(f"Error: {e}")
    import traceback
    traceback.print_exc()
finally:
    if 'connection' in locals() and connection:
        connection.close()
        print("\nDatabase connection closed")
