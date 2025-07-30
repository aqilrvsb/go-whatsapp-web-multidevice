import re

print("Fixing remaining limit keyword occurrences...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix the UPDATE statement
old_update = '''status = ?, ai = ?, `limit` = ?, updated_at = ?'''
new_update = '''status = ?, ai = ?, ` + "`limit`" + ` = ?, updated_at = ?'''

content = content.replace(old_update, new_update)

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed remaining limit keyword occurrences!")
