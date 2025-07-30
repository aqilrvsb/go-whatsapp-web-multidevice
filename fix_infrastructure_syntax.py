import os
import re

print("Fixing Go syntax errors in infrastructure/whatsapp directory...")

# Directory to process
infra_dir = r'src\infrastructure\whatsapp'

fixes_applied = {}

for filename in os.listdir(infra_dir):
    if filename.endswith('.go'):
        filepath = os.path.join(infra_dir, filename)
        print(f"\nProcessing {filename}...")
        
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        fixes = []
        
        # Fix 1: SELECT should be select in Go code (not SQL)
        if 'SELECT {' in content:
            content = content.replace('SELECT {', 'select {')
            fixes.append("Fixed SELECT -> select in Go switch statement")
        
        # Fix 2: Fix SQL keywords that have incorrect backticks
        # `order` should be ORDER in SQL
        content = re.sub(r'`order`\s+BY', 'ORDER BY', content, flags=re.IGNORECASE)
        if 'ORDER BY' in content and '`order`' in original_content:
            fixes.append("Fixed `order` BY -> ORDER BY")
        
        # Fix 3: Fix 'from' that should be FROM in SQL
        content = re.sub(r'(\n\s+)from\s+', r'\1FROM ', content)
        if 'FROM ' in content and original_content != content:
            fixes.append("Fixed 'from' -> FROM")
        
        # Fix 4: Fix 'limit' that should be LIMIT in SQL (when not a column name)
        content = re.sub(r'(\n\s+)limit\s+(\d+)', r'\1LIMIT \2', content)
        if 'LIMIT ' in content and original_content != content:
            fixes.append("Fixed 'limit' -> LIMIT")
        
        # Fix 5: Fix case sensitivity in SQL keywords
        sql_keywords = ['SELECT', 'FROM', 'WHERE', 'ORDER BY', 'GROUP BY', 'LIMIT', 'INSERT', 'UPDATE', 'DELETE', 'AND', 'OR']
        for keyword in sql_keywords:
            # Only fix if it's clearly in SQL context (has newline and indent before it)
            pattern = rf'(\n\s+){keyword.lower()}(\s+)'
            replacement = rf'\1{keyword}\2'
            content = re.sub(pattern, replacement, content)
        
        # Fix 6: Remove backticks from SQL keywords (not column names)
        content = re.sub(r'`(SELECT|FROM|WHERE|ORDER|GROUP|LIMIT|AND|OR|INSERT|UPDATE|DELETE)`', r'\1', content, flags=re.IGNORECASE)
        
        # Fix 7: type as variable name - need to see context
        # Fix 8: status as variable name - need to see context
        # Fix 9: case in wrong context - need to see context
        
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            fixes_applied[filename] = fixes
            print(f"  + Fixed: {', '.join(fixes)}")
        else:
            print(f"  - No SQL keyword fixes needed")

print("\n" + "="*50)
print("Summary of fixes applied:")
for file, fixes in fixes_applied.items():
    print(f"\n{file}:")
    for fix in fixes:
        print(f"  - {fix}")

# Now let's check specific error locations
print("\n" + "="*50)
print("Checking specific error locations...")

# Check auto_reconnect.go line 42
file_path = os.path.join(infra_dir, 'auto_reconnect.go')
if os.path.exists(file_path):
    with open(file_path, 'r') as f:
        lines = f.readlines()
    if len(lines) > 42:
        print(f"\nauto_reconnect.go line 42: {lines[41].strip()}")

# Check chat_store.go line 339
file_path = os.path.join(infra_dir, 'chat_store.go')
if os.path.exists(file_path):
    with open(file_path, 'r') as f:
        lines = f.readlines()
    if len(lines) > 339:
        print(f"\nchat_store.go line 339: {lines[338].strip()}")

# Check chat_to_leads.go line 47
file_path = os.path.join(infra_dir, 'chat_to_leads.go')
if os.path.exists(file_path):
    with open(file_path, 'r') as f:
        lines = f.readlines()
    if len(lines) > 47:
        print(f"\nchat_to_leads.go line 47: {lines[46].strip()}")
