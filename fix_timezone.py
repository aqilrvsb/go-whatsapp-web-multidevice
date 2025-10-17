import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Find and replace the timezone issue
old_pattern = r"STR_TO_DATE\(CONCAT\(c\.campaign_date, ' ', COALESCE\(c\.time_schedule, '00:00:00'\)\), '%Y-%m-%d %H:%i:%s'\) <= NOW\(\)\)"
new_pattern = r"STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= DATE_ADD(NOW(), INTERVAL 8 HOUR))"

content = re.sub(old_pattern, new_pattern, content)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed timezone issue in ProcessCampaigns!")
print("Campaigns will now work with Malaysia timezone (+8 hours)")
print("\nThe fix changes:")
print("OLD: ... <= NOW()")
print("NEW: ... <= DATE_ADD(NOW(), INTERVAL 8 HOUR)")
print("\nThis means if you set a campaign for 14:21 Malaysia time,")
print("it will trigger when server time reaches 06:21 UTC.")
