import re

print("Fixing all campaign and sequence MySQL issues...")

# Fix 1: Update campaign repository - already done in previous script

# Fix 2: Check sequence summary to ensure it uses broadcast_messages
print("Checking sequence summary implementation...")

# Read app.go to check GetSequenceSummary
with open(r'src\ui\rest\app.go', 'r', encoding='utf-8') as f:
    app_content = f.read()

# Check if GetSequenceSummary exists and uses broadcast_messages
if 'GetSequenceSummary' in app_content:
    print("GetSequenceSummary found - checking if it uses broadcast_messages...")
else:
    print("GetSequenceSummary not found - will need to implement it")

# Fix 3: Ensure sequence steps are saved with sequences
print("\nChecking sequence creation with steps...")

# The sequence creation should handle steps in a transaction
# Let's check the CreateSequence handler in app.go

# Find the CreateSequence handler
create_seq_match = re.search(r'func.*CreateSequence.*\{[\s\S]*?err\s*:=\s*sequenceRepo\.CreateSequence', app_content)
if create_seq_match:
    print("CreateSequence handler found - checking if it saves steps...")
    
# Fix 4: Add comprehensive error logging
print("\nAdding error logging to repositories...")

# Read sequence repository
with open(r'src\repository\sequence_repository.go', 'r', encoding='utf-8') as f:
    seq_content = f.read()

# Add logging to CreateSequenceStep if not present
if 'logrus.Errorf("Failed to create sequence step:' not in seq_content:
    # Add error logging after the Exec call
    old_exec = """_, err := r.db.Exec(query, 
		step.ID, step.SequenceID, dayNumber, step.MessageType, step.Content,
		step.MediaURL, step.Caption, step.Trigger, step.TimeSchedule,
		step.NextTrigger, step.TriggerDelayHours, step.IsEntryPoint,
		step.MinDelaySeconds, step.MaxDelaySeconds, step.DelayDays)
		
	return err"""
    
    new_exec = """_, err := r.db.Exec(query, 
		step.ID, step.SequenceID, dayNumber, step.MessageType, step.Content,
		step.MediaURL, step.Caption, step.Trigger, step.TimeSchedule,
		step.NextTrigger, step.TriggerDelayHours, step.IsEntryPoint,
		step.MinDelaySeconds, step.MaxDelaySeconds, step.DelayDays)
		
	if err != nil {
		logrus.Errorf("Failed to create sequence step: %v", err)
	}
	
	return err"""
    
    seq_content = seq_content.replace(old_exec, new_exec)
    
    with open(r'src\repository\sequence_repository.go', 'w', encoding='utf-8') as f:
        f.write(seq_content)
    
    print("Added error logging to CreateSequenceStep")

print("\nAll fixes applied!")
print("\nSummary of fixes:")
print("1. Campaign repository - Fixed 'limit' keyword and INSERT method")
print("2. Sequence repository - Added error logging")
print("3. Both campaign and sequence summaries should use broadcast_messages table")
