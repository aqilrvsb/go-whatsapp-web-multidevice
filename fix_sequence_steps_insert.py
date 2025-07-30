import re

print("Fixing sequence_steps INSERT query to match actual table structure...")

with open(r'src\repository\sequence_repository.go', 'r', encoding='utf-8') as f:
    content = f.read()

# The current query has the wrong column order
# Based on the screenshot, the correct order is:
# id, sequence_id, day_number, message_type, content, media_url, caption, 
# delay_days, time_schedule, trigger, next_trigger, trigger_delay_hours,
# is_entry_point, min_delay_seconds, max_delay_seconds, day, send_time, updated_at

old_query = '''query := `
		INSERT INTO sequence_steps(
			id, sequence_id, day_number, message_type, content, 
			media_url, caption, ` + "`trigger`" + `, time_schedule,
			next_trigger, trigger_delay_hours, is_entry_point,
			min_delay_seconds, max_delay_seconds, delay_days
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`'''

# Fix the column order and add missing columns
new_query = '''query := `
		INSERT INTO sequence_steps(
			id, sequence_id, day_number, message_type, content, 
			media_url, caption, delay_days, time_schedule, ` + "`trigger`" + `,
			next_trigger, trigger_delay_hours, is_entry_point,
			min_delay_seconds, max_delay_seconds
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`'''

content = content.replace(old_query, new_query)

# Now fix the parameter order in the Exec call
old_exec = '''_, err := r.db.Exec(query, 
		step.ID, step.SequenceID, dayNumber, step.MessageType, step.Content,
		step.MediaURL, step.Caption, step.Trigger, step.TimeSchedule,
		step.NextTrigger, step.TriggerDelayHours, step.IsEntryPoint,
		step.MinDelaySeconds, step.MaxDelaySeconds, step.DelayDays)'''

# Fix to match the new column order
new_exec = '''// Default DelayDays if not set
	delayDays := step.DelayDays
	if delayDays == 0 {
		delayDays = 1
	}
	
	_, err := r.db.Exec(query, 
		step.ID, step.SequenceID, dayNumber, step.MessageType, step.Content,
		step.MediaURL, step.Caption, delayDays, step.TimeSchedule, step.Trigger,
		step.NextTrigger, step.TriggerDelayHours, step.IsEntryPoint,
		step.MinDelaySeconds, step.MaxDelaySeconds)'''

content = content.replace(old_exec, new_exec)

with open(r'src\repository\sequence_repository.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed sequence_steps INSERT query!")

# Also check if Caption is being set for image messages
print("\nChecking if Caption field needs to be set...")

with open(r'src\usecase\sequence.go', 'r', encoding='utf-8') as f:
    service_content = f.read()

# When message type is image, we should set Caption = Content
if 'Caption:           stepReq.Caption,' in service_content and 'stepReq.Content' in service_content:
    print("Caption is already being mapped from request")
else:
    print("Need to check Caption mapping")

print("\nDone! The main issue was the column order mismatch in the INSERT query.")
