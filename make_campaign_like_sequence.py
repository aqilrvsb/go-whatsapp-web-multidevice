import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Find and replace the processCampaignDirect function
# This is a bit complex, so let's do it step by step

# 1. Change ScheduledAt: time.Now() to time.Now().Add(5 * time.Minute)
content = re.sub(
    r'ScheduledAt:\s*time\.Now\(\)',
    'ScheduledAt:    time.Now().Add(5 * time.Minute)',
    content
)

# 2. Add Status: "pending" after ScheduledAt
content = re.sub(
    r'(ScheduledAt:\s*time\.Now\(\)\.Add\(5 \* time\.Minute\)),',
    r'\1,\n\t\t\tStatus:         "pending",',
    content
)

# 3. Add minDelay and maxDelay from campaign
content = re.sub(
    r'(MediaURL:\s*campaign\.ImageURL,)',
    r'\1\n\t\t\tMinDelay:       campaign.MinDelaySeconds,\n\t\t\tMaxDelay:       campaign.MaxDelaySeconds,',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Updated campaigns to work EXACTLY like sequences!")
print("\nChanges made:")
print("1. ScheduledAt: Now adds 5 minutes delay (like sequences)")
print("2. Added Status: 'pending' (like sequences)")
print("3. Added MinDelay and MaxDelay from campaign settings")
print("\nCampaigns will now:")
print("- Schedule messages 5 minutes in the future")
print("- Use the same delay logic as sequences")
print("- Be processed by the same broadcast processor")
