import re

print("Properly fixing limit keyword in campaign_repository.go...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# The issue is the backticks are already there but not properly placed
# Fix the INSERT statement - the backticks around limit need to be inside the SQL string
content = content.replace('ai, `limit`, created_at', 'ai, `limit`, created_at')

# But the real issue is likely in the Go struct field access
# When accessing campaign.Limit in Go code, it should not have backticks
# Only in SQL strings should limit have backticks

# Let me check for patterns where limit appears outside SQL strings
# and ensure they don't have backticks

# First, let's properly escape limit in SQL strings only
# Pattern 1: In column lists
content = re.sub(r'(\s+)limit(\s*[,)])', r'\1`limit`\2', content, flags=re.IGNORECASE)
content = re.sub(r'([,\(]\s*)limit(\s*[,)])', r'\1`limit`\2', content, flags=re.IGNORECASE)

# But make sure we don't double-escape
content = content.replace('``limit``', '`limit`')

# Pattern 2: When limit is at the end of a line before FROM
content = re.sub(r'(\s+)limit(\s+FROM)', r'\1`limit`\2', content, flags=re.IGNORECASE)

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed!")

# Now let's also check what the actual errors are
print("\nChecking specific lines...")
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    lines = f.readlines()
    
# Check the problem lines
problem_lines = [70, 129, 165, 302, 480]
for line_num in problem_lines:
    if line_num <= len(lines):
        print(f"Line {line_num}: {lines[line_num-1].strip()}")
