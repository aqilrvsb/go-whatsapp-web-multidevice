import os
import re

print("Final targeted fix for specific syntax errors...")

fixes = [
    {
        'file': 'multidevice_auto_reconnect.go',
        'line': 28,  # Line 29 in error is index 28
        'fix': lambda line: line.replace('`ORDER`', 'ORDER')
    },
    {
        'file': 'startup_loader.go',
        'line': 29,
        'fix': lambda line: line.replace('`order`', 'ORDER').replace('`ORDER`', 'ORDER')
    },
    {
        'file': 'whatsapp_web_personal.go',
        'line': 63,
        'fix': lambda line: line.replace('`order`', 'ORDER').replace('`ORDER`', 'ORDER')
    },
    {
        'file': 'whatsapp_web_personal.go',
        'line': 194,
        'fix': lambda line: line.replace('`order`', 'ORDER').replace('`ORDER`', 'ORDER')
    },
    {
        'file': 'worker_client_manager.go',
        'line': 139,
        'fix': lambda line: line.replace('`limit`', 'LIMIT').replace('`LIMIT`', 'LIMIT')
    }
]

base_dir = r'src\infrastructure\whatsapp'

for fix_info in fixes:
    file_path = os.path.join(base_dir, fix_info['file'])
    if os.path.exists(file_path):
        print(f"\nFixing {fix_info['file']} line {fix_info['line'] + 1}...")
        
        with open(file_path, 'r', encoding='utf-8') as f:
            lines = f.readlines()
        
        if len(lines) > fix_info['line']:
            original_line = lines[fix_info['line']]
            fixed_line = fix_info['fix'](original_line)
            
            if original_line != fixed_line:
                lines[fix_info['line']] = fixed_line
                print(f"  Original: {original_line.strip()}")
                print(f"  Fixed:    {fixed_line.strip()}")
                
                with open(file_path, 'w', encoding='utf-8') as f:
                    f.writelines(lines)
            else:
                print(f"  No change needed")

# Also do a general sweep to remove any remaining backticks around SQL keywords
print("\nGeneral cleanup of SQL keywords...")

sql_keywords = ['SELECT', 'FROM', 'WHERE', 'ORDER', 'GROUP', 'LIMIT', 'INSERT', 'UPDATE', 'DELETE', 
                'AND', 'OR', 'BY', 'ASC', 'DESC', 'INTO', 'VALUES', 'SET']

for filename in os.listdir(base_dir):
    if filename.endswith('.go'):
        file_path = os.path.join(base_dir, filename)
        
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original = content
        
        # Remove backticks around SQL keywords
        for keyword in sql_keywords:
            content = content.replace(f'`{keyword}`', keyword)
            content = content.replace(f'`{keyword.lower()}`', keyword)
        
        if content != original:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"  + Cleaned SQL keywords in {filename}")

print("\nAll targeted fixes complete!")
