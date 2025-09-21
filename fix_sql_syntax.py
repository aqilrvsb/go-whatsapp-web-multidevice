import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\campaign_status_monitor.go"
with open(file_path, 'r') as f:
    content = f.read()

# Fix the PostgreSQL EXTRACT(EPOCH FROM ...) to MySQL TIMESTAMPDIFF
old_line = "EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - MIN(CASE WHEN status = 'queued' THEN updated_at END)))/60 AS oldest_queued"
new_line = "TIMESTAMPDIFF(MINUTE, MIN(CASE WHEN status = 'queued' THEN updated_at END), CURRENT_TIMESTAMP) AS oldest_queued"

content = content.replace(old_line, new_line)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed SQL syntax error!")
print("Changed PostgreSQL EXTRACT(EPOCH FROM ...) to MySQL TIMESTAMPDIFF()")
