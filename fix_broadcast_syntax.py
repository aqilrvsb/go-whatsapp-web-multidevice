import os
import re

print("Fixing syntax errors in infrastructure/broadcast directory...")

base_dir = r'src\infrastructure\broadcast'

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
        content = re.sub(r'(\s+)CASE\s+', r'\1case ', content)
        
        # Fix 3: DEFAULT in Go code should be default
        content = re.sub(r'(\s+)DEFAULT\s*:', r'\1default:', content)
        
        # Fix 4: Remove backticks around SQL keywords
        sql_keywords = ['SELECT', 'FROM', 'WHERE', 'ORDER', 'GROUP', 'LIMIT', 
                       'INSERT', 'UPDATE', 'DELETE', 'AND', 'OR', 'BY', 
                       'ASC', 'DESC', 'INTO', 'VALUES', 'SET', 'STATUS']
        
        for keyword in sql_keywords:
            content = content.replace(f'`{keyword}`', keyword)
            content = content.replace(f'`{keyword.lower()}`', keyword)
        
        # Fix 5: Fix SQL keywords in SQL queries
        def fix_sql_in_backticks(match):
            sql = match.group(0)
            if any(kw in sql.upper() for kw in ['SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE']):
                sql = re.sub(r'\bselect\b', 'SELECT', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bfrom\b', 'FROM', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bwhere\b', 'WHERE', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\border\s+by\b', 'ORDER BY', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\blimit\b', 'LIMIT', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\band\b', 'AND', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bupdate\b', 'UPDATE', sql, flags=re.IGNORECASE)
                sql = re.sub(r'\bset\b', 'SET', sql, flags=re.IGNORECASE)
            return sql
        
        content = re.sub(r'`[^`]+`', fix_sql_in_backticks, content, flags=re.DOTALL)
        
        # Fix 6: Special handling for status in SQL contexts
        # If 'status' appears in backticks in a SQL query, it's likely a column name
        # But if it appears with backticks outside SQL, remove them
        lines = content.split('\n')
        in_sql = False
        for i, line in enumerate(lines):
            if 'query :=' in line or (i > 0 and 'query :=' in lines[i-1]):
                in_sql = True
            elif ';' in line and in_sql:
                in_sql = False
            
            if not in_sql and '`status`' in line:
                lines[i] = line.replace('`status`', 'status')
        
        content = '\n'.join(lines)
        
        if content != original:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"  + Fixed syntax issues")
        else:
            print(f"  - No changes needed")

# Now let's also fix any other directories that might have issues
other_dirs = ['src/infrastructure/sequence', 'src/usecase', 'src/ui/rest']

for dir_path in other_dirs:
    if os.path.exists(dir_path):
        print(f"\nChecking {dir_path}...")
        for filename in os.listdir(dir_path):
            if filename.endswith('.go'):
                file_path = os.path.join(dir_path, filename)
                
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                original = content
                
                # Apply same fixes
                content = re.sub(r'(\s+)SELECT\s*{', r'\1select {', content)
                content = re.sub(r'(\s+)CASE\s+', r'\1case ', content)
                content = re.sub(r'(\s+)DEFAULT\s*:', r'\1default:', content)
                
                if content != original:
                    with open(file_path, 'w', encoding='utf-8') as f:
                        f.write(content)
                    print(f"  + Fixed {filename}")

print("\nAll broadcast and other syntax errors fixed!")
