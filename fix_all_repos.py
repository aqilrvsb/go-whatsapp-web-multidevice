import os 
import re 
 
repo_dir = r'src\repository' 
 
for filename in os.listdir(repo_dir): 
    if filename.endswith('.go'): 
        filepath = os.path.join(repo_dir, filename) 
        print(f"Fixing {filename}...") 
ECHO is off.
        with open(filepath, 'r', encoding='utf-8') as f: 
            content = f.read() 
ECHO is off.
        # Fix missing query := declarations 
        # Pattern: backtick at start of line (with indentation) followed by SQL 
