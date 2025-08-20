import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Find and replace the problematic line
old_line = "AND (ud.status = 'connected' OR ud.status = 'online' OR ud.platform IS NOT NULL)"
new_line = "-- Device status check removed to allow campaigns to work with offline devices"

content = content.replace(old_line, new_line)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed the device status check in processCampaignDirect!")
print("Campaigns will now work regardless of device online status.")
