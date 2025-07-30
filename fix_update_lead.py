import re

print("Fixing UpdateLead parameter order in lead repository...")

# Read the file
with open(r'src\repository\lead_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix the parameter order in UpdateLead
old_exec = """result, err := r.db.Exec(query, id, lead.DeviceID, lead.Name, lead.Phone,
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.UpdatedAt)"""

new_exec = """result, err := r.db.Exec(query, lead.DeviceID, lead.Name, lead.Phone,
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.UpdatedAt, id)"""

content = content.replace(old_exec, new_exec)

# Save the file
with open(r'src\repository\lead_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed UpdateLead parameter order!")
