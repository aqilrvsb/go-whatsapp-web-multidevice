import pymysql
from datetime import datetime
import sys

# Set UTF-8 encoding
sys.stdout.reconfigure(encoding='utf-8')

print("Testing duplicate prevention logic...")

# Connect to MySQL
connection = pymysql.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway',
    cursorclass=pymysql.cursors.DictCursor
)

cursor = connection.cursor()

# Check for campaign duplicates
print("\n=== CHECKING CAMPAIGN DUPLICATES ===")
campaign_duplicate_check = """
SELECT 
    recipient_phone,
    campaign_id,
    device_id,
    COUNT(*) as duplicate_count,
    GROUP_CONCAT(status) as statuses,
    GROUP_CONCAT(DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s')) as created_times
FROM broadcast_messages
WHERE campaign_id IS NOT NULL
GROUP BY recipient_phone, campaign_id, device_id
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC
LIMIT 10
"""

cursor.execute(campaign_duplicate_check)
campaign_duplicates = cursor.fetchall()

if len(campaign_duplicates) > 0:
    print(f"WARNING: Found {len(campaign_duplicates)} campaign duplicate entries!")
    for dup in campaign_duplicates[:5]:
        print(f"\nPhone: {dup['recipient_phone']}")
        print(f"Campaign ID: {dup['campaign_id']}")
        print(f"Device ID: {dup['device_id']}")
        print(f"Duplicate Count: {dup['duplicate_count']}")
        print(f"Statuses: {dup['statuses']}")
else:
    print("✅ No campaign duplicates found!")

# Check message ordering for campaigns
print("\n\n=== CHECKING CAMPAIGN MESSAGE ORDERING ===")
campaign_order_check = """
SELECT 
    device_id,
    COUNT(DISTINCT campaign_id) as campaigns_count,
    COUNT(*) as total_messages,
    MIN(scheduled_at) as first_scheduled,
    MAX(scheduled_at) as last_scheduled
FROM broadcast_messages
WHERE campaign_id IS NOT NULL
AND status = 'pending'
GROUP BY device_id
ORDER BY total_messages DESC
LIMIT 10
"""

cursor.execute(campaign_order_check)
results = cursor.fetchall()

print(f"Found {len(results)} devices with pending campaign messages")
for result in results[:5]:
    print(f"\nDevice: {result['device_id']}")
    print(f"Campaigns: {result['campaigns_count']}")
    print(f"Messages: {result['total_messages']}")
    print(f"First scheduled: {result['first_scheduled']}")
    print(f"Last scheduled: {result['last_scheduled']}")

cursor.close()
connection.close()
print("\n✅ Analysis complete!")
