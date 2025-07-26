import re

# Read the file
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence_trigger_processor.go', 'r') as f:
    content = f.read()

# Replace the specific line
old_line = 'WHERE trigger = $1 AND is_active = true'
new_line = 'WHERE trigger = $1'

content = content.replace(old_line, new_line)

# Write back
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence_trigger_processor.go', 'w') as f:
    f.write(content)

print("Fixed! Removed is_active check from sequence linking")
