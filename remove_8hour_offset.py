import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# REMOVE the 8 hour addition for campaigns
# Change: ScheduledAt: time.Now().Add(5 * time.Minute).Add(8 * time.Hour)
# To: ScheduledAt: time.Now().Add(5 * time.Minute)
content = re.sub(
    r'ScheduledAt:\s*time\.Now\(\)\.Add\(5 \* time\.Minute\)\.Add\(8 \* time\.Hour\)',
    'ScheduledAt:    time.Now().Add(5 * time.Minute)',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed campaign scheduling!")
print("\nNow campaigns will:")
print("1. Schedule messages 5 minutes from server time (like sequences)")
print("2. NO timezone adjustment in enrollment")
print("\nThe timezone handling should be done in the UI when creating campaigns,")
print("not in the backend processing.")
