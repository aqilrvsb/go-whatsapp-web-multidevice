import re

# Update interface
interface_file = r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\domains\sequence\sequence.go'
with open(interface_file, 'r') as f:
    content = f.read()

# Find ProcessSequences() and add after it
if 'DeleteSequenceContactsByStatus' not in content:
    content = content.replace(
        'ProcessSequences() error // Called by cron job',
        '''ProcessSequences() error // Called by cron job
	DeleteSequenceContactsByStatus(sequenceID string, status string) (int64, error)
	DeleteAllSequenceContacts(sequenceID string) (int64, error)'''
    )
    
    with open(interface_file, 'w') as f:
        f.write(content)
    print("Updated interface")

# Add repository methods
repo_file = r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\repository\broadcast_repository.go'
with open(repo_file, 'r') as f:
    content = f.read()

if 'DeleteBySequenceAndStatus' not in content:
    # Add at the end of file before last }
    methods = '''

// DeleteBySequenceAndStatus deletes messages by sequence and status
func (r *BroadcastRepository) DeleteBySequenceAndStatus(sequenceID, status string) (int64, error) {
	result, err := r.db.Exec(`
		DELETE FROM broadcast_messages 
		WHERE sequence_id = $1 AND status = $2
	`, sequenceID, status)
	
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}

// DeleteBySequence deletes all messages for a sequence
func (r *BroadcastRepository) DeleteBySequence(sequenceID string) (int64, error) {
	result, err := r.db.Exec(`
		DELETE FROM broadcast_messages 
		WHERE sequence_id = $1
	`, sequenceID)
	
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}'''
    
    last_brace = content.rfind('}')
    content = content[:last_brace] + methods + '\n' + content[last_brace:]
    
    with open(repo_file, 'w') as f:
        f.write(content)
    print("Added repository methods")

print("Done!")
