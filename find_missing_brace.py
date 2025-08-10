import re

with open('public_device.html', 'r', encoding='utf-8') as f:
    content = f.read()

# Find clearAllSessions function
match = re.search(r'function clearAllSessions\(\) \{', content)
if match:
    start_pos = match.start()
    
    # Count braces from this position
    brace_count = 0
    i = start_pos
    in_string = False
    string_char = None
    
    while i < len(content):
        char = content[i]
        
        # Handle string literals
        if char in ['"', "'", '`'] and (i == 0 or content[i-1] != '\\'):
            if not in_string:
                in_string = True
                string_char = char
            elif char == string_char:
                in_string = False
                
        if not in_string:
            if char == '{':
                brace_count += 1
            elif char == '}':
                brace_count -= 1
                if brace_count == 0:
                    # Found the closing brace
                    line_num = content[:i].count('\n') + 1
                    print(f'clearAllSessions function ends at line {line_num}')
                    # Check what comes after
                    next_content = content[i:i+200].replace('\n', '\\n')
                    print(f'Content after closing brace: {next_content}')
                    break
        i += 1
    
    if brace_count > 0:
        print(f'Missing {brace_count} closing brace(s) for clearAllSessions')
        # Find where we are when content ends
        line_num = content.count('\n') + 1
        print(f'File ends at line {line_num} with {brace_count} unclosed braces')
