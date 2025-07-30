import os
import re

print("Comprehensive fix for all Go syntax errors...")

base_dir = r'src\infrastructure\whatsapp'

# List of files with select/case issues from the error log
problem_files = [
    'event_processor.go',
    'keepalive_manager.go', 
    'multidevice_auto_reconnect.go',
    'optimized_client_manager.go'
]

# Process all Go files in the directory
for filename in os.listdir(base_dir):
    if filename.endswith('.go'):
        file_path = os.path.join(base_dir, filename)
        print(f"\nProcessing {filename}...")
        
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original = content
        
        # Fix 1: SELECT { should be select {
        content = re.sub(r'(\s+)SELECT\s*{', r'\1select {', content)
        
        # Fix 2: CASE in Go code should be case
        # Look for CASE followed by <- (channel) or : (label)
        content = re.sub(r'(\s+)CASE\s+', r'\1case ', content)
        
        # Fix 3: DEFAULT in Go code should be default
        content = re.sub(r'(\s+)DEFAULT\s*:', r'\1default:', content)
        
        # Fix 4: Fix SQL keywords that are incorrectly lowercase
        # But only inside SQL query strings (between backticks)
        def fix_sql_in_backticks(match):
            sql = match.group(0)
            # Only fix if it looks like SQL (has SELECT, FROM, etc.)
            if any(keyword in sql.upper() for keyword in ['SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE']):
                sql = re.sub(r'\bselect\b', 'SELECT', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bfrom\b', 'FROM', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bwhere\b', 'WHERE', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\border\s+by\b', 'ORDER BY', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bgroup\s+by\b', 'GROUP BY', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\blimit\b', 'LIMIT', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\band\b', 'AND', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bor\b', 'OR', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\binsert\s+into\b', 'INSERT INTO', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bupdate\b', 'UPDATE', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bdelete\s+from\b', 'DELETE FROM', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bset\b', 'SET', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bvalues\b', 'VALUES', sql, flags=re.IGNORECASE)
            return sql
        
        # Apply SQL fixes only to content between backticks
        content = re.sub(r'`[^`]+`', fix_sql_in_backticks, content, flags=re.DOTALL)
        
        # Fix 5: Remove any remaining backticks around Go keywords (not in SQL strings)
        # This is for cases where backticks are misused outside of SQL
        lines = content.split('\n')
        for i, line in enumerate(lines):
            # Skip lines that are part of SQL queries (contain query := or are indented SQL)
            if 'query :=' not in line and not line.strip().startswith('`'):
                # Remove backticks around keywords like order, limit, etc when not in SQL context
                line = re.sub(r'`(order|limit|status|type|case|select|from)`', r'\1', line)
            lines[i] = line
        content = '\n'.join(lines)
        
        if content != original:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"  + Fixed syntax issues")
        else:
            print(f"  - No changes needed")

# Special fix for multidevice_auto_reconnect.go line 29
file_path = os.path.join(base_dir, 'multidevice_auto_reconnect.go')
if os.path.exists(file_path):
    print(f"\nSpecial fix for multidevice_auto_reconnect.go...")
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    
    # Check line 29 (index 28)
    if len(lines) > 28:
        # Look for 'order' used incorrectly
        if 'order' in lines[28].lower() and 'ORDER BY' not in lines[28]:
            # This might be a SQL query missing proper formatting
            lines[28] = lines[28].replace('order', 'ORDER')
    
    with open(file_path, 'w', encoding='utf-8') as f:
        f.writelines(lines)

print("\nAll syntax fixes applied!")
