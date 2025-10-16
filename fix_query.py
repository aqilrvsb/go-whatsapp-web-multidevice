import os

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\repository\lead_repository.go"
with open(file_path, 'r') as f:
    lines = f.readlines()

# Find and fix the line
for i, line in enumerate(lines):
    if line.strip() == "`" and i > 190 and i < 210:
        # This is line 194, change it to query :=
        lines[i] = "\tquery := `\n"
        break

# Write back
with open(file_path, 'w') as f:
    f.writelines(lines)

print("Fixed the query declaration issue!")
