import re

def check_javascript_syntax(filename):
    with open(filename, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Extract all script blocks
    script_blocks = re.findall(r'<script[^>]*>(.*?)</script>', content, re.DOTALL)
    
    total_open = 0
    total_close = 0
    
    print(f"Found {len(script_blocks)} script blocks\n")
    
    for i, script in enumerate(script_blocks):
        open_braces = script.count('{')
        close_braces = script.count('}')
        open_parens = script.count('(')
        close_parens = script.count(')')
        
        total_open += open_braces
        total_close += close_braces
        
        print(f"Script block {i+1}:")
        print(f"  Braces: {open_braces} open, {close_braces} close (diff: {open_braces - close_braces})")
        print(f"  Parentheses: {open_parens} open, {close_parens} close (diff: {open_parens - close_parens})")
        
        if open_braces != close_braces:
            print(f"  WARNING: UNBALANCED BRACES!")
            # Try to find where the issue might be
            lines = script.split('\n')
            brace_count = 0
            for line_no, line in enumerate(lines):
                brace_count += line.count('{') - line.count('}')
                if '{' in line or '}' in line:
                    print(f"    Line ~{line_no}: {line.strip()[:80]}... (running count: {brace_count})")
    
    print(f"\nTotal braces across all scripts: {total_open} open, {total_close} close")
    print(f"DIFFERENCE: {total_open - total_close}")
    
    # Check for specific syntax errors
    print("\n--- Checking for common syntax errors ---")
    
    # Check each script block for issues
    for i, script in enumerate(script_blocks):
        lines = script.split('\n')
        for j, line in enumerate(lines):
            # Check for function declarations without parentheses
            if re.search(r'function\s+\w+\s*{', line):
                print(f"Script {i+1}, Line ~{j}: Missing () in function: {line.strip()}")
            
            # Check for duplicate parameters
            if re.search(r'\)\s*{\s*\(.*\)\s*{', line):
                print(f"Script {i+1}, Line ~{j}: Duplicate parameters: {line.strip()}")
            
            # Check for extra });
            if line.strip() == '});' and j+1 < len(lines) and lines[j+1].strip() == '});':
                print(f"Script {i+1}, Line ~{j}: Possible extra closing brace")

if __name__ == "__main__":
    check_javascript_syntax('src/views/public_device.html')
