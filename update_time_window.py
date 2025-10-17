import os
import re

print("="*60)
print("WhatsApp Broadcast System - Update Time Window to 3 Hours")
print("="*60)

# File to update
file_path = r"C:\Users\aqilz\go-whatsapp-web-multidevice-main\src\repository\broadcast_repository.go"

# Read the file
print(f"\n1. Reading file: {file_path}")
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Count occurrences of INTERVAL 1 HOUR
count_1_hour = content.count("INTERVAL 1 HOUR")
print(f"   Found {count_1_hour} occurrences of 'INTERVAL 1 HOUR'")

# Find and show the specific line we need to change (around line 165)
lines = content.split('\n')
changes_made = []

print("\n2. Analyzing occurrences:")
for i, line in enumerate(lines, 1):
    if "INTERVAL 1 HOUR" in line:
        # Show context
        context_start = max(0, i-3)
        context_end = min(len(lines), i+2)
        print(f"\n   Occurrence at line {i}:")
        for j in range(context_start, context_end):
            if j == i-1:  # Current line (0-indexed)
                print(f"   >>> {j+1}: {lines[j]}")
            else:
                print(f"       {j+1}: {lines[j]}")
        
        # Check if this is the time window restriction we want to change
        # It should be in a WHERE clause with DATE_SUB
        if "DATE_SUB" in line and "scheduled_at" in lines[i-2]:
            print(f"   ✓ This is the time window restriction to update!")
            changes_made.append(i)
            lines[i-1] = line.replace("INTERVAL 1 HOUR", "INTERVAL 3 HOUR")
        else:
            print(f"   - Skipping this occurrence")

# Save the updated file
if changes_made:
    print(f"\n3. Updating line(s): {changes_made}")
    print("   Changing: INTERVAL 1 HOUR → INTERVAL 3 HOUR")
    
    # Create backup
    backup_path = file_path + ".backup"
    with open(backup_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"\n4. Created backup: {backup_path}")
    
    # Write updated content
    updated_content = '\n'.join(lines)
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(updated_content)
    print(f"\n5. Updated file: {file_path}")
    
    print("\n" + "="*60)
    print("✅ SUCCESS: Time window updated from 1 HOUR to 3 HOURS")
    print("="*60)
    print("\nNOTE: You need to rebuild and restart the application for changes to take effect:")
    print("  1. cd C:\\Users\\aqilz\\go-whatsapp-web-multidevice-main\\src")
    print("  2. go build -o ../whatsapp.exe")
    print("  3. Restart the whatsapp.exe application")
else:
    print("\n⚠️ No changes made - could not find the time window restriction line")
    print("Please check the file manually.")
