import re

print("Adding debug logging to CreateLead...")

# Read the file
with open(r'src\repository\lead_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for log if not present
if 'import (' in content and '"log"' not in content:
    content = content.replace('import (', 'import (\n\t"log"')

# Add debug logging before the query execution
old_exec = """result, err := r.db.Exec(query, lead.DeviceID, lead.UserID, lead.Name, lead.Phone, 
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.Platform, lead.CreatedAt, lead.UpdatedAt)"""

new_exec = """// Debug logging
	log.Printf("CreateLead - DeviceID: %s, UserID: %s, Name: %s, Phone: %s", lead.DeviceID, lead.UserID, lead.Name, lead.Phone)
	log.Printf("CreateLead - Niche: %s, Status: %s, TargetStatus: %s, Platform: %s", lead.Niche, status, lead.TargetStatus, lead.Platform)
	log.Printf("CreateLead - Journey: %s, Trigger: %s", journey, lead.Trigger)
	
	result, err := r.db.Exec(query, lead.DeviceID, lead.UserID, lead.Name, lead.Phone, 
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.Platform, lead.CreatedAt, lead.UpdatedAt)"""

content = content.replace(old_exec, new_exec)

# Also add error logging
old_error = """if err != nil {
		return err
	}"""

new_error = """if err != nil {
		log.Printf("CreateLead - Error executing query: %v", err)
		return err
	}"""

# Replace only the first occurrence after the Exec
content = content.replace(old_error, new_error, 1)

# Save the file
with open(r'src\repository\lead_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Added debug logging to CreateLead!")
