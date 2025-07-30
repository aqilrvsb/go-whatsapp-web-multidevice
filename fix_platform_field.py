import re

print("Fixing CreateLead to set Platform field...")

# Read the file
with open(r'src\ui\rest\app.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Find and fix the CreateLead function
old_lead_creation = """lead := &models.Lead{
		UserID:       session.UserID,
		DeviceID:     request.DeviceID,
		Name:         request.Name,
		Phone:        request.Phone,
		Email:        "",
		Niche:        request.Niche,
		Source:       "manual", // Set source as manual since it's added from UI
		Status:       "", // Keep empty for backward compatibility
		TargetStatus: request.TargetStatus, // Use TargetStatus directly from request
		Trigger:      request.Trigger, // Add trigger
		Notes:        request.Journey, // Map journey to notes field
	}"""

new_lead_creation = """lead := &models.Lead{
		UserID:       session.UserID,
		DeviceID:     request.DeviceID,
		Name:         request.Name,
		Phone:        request.Phone,
		Email:        "",
		Niche:        request.Niche,
		Source:       "manual", // Set source as manual since it's added from UI
		Status:       "", // Keep empty for backward compatibility
		TargetStatus: request.TargetStatus, // Use TargetStatus directly from request
		Trigger:      request.Trigger, // Add trigger
		Notes:        request.Journey, // Map journey to notes field
		Platform:     "", // Add platform field (empty for manual leads)
	}"""

content = content.replace(old_lead_creation, new_lead_creation)

# Save the file
with open(r'src\ui\rest\app.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed CreateLead to set Platform field!")
