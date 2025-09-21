import mysql.connector

conn = mysql.connector.connect(
    host='159.89.198.71',
    port=3306,
    user='admin_aqil',
    password='admin_aqil',
    database='admin_railway'
)
cursor = conn.cursor()

cursor.execute("""
    UPDATE campaigns 
    SET status = 'pending',
        time_schedule = TIME(DATE_SUB(NOW(), INTERVAL 1 MINUTE))
    WHERE id = 70
""")
conn.commit()

print("Campaign 70 reset to 'pending' status")
print("It will be processed in the next run cycle (within 5 minutes)")

conn.close()
