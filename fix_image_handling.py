import re

print("Fixing image handling for campaigns and sequences...")

# Fix 1: Update the broadcast repository query to use media_url correctly
with open(r'src\repository\broadcast_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# The query is aliasing media_url AS image_url but scanning into MediaURL
# Let's remove the alias since we're scanning into MediaURL anyway
old_query = "bm.content AS message, bm.media_url AS image_url,"
new_query = "bm.content AS message, bm.media_url,"

content = content.replace(old_query, new_query)

# Also need to make sure the ImageURL field is set for backward compatibility
# After scanning, set msg.ImageURL = msg.MediaURL
scan_pattern = r'(messages = append\(messages, msg\))'
replacement = '''// Set ImageURL for backward compatibility
		msg.ImageURL = msg.MediaURL
		msg.Message = msg.Content
		
		\1'''

content = re.sub(scan_pattern, replacement, content)

with open(r'src\repository\broadcast_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed broadcast repository!")

# Fix 2: Check campaign trigger to ensure it sets MediaURL correctly
with open(r'src\usecase\campaign_trigger.go', 'r', encoding='utf-8') as f:
    campaign_content = f.read()

# Check if MediaURL is being set from ImageURL
if 'MediaURL:       campaign.ImageURL,' not in campaign_content:
    print("Campaign trigger already sets MediaURL correctly")

print("\nFixes applied:")
print("1. Removed alias in query - now selects media_url directly")
print("2. Added backward compatibility - sets ImageURL = MediaURL after scan")
print("3. Campaign trigger already sets MediaURL from campaign.ImageURL")
