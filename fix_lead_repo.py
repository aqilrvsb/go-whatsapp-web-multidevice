import os

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\repository\lead_repository.go"
with open(file_path, 'r') as f:
    content = f.read()

# Replace the specific line
old_line = "rows, err := r.db.Query(query, deviceID, niche, status)"
new_line = "rows, err := r.db.Query(query, deviceID, niche, niche, status, status)"

content = content.replace(old_line, new_line)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed the SQL parameter issue!")
