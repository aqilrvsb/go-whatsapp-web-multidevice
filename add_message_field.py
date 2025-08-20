import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\direct_broadcast_processor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Add Message field (sequences have both Message and Content)
content = re.sub(
    r'(Type:\s*"text",)',
    r'\1\n\t\t\tMessage:        campaign.Message,',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Added Message field to match sequences exactly!")
