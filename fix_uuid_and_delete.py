import re

# Read the sequence trigger processor file
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence_trigger_processor.go', 'r') as f:
    content = f.read()

# 1. Add uuid import if not present
if 'github.com/google/uuid' not in content:
    # Add uuid import after other imports
    import_section = re.search(r'import \((.*?)\)', content, re.DOTALL)
    if import_section:
        imports = import_section.group(1)
        new_imports = imports.rstrip() + '\n\t"github.com/google/uuid"\n'
        content = content.replace(imports, new_imports)

# 2. Fix the INSERT query to include ID
old_insert = '''	// Insert all messages into broadcast_messages
	for _, msg := range allMessages {
		insertQuery := `
			INSERT INTO broadcast_messages (
				user_id, device_id, sequence_id, sequence_stepid,
				recipient_phone, recipient_name, message_type,
				content, media_url, status, scheduled_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		
		_, err = tx.Exec(insertQuery,
			msg.UserID, msg.DeviceID, msg.SequenceID, msg.SequenceStepID,
			msg.RecipientPhone, msg.RecipientName, msg.Type,
			msg.Content, msg.MediaURL, msg.Status, msg.ScheduledAt)'''

new_insert = '''	// Insert all messages into broadcast_messages
	for _, msg := range allMessages {
		// Generate UUID for message ID
		messageID := uuid.New().String()
		
		insertQuery := `
			INSERT INTO broadcast_messages (
				id, user_id, device_id, sequence_id, sequence_stepid,
				recipient_phone, recipient_name, message_type,
				content, media_url, status, scheduled_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
		
		// Handle potential nil values for media_url
		var mediaURL interface{} = nil
		if msg.MediaURL != "" {
			mediaURL = msg.MediaURL
		}
		
		_, err = tx.Exec(insertQuery,
			messageID, msg.UserID, msg.DeviceID, msg.SequenceID, msg.SequenceStepID,
			msg.RecipientPhone, msg.RecipientName, msg.Type,
			msg.Content, mediaURL, msg.Status, msg.ScheduledAt)'''

content = content.replace(old_insert, new_insert)

# Write back
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence_trigger_processor.go', 'w') as f:
    f.write(content)

print("Fixed UUID issue in sequence_trigger_processor.go")

# Now add the delete methods to sequence.go
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence.go', 'r') as f:
    sequence_content = f.read()

# Check if methods already exist
if 'DeleteSequenceContactsByStatus' not in sequence_content:
    # Add at the end of the file, before the last closing brace
    delete_methods = '''
// DeleteSequenceContactsByStatus deletes broadcast messages for a sequence based on status
func (s *sequenceService) DeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error) {
	// Validate status
	validStatuses := map[string]bool{
		"pending": true,
		"sent":    true,
		"failed":  true,
	}
	
	if !validStatuses[status] {
		return 0, fmt.Errorf("invalid status: %s. Must be pending, sent, or failed", status)
	}
	
	// Get repository
	broadcastRepo := repository.GetBroadcastRepository()
	
	// Use the new repository method
	rowsAffected, err := broadcastRepo.DeleteBySequenceAndStatus(sequenceID, status)
	if err != nil {
		logrus.Errorf("Failed to delete sequence contacts: %v", err)
		return 0, fmt.Errorf("failed to delete sequence contacts: %w", err)
	}
	
	logrus.Infof("Deleted %d broadcast messages for sequence %s with status %s", 
		rowsAffected, sequenceID, status)
	
	return rowsAffected, nil
}

// DeleteAllSequenceContacts deletes all broadcast messages for a sequence
func (s *sequenceService) DeleteAllSequenceContacts(sequenceID string) (int64, error) {
	// Get repository
	broadcastRepo := repository.GetBroadcastRepository()
	
	// Use the new repository method
	rowsAffected, err := broadcastRepo.DeleteBySequence(sequenceID)
	if err != nil {
		logrus.Errorf("Failed to delete all sequence contacts: %v", err)
		return 0, fmt.Errorf("failed to delete all sequence contacts: %w", err)
	}
	
	logrus.Infof("Deleted %d broadcast messages for sequence %s", 
		rowsAffected, sequenceID)
	
	return rowsAffected, nil
}
'''
    
    # Insert before the last closing brace
    last_brace = sequence_content.rfind('}')
    sequence_content = sequence_content[:last_brace] + delete_methods + sequence_content[last_brace:]
    
    with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\usecase\sequence.go', 'w') as f:
        f.write(sequence_content)
    
    print("Added delete methods to sequence.go")

# Update the sequence interface
with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\domains\sequence\sequence.go', 'r') as f:
    interface_content = f.read()

if 'DeleteSequenceContactsByStatus' not in interface_content:
    # Find the ISequenceUsecase interface and add methods
    interface_pattern = r'(type ISequenceUsecase interface \{[^}]+)'
    match = re.search(interface_pattern, interface_content, re.DOTALL)
    if match:
        interface_body = match.group(1)
        new_interface = interface_body.rstrip() + '''
	DeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error)
	DeleteAllSequenceContacts(sequenceID string) (int64, error)
'''
        interface_content = interface_content.replace(interface_body, new_interface)
        
        with open(r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\domains\sequence\sequence.go', 'w') as f:
            f.write(interface_content)
        
        print("Updated sequence interface")

print("All fixes applied!")
