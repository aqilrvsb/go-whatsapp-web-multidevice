import re

print("Properly fixing limit keyword with Go string concatenation...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix Pattern 1: In the INSERT query
old_query = '''query := `
		INSERT INTO campaigns(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 time_schedule, min_delay_seconds, max_delay_seconds, status, ai, `limit`, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`'''

new_query = '''query := `
		INSERT INTO campaigns(user_id, campaign_date, title, niche, target_status, message, image_url, 
		 time_schedule, min_delay_seconds, max_delay_seconds, status, ai, ` + "`limit`" + `, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`'''

content = content.replace(old_query, new_query)

# Fix Pattern 2: In SELECT queries - find all occurrences
# Look for patterns like "ai, `limit`, created_at" and fix them
content = re.sub(r', `limit`,', r', ` + "`limit`" + `,', content)
content = re.sub(r'COALESCE\(`limit`,', r'COALESCE(` + "`limit`" + `,', content)

# Fix Pattern 3: Fix c.Limit references in SELECT
# These should remain as is (without backticks) when referencing the struct field

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed limit keyword with proper Go concatenation!")
