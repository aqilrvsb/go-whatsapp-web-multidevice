import re

print("Fixing all missing 'query :=' declarations in campaign_repository.go...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Pattern to find SQL strings that are not assigned to a variable
# Look for backtick at the start of a line (with possible indentation) not preceded by :=
pattern = r'(\n\s+)`\n\s*(UPDATE|INSERT|SELECT|DELETE)'

# Replace with query := `
def replacer(match):
    return match.group(1) + 'query := `\n' + match.group(1)[1:] + match.group(2)

content = re.sub(pattern, replacer, content)

# Also fix cases where the backtick is on the same line as the SQL
pattern2 = r'(\n\s+)`(UPDATE|INSERT|SELECT|DELETE)'
content = re.sub(pattern2, r'\1query := `\2', content)

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed all missing query declarations!")

# Let's check the file for any remaining issues
print("\nChecking for potential remaining issues...")
lines = content.split('\n')
for i, line in enumerate(lines):
    # Check for backticks at the start of a line without assignment
    if line.strip().startswith('`') and not 'query :=' in lines[i-1] and not '=' in line:
        print(f"Potential issue at line {i+1}: {line.strip()[:50]}...")
