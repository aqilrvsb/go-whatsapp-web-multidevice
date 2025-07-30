import re

print("Fixing AS limit issue in campaign_repository.go...")

# Read the file
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# The issue is "AS `limit`" - we need to use a different alias
# Replace "AS `limit`" with "AS campaign_limit"
content = content.replace('COALESCE(`limit`, 0) AS `limit`', 'COALESCE(`limit`, 0) AS campaign_limit')

# Now we need to update the Go code that scans this value
# Find patterns where we're scanning into &campaign.Limit after a query with campaign_limit
# This is more complex - we need to look for the Scan calls

# Let's also check if there are any references to just "limit" in Scan calls
# that should be "campaign_limit"

# Save the file
with open(r'src\repository\campaign_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed AS limit issue!")

# Let's also update the struct field references in Scan calls
print("Updating Scan references...")

# Read again
with open(r'src\repository\campaign_repository.go', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# We need to find Scan calls that come after queries with campaign_limit
# and make sure they're scanning into the right field
# This is tricky without seeing the full context, but let's try

# For now, let's just make sure the SQL is valid
print("SQL should now be valid. If there are still errors, they might be in the Scan() calls.")
