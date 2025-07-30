print("Manually fixing broadcast repository...")

with open(r'src\repository\broadcast_repository.go', 'r', encoding='utf-8', errors='replace') as f:
    content = f.read()

# Remove all control characters
import string
printable = set(string.printable)
content = ''.join(filter(lambda x: x in printable, content))

# Now find and fix the append statements
lines = content.split('\n')
new_lines = []
i = 0
while i < len(lines):
    line = lines[i]
    if 'messages = append(messages, msg)' in line and i > 0:
        # Check if the previous lines already have the compatibility code
        if i >= 4 and 'Set ImageURL for backward compatibility' in lines[i-4]:
            # Skip the duplicate
            new_lines.append(line)
        else:
            # Add the compatibility code before append
            new_lines.append('\t\t// Set ImageURL for backward compatibility')
            new_lines.append('\t\tmsg.ImageURL = msg.MediaURL')
            new_lines.append('\t\tmsg.Message = msg.Content')
            new_lines.append('\t\t')
            new_lines.append(line)
    else:
        new_lines.append(line)
    i += 1

content = '\n'.join(new_lines)

with open(r'src\repository\broadcast_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed broadcast repository!")
