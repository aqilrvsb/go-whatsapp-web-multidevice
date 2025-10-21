import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# For campaigns, we need to subtract 8 hours from scheduled time
# because the campaign times are in Malaysia timezone (+8)
# but the server processes in UTC

# Change: ScheduledAt: time.Now().Add(5 * time.Minute)
# To: ScheduledAt: time.Now().Add(5 * time.Minute).Add(-8 * time.Hour)
content = re.sub(
    r'ScheduledAt:\s*time\.Now\(\)\.Add\(5 \* time\.Minute\)',
    'ScheduledAt:    time.Now().Add(5 * time.Minute).Add(-8 * time.Hour)',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed campaign enrollment timezone!")
print("\nCampaigns now:")
print("1. Subtract 8 hours from scheduled time (Malaysia to UTC conversion)")
print("2. Then add 5 minutes delay (like sequences)")
print("\nExample:")
print("- Malaysia time: 14:00 (2:00 PM)")
print("- UTC time: 06:00 (6:00 AM)")
print("- Scheduled at: 06:05 (5 minutes after UTC time)")
