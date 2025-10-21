import re

# Read the file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\repository\broadcast_repository.go"
with open(file_path, 'r') as f:
    content = f.read()

# Fix the double timezone offset
# Change: AND bm.scheduled_at <= DATE_ADD(?, INTERVAL 8 HOUR)
# To: AND bm.scheduled_at <= ?
content = re.sub(
    r'AND bm\.scheduled_at <= DATE_ADD\(\?, INTERVAL 8 HOUR\)',
    'AND bm.scheduled_at <= ?',
    content
)

# Also fix the other line
# Change: AND bm.scheduled_at >= DATE_ADD(DATE_SUB(?, INTERVAL 1 HOUR), INTERVAL 8 HOUR)
# To: AND bm.scheduled_at >= DATE_SUB(?, INTERVAL 1 HOUR)
content = re.sub(
    r'AND bm\.scheduled_at >= DATE_ADD\(DATE_SUB\(\?, INTERVAL 1 HOUR\), INTERVAL 8 HOUR\)',
    'AND bm.scheduled_at >= DATE_SUB(?, INTERVAL 1 HOUR)',
    content
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Fixed double timezone offset in GetPendingMessages!")
print("Campaign messages should now be sent properly.")
