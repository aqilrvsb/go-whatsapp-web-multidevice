import re

# Read the file
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\ui\rest\app.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Replace pattern for date filters
date_filter_replacement = '''if startDate != "" && endDate != "" {
			query += ` AND DATE(scheduled_at) BETWEEN ? AND ?`
			args = append(args, startDate, endDate)
		} else if startDate != "" {
			query += ` AND DATE(scheduled_at) >= ?`
			args = append(args, startDate)
		} else if endDate != "" {
			query += ` AND DATE(scheduled_at) <= ?`
			args = append(args, endDate)
		}'''

# Replace all occurrences of showTodayOnly date filter
pattern = r'if showTodayOnly \{\s*query \+= ` AND DATE\(scheduled_at\) = CURDATE\(\)`\s*\}'
content = re.sub(pattern, date_filter_replacement, content)

# Write back
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\ui\rest\app.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Replacements done!")
