import re

print("Fixing backtick syntax for MySQL limit keyword...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix the backtick syntax - needs to be inside the SQL string
# Pattern 1: In INSERT statement
old_insert = """INSERT INTO campaigns(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 time_schedule, min_delay_seconds, max_delay_seconds, status, ai, `limit`, created_at, updated_at)"""

new_insert = """INSERT INTO campaigns(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 time_schedule, min_delay_seconds, max_delay_seconds, status, ai, """ + "`limit`" + """, created_at, updated_at)"""

content = content.replace(old_insert, new_insert)

# Pattern 2: In SELECT statements - fix all occurrences
content = re.sub(r'(\s+)(`limit`)(\s+)', r'\1' + '`limit`' + r'\3', content)

# Pattern 3: Fix in COALESCE statements
content = re.sub(r'COALESCE\(`limit`,', 'COALESCE(`limit`,', content)

# Pattern 4: Fix any remaining standalone `limit`
content = re.sub(r'SELECT ([^`]*)`limit`', r'SELECT \1' + '`limit`', content)

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed backtick syntax!")
