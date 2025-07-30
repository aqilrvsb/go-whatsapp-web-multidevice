import os
import re

print("Fixing remaining syntax errors in infrastructure/whatsapp...")

# Dictionary of files and their specific fixes
fixes = {
    'chat_store.go': [
        ('`order`', 'ORDER'),  # SQL keyword should be uppercase without backticks
    ],
    'chat_to_leads.go': [
        ('target_`status`', 'target_status'),  # Remove incorrect backticks
        ('```trigger```', '`trigger`'),  # Fix triple backticks
        ('`status`', 'status'),  # status as column name doesn't need backticks in this context
        ('`trigger`', 'trigger'),  # Same for trigger
    ],
    'device_handler.go': [],  # Will check for select/case issues
    'device_health_monitor.go': [],  # Will check for select/case issues
    'device_status_normalizer.go': [],  # Will check for select/case issues
}

base_dir = r'src\infrastructure\whatsapp'

# First, fix specific string replacements
for filename, replacements in fixes.items():
    file_path = os.path.join(base_dir, filename)
    if os.path.exists(file_path):
        print(f"\nProcessing {filename}...")
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        for old, new in replacements:
            if old in content:
                content = content.replace(old, new)
                print(f"  - Replaced '{old}' with '{new}'")
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)

# Now fix select/case issues in Go code
print("\nFixing select/case syntax issues...")

for filename in ['device_handler.go', 'device_health_monitor.go', 'device_status_normalizer.go']:
    file_path = os.path.join(base_dir, filename)
    if os.path.exists(file_path):
        print(f"\nProcessing {filename}...")
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Fix SELECT { to select {
        content = re.sub(r'(\s+)SELECT\s*{', r'\1select {', content)
        
        # Fix any uppercase CASE to case in Go switch statements
        # This is tricky - we need to preserve SQL CASE but fix Go case
        # Look for patterns like "CASE <-" or "CASE variable:"
        content = re.sub(r'(\s+)CASE\s+(<-|\w+:)', r'\1case \2', content)
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"  - Fixed select/case keywords")

# Additional fix for SQL keywords that should be uppercase
print("\nFixing remaining SQL keywords...")

for filename in os.listdir(base_dir):
    if filename.endswith('.go'):
        file_path = os.path.join(base_dir, filename)
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original = content
        
        # Fix SQL keywords in SQL queries (inside backticks)
        # This regex looks for SQL queries and fixes keywords within them
        def fix_sql_keywords(match):
            sql = match.group(0)
            # Fix common SQL keywords
            sql = re.sub(r'\bselect\b', 'SELECT', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\bfrom\b', 'FROM', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\bwhere\b', 'WHERE', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\border\s+by\b', 'ORDER BY', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\bgroup\s+by\b', 'GROUP BY', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\blimit\b', 'LIMIT', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\band\b', 'AND', sql, flags=re.IGNORECASE)
            sql = re.sub(r'\bor\b', 'OR', sql, flags=re.IGNORECASE)
            return sql
        
        # Find SQL queries (between backticks) and fix them
        content = re.sub(r'`[^`]+`', fix_sql_keywords, content, flags=re.DOTALL)
        
        if content != original:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"  - Fixed SQL keywords in {filename}")

print("\nAll syntax errors should be fixed now!")
