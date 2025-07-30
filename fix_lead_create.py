import re

print("Fixing lead repository for MySQL compatibility...")

# Read the file
with open(r'src\repository\lead_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix 1: Replace QueryRow with Exec for INSERT and use LastInsertId
old_create_lead = '''	var id int
	err := r.db.QueryRow(query, lead.DeviceID, lead.UserID, lead.Name, lead.Phone, 
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.Platform, lead.CreatedAt, lead.UpdatedAt).Scan(&id)
	
	if err == nil {
		lead.ID = fmt.Sprintf("%d", id)
	}
		
	return err'''

new_create_lead = '''	result, err := r.db.Exec(query, lead.DeviceID, lead.UserID, lead.Name, lead.Phone, 
		lead.Niche, journey, status, lead.TargetStatus, lead.Trigger, lead.Platform, lead.CreatedAt, lead.UpdatedAt)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	lead.ID = fmt.Sprintf("%d", id)
	return nil'''

content = content.replace(old_create_lead, new_create_lead)

# Fix 2: Ensure all column references have proper escaping
# The trigger column should already be escaped with backticks

# Save the file
with open(r'src\repository\lead_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed lead repository for MySQL!")

# Now let's also check the lead model to ensure it has all required fields
print("\nChecking lead model structure...")
