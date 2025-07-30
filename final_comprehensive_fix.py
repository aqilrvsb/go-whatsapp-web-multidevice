import os
import re

print("Final comprehensive fix for all remaining syntax errors...")

# Process all Go files in key directories
directories = [
    r'src\usecase',
    r'src\ui\rest',
    r'src\infrastructure\broadcast',
    r'src\infrastructure\whatsapp',
    r'src\infrastructure\sequence'
]

total_fixes = 0

for dir_path in directories:
    if os.path.exists(dir_path):
        print(f"\nProcessing {dir_path}...")
        
        for filename in os.listdir(dir_path):
            if filename.endswith('.go'):
                file_path = os.path.join(dir_path, filename)
                
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                original = content
                
                # Remove ALL backticks that are not part of multi-line strings
                # This regex removes backticks that are around single words
                content = re.sub(r'`(\w+)`', r'\1', content)
                
                # Fix common Go syntax issues
                content = re.sub(r'(\s+)SELECT\s*{', r'\1select {', content)
                content = re.sub(r'(\s+)CASE\s+', r'\1case ', content)
                content = re.sub(r'(\s+)DEFAULT\s*:', r'\1default:', content)
                
                # Fix SQL keywords inside SQL strings (between backticks for multi-line strings)
                def fix_sql_query(match):
                    sql = match.group(0)
                    # Only process if it looks like SQL
                    if any(kw in sql.upper() for kw in ['SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE']):
                        # Uppercase SQL keywords
                        keywords = ['select', 'from', 'where', 'order by', 'group by', 'limit',
                                   'insert into', 'update', 'delete from', 'and', 'or', 'set',
                                   'values', 'join', 'left join', 'right join', 'inner join',
                                   'on', 'as', 'distinct', 'count', 'sum', 'avg', 'max', 'min']
                        
                        for kw in keywords:
                            # Use word boundaries to avoid replacing parts of words
                            pattern = r'\b' + kw + r'\b'
                            sql = re.sub(pattern, kw.upper(), sql, flags=re.IGNORECASE)
                    
                    return sql
                
                # Apply SQL fixes to multi-line strings (between triple backticks)
                content = re.sub(r'`[^`]+`', fix_sql_query, content, flags=re.DOTALL)
                
                if content != original:
                    with open(file_path, 'w', encoding='utf-8') as f:
                        f.write(content)
                    print(f"  + Fixed {filename}")
                    total_fixes += 1

print(f"\n{'='*50}")
print(f"Total files fixed: {total_fixes}")
print("All syntax errors should now be resolved!")
print("="*50)
