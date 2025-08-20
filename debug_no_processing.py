import mysql.connector
from datetime import datetime

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

print("=== DEBUGGING WHY CAMPAIGN ISN'T PROCESSING ===\n")

# 1. Check campaign details
cursor.execute("""
    SELECT 
        id, title, status, 
        campaign_date, time_schedule,
        created_at, updated_at,
        STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') as scheduled_datetime,
        NOW() as current_time,
        CASE 
            WHEN STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') <= NOW() 
            THEN 'YES' ELSE 'NO' 
        END as should_trigger
    FROM campaigns 
    WHERE id = 70
""")
campaign = cursor.fetchone()

print(f"Campaign: {campaign['title']}")
print(f"Status: {campaign['status']}")
print(f"Scheduled: {campaign['scheduled_datetime']}")
print(f"Current Time: {campaign['current_time']}")
print(f"Should Trigger: {campaign['should_trigger']}")
print(f"Last Updated: {campaign['updated_at']}")

# 2. Check if the old campaign trigger is still running
print("\n=== CHECKING FOR OLD CAMPAIGN TRIGGER ===")
print("The logs show GetCampaignSummary but no campaign processing.")
print("This suggests:")
print("1. The new unified processor might not be deployed")
print("2. OR the old campaign trigger is disabled but new one isn't running")

# 3. Let's verify the campaign query works
cursor.execute("""
    SELECT COUNT(*) as count
    FROM campaigns c
    WHERE c.status = 'pending'
    AND (
        (c.scheduled_at IS NOT NULL AND c.scheduled_at <= NOW())
        OR
        (c.scheduled_at IS NULL AND 
         STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= NOW())
    )
""")
result = cursor.fetchone()
print(f"\nCampaigns ready to process: {result['count']}")

print("\n=== SOLUTION ===")
print("You need to:")
print("1. Make sure you deployed the new build")
print("2. Restart your application")
print("3. Look for this log on startup:")
print("   'Starting Direct Broadcast Processor (Sequences + Campaigns)...'")
print("4. Then every 5 minutes you should see:")
print("   '✅ Campaigns: Processed X campaigns'")
print("\nIf you don't see these logs, the unified processor isn't running!")

conn.close()
