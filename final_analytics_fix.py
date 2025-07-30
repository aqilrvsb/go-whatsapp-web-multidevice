import re

print("Final fix for analytics handlers...")

file_path = r'src\ui\rest\analytics_handlers.go'

with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Remove unused import
content = content.replace('"github.com/aldinokemal/go-whatsapp-web-multidevice/config"\n\t', '')

# Fix the database.GetDB() usage - it returns only 1 value, not 2
# Change patterns like "db, err := database.GetDB()" to "db := database.GetDB()"
content = re.sub(r'(\s+)db, err := database\.GetDB\(\)', r'\1db := database.GetDB()', content)

# Remove the error handling blocks that follow since there's no error
# This regex removes the if err != nil blocks
content = re.sub(r'\s*if err != nil \{[^}]+return[^}]+\}', '', content)

# Remove any empty lines that might have been left
content = re.sub(r'\n\s*\n\s*\n', '\n\n', content)

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed all issues in analytics handlers!")
