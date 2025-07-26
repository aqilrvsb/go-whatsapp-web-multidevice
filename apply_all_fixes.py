import re
import os

project_dir = r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main'

# 1. Fix UUID issue in sequence_trigger_processor.go
print("Fixing UUID issue...")
trigger_file = os.path.join(project_dir, 'src', 'usecase', 'sequence_trigger_processor.go')

with open(trigger_file, 'r') as f:
    content = f.read()

# Add uuid import if not present
if '"github.com/google/uuid"' not in content:
    import_section = re.search(r'import \((.*?)\)', content, re.DOTALL)
    if import_section:
        imports = import_section.group(1).rstrip()
        new_imports = imports + '\n\t"github.com/google/uuid"'
        content = content.replace(imports, new_imports)
        print("Added UUID import")

# Fix the INSERT query to include ID with proper UUID generation
old_pattern = r'(// Insert all messages into broadcast_messages.*?)(\n\s+)(_, err = tx\.Exec\(insertQuery,\s*\n\s+msg\.UserID.*?msg\.ScheduledAt\))'
new_insert = r'\1\2// Generate UUID for message ID\n\t\tmessageID := uuid.New().String()\n\t\t\n\t\t// Handle potential nil values for media_url\n\t\tvar mediaURL interface{} = nil\n\t\tif msg.MediaURL != "" {\n\t\t\tmediaURL = msg.MediaURL\n\t\t}\n\t\t\n\t\t_, err = tx.Exec(insertQuery,\n\t\t\tmessageID, msg.UserID, msg.DeviceID, msg.SequenceID, msg.SequenceStepID,\n\t\t\tmsg.RecipientPhone, msg.RecipientName, msg.Type,\n\t\t\tmsg.Content, mediaURL, msg.Status, msg.ScheduledAt)'

# Update the INSERT statement to include ID
content = re.sub(
    r'INSERT INTO broadcast_messages \(\s*user_id, device_id',
    'INSERT INTO broadcast_messages (\n\t\t\t\tid, user_id, device_id',
    content
)

content = re.sub(
    r'VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11\)',
    'VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)',
    content
)

# Apply the fix for exec call
content = re.sub(old_pattern, new_insert, content, flags=re.DOTALL)

with open(trigger_file, 'w') as f:
    f.write(content)

print("Fixed UUID issue in sequence_trigger_processor.go")

# 2. Add delete methods to sequence.go WITHOUT breaking it
print("\nAdding delete methods to sequence.go...")
sequence_file = os.path.join(project_dir, 'src', 'usecase', 'sequence.go')

with open(sequence_file, 'r') as f:
    seq_content = f.read()

# Only add if not already present
if 'DeleteSequenceContactsByStatus' not in seq_content:
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
	
	// Use the repository method
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
	
	// Use the repository method
	rowsAffected, err := broadcastRepo.DeleteBySequence(sequenceID)
	if err != nil {
		logrus.Errorf("Failed to delete all sequence contacts: %v", err)
		return 0, fmt.Errorf("failed to delete all sequence contacts: %w", err)
	}
	
	logrus.Infof("Deleted %d broadcast messages for sequence %s", 
		rowsAffected, sequenceID)
	
	return rowsAffected, nil
}'''
    
    # Add before the final closing brace of the file
    # Find the last closing brace
    last_brace_pos = seq_content.rfind('}')
    if last_brace_pos > 0:
        seq_content = seq_content[:last_brace_pos] + delete_methods + '\n' + seq_content[last_brace_pos:]
    
    with open(sequence_file, 'w') as f:
        f.write(seq_content)
    
    print("Added delete methods to sequence.go")

# 3. Update the sequence interface
print("\nUpdating sequence interface...")
interface_file = os.path.join(project_dir, 'src', 'domains', 'sequence', 'sequence.go')

with open(interface_file, 'r') as f:
    interface_content = f.read()

if 'DeleteSequenceContactsByStatus' not in interface_content:
    # Find ProcessSequences() and add after it
    process_pos = interface_content.find('ProcessSequences() error')
    if process_pos > 0:
        # Find the end of this line
        newline_pos = interface_content.find('\n', process_pos)
        if newline_pos > 0:
            insert_text = '\n\tDeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error)\n\tDeleteAllSequenceContacts(sequenceID string) (int64, error)'
            interface_content = interface_content[:newline_pos] + insert_text + interface_content[newline_pos:]
            
            with open(interface_file, 'w') as f:
                f.write(interface_content)
            
            print("Updated sequence interface")

print("\nAll fixes applied successfully!")
