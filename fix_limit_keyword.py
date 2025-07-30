import re

print("Fixing limit keyword issues in campaign_repository.go...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix all instances where limit is used as a column name
# In INSERT statements
content = re.sub(r',\s*`limit`\s*,', ', `limit`,', content)
content = re.sub(r',\s*`limit`\s*\)', ', `limit`)', content)

# In SELECT statements
content = re.sub(r'COALESCE\(`limit`, 0\) AS `limit`', 'COALESCE(`limit`, 0) AS campaign_limit', content)
content = re.sub(r',\s*`limit`\s*FROM', ', `limit` FROM', content)

# Fix any standalone limit references
content = re.sub(r'(\s+)limit(\s+)FROM', r'\1`limit`\2FROM', content)
content = re.sub(r'(\s+)limit(\s*,)', r'\1`limit`\2', content)

# Write back
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed limit keyword issues!")
