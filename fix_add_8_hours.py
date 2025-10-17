import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Change from subtract to ADD 8 hours
# Change: ScheduledAt: time.Now().Add(5 * time.Minute).Add(-8 * time.Hour)
# To: ScheduledAt: time.Now().Add(5 * time.Minute).Add(8 * time.Hour)
content = re.sub(
    r'ScheduledAt:\s*time\.Now\(\)\.Add\(5 \* time\.Minute\)\.Add\(-8 \* time\.Hour\)',
    'ScheduledAt:    time.Now().Add(5 * time.Minute).Add(8 * time.Hour)',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("FIXED! Now ADDING 8 hours instead of subtracting!")
print("\nCampaigns now:")
print("1. Add 5 minutes delay (like sequences)")
print("2. Add 8 hours for timezone adjustment")
print("\nExample:")
print("- Server time (UTC): 06:00")
print("- Scheduled at: 14:05 (8 hours later + 5 minutes)")
print("- This matches Malaysia time!")
