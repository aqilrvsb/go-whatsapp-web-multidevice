import os
import re

print("Fixing all SQL syntax in repository files...")

repo_dir = r'src\repository'

for filename in os.listdir(repo_dir):
    if filename.endswith('.go'):
        filepath = os.path.join(repo_dir, filename)
        print(f"Processing {filename}...")
        
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Fix pattern 1: Multi-line SQL without query :=
        # Look for indent + backtick + newline + indent + SQL keyword
        def fix_multiline_sql(match):
            indent = match.group(1)
            sql_keyword = match.group(2)
            # Return with query := added
            return f'{indent}query := `\n{indent}\t{sql_keyword}'
        
        pattern1 = r'(\n\t+)`\n\s*(SELECT|INSERT|UPDATE|DELETE)'
        content = re.sub(pattern1, fix_multiline_sql, content)
        
        # Fix pattern 2: Single-line SQL without query :=
        pattern2 = r'(\n\t+)`(SELECT|INSERT|UPDATE|DELETE)'
        content = re.sub(pattern2, r'\1query := `\2', content)
        
        # Special case: fix lines that just have a backtick followed by SQL
        # This happens when the SQL is on the same line
        pattern3 = r'(\n\t+)(\s*)`(\s*)(SELECT|INSERT|UPDATE|DELETE)'
        content = re.sub(pattern3, r'\1query := `\3\4', content)
        
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"  - Fixed SQL syntax in {filename}")
        else:
            print(f"  - No changes needed in {filename}")

print("\nAll files processed!")
