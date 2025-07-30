import re

print("Fixing regex replacement issue...")

with open(r'src\repository\broadcast_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Remove the invalid characters (likely from the regex replacement)
# First, let's see what's there
lines = content.split('\n')
for i, line in enumerate(lines):
    if '\x01' in line:
        print(f"Found invalid character at line {i+1}")
        # Replace the line with the correct code
        if 'messages = append(messages, msg)' in line:
            lines[i] = '\t\tmessages = append(messages, msg)'

# Rejoin the content
content = '\n'.join(lines)

# Now properly add the compatibility code
# Find where messages are appended and add the compatibility setting before it
pattern = r'(\t\tmessages = append\(messages, msg\))'
replacement = '''		// Set ImageURL for backward compatibility
		msg.ImageURL = msg.MediaURL
		msg.Message = msg.Content
		
\1'''

content = re.sub(pattern, replacement, content, count=1)

with open(r'src\repository\broadcast_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed broadcast repository!")
