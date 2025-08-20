import mysql.connector

# Connect to MySQL
conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor(dictionary=True)

# Check the campaign
cursor.execute("""
    SELECT 
        id, 
        title, 
        campaign_date, 
        time_schedule, 
        status,
        STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') as scheduled_datetime,
        NOW() as current_time,
        CASE 
            WHEN STR_TO_DATE(CONCAT(campaign_date, ' ', time_schedule), '%Y-%m-%d %H:%i:%s') <= NOW() 
            THEN 'YES - Should trigger now!'
            ELSE 'NO - Still future'
        END as should_trigger
    FROM campaigns
    WHERE id = 70
""")

result = cursor.fetchone()
if result:
    print(f"Campaign: {result['title']}")
    print(f"Status: {result['status']}")
    print(f"Scheduled: {result['scheduled_datetime']}")
    print(f"Current: {result['current_time']}")
    print(f"Should Trigger? {result['should_trigger']}")
    
    if result['should_trigger'].startswith('YES'):
        print("\nCAMPAIGN SHOULD TRIGGER ON NEXT RUN!")
        print("The processor runs every 5 minutes.")
        print("Watch your logs for: 'Processing campaign: kiki (Copy)'")
    
conn.close()
