import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# UNDO - Put back the 8 hour addition
# Change: ScheduledAt: time.Now().Add(5 * time.Minute)
# Back to: ScheduledAt: time.Now().Add(5 * time.Minute).Add(8 * time.Hour)
content = re.sub(
    r'ScheduledAt:\s*time\.Now\(\)\.Add\(5 \* time\.Minute\)',
    'ScheduledAt:    time.Now().Add(5 * time.Minute).Add(8 * time.Hour)',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("UNDONE - Restored the +8 hour offset for campaigns")
print("Now back to: time.Now().Add(5 * time.Minute).Add(8 * time.Hour)")
