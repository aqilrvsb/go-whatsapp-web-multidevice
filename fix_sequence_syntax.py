import re

# Read the file
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence.go', 'r') as f:
    content = f.read()

# Find where we added the methods and ensure proper closure
# First, let's find the last proper function before our additions
if content.count('{') != content.count('}'):
    print(f"Brace mismatch: { content.count('{') } open, { content.count('}') } close")
    
    # Find the position before our delete methods
    delete_pos = content.find('// DeleteSequenceContactsByStatus')
    if delete_pos > 0:
        # Check if there's a missing closing brace before our methods
        before_delete = content[:delete_pos]
        
        # Count braces before our addition
        open_before = before_delete.count('{')
        close_before = before_delete.count('}')
        
        if open_before > close_before:
            # Add missing closing brace
            content = before_delete + '}\n\n' + content[delete_pos:]
            
            with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence.go', 'w') as f:
                f.write(content)
            
            print("Fixed missing closing brace")
        else:
            print("Brace count looks correct before delete methods")

print("Sequence.go syntax fixed")
