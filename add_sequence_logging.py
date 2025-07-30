import re

print("Adding detailed logging for sequence step creation...")

# Add logging to the service layer
with open(r'src\usecase\sequence.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Find the CreateSequence function and add more logging
old_step_creation = '''step := &models.SequenceStep{
			SequenceID:        sequence.ID,
			DayNumber:         stepReq.DayNumber,
			Trigger:           stepReq.Trigger,
			NextTrigger:       stepReq.NextTrigger,
			TriggerDelayHours: stepReq.TriggerDelayHours,
			IsEntryPoint:      stepReq.IsEntryPoint,
			MessageType:       stepReq.MessageType,
			Content:           stepReq.Content,
			MediaURL:          stepReq.MediaURL,
			Caption:           stepReq.Caption,
			TimeSchedule:      stepReq.TimeSchedule,
			MinDelaySeconds:   stepReq.MinDelaySeconds,
			MaxDelaySeconds:   stepReq.MaxDelaySeconds,
		}'''

new_step_creation = '''// Log incoming step request
		logrus.Infof("Processing step %d - Type: %s, Content: %s, MediaURL: %s, ImageURL: %s",
			i+1, stepReq.MessageType, stepReq.Content, stepReq.MediaURL, stepReq.ImageURL)
		
		step := &models.SequenceStep{
			SequenceID:        sequence.ID,
			DayNumber:         stepReq.DayNumber,
			Trigger:           stepReq.Trigger,
			NextTrigger:       stepReq.NextTrigger,
			TriggerDelayHours: stepReq.TriggerDelayHours,
			IsEntryPoint:      stepReq.IsEntryPoint,
			MessageType:       stepReq.MessageType,
			Content:           stepReq.Content,
			MediaURL:          stepReq.MediaURL,
			Caption:           stepReq.Caption,
			TimeSchedule:      stepReq.TimeSchedule,
			MinDelaySeconds:   stepReq.MinDelaySeconds,
			MaxDelaySeconds:   stepReq.MaxDelaySeconds,
		}'''

content = content.replace(old_step_creation, new_step_creation)

with open(r'src\usecase\sequence.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Added logging to service layer")

# Add logging to repository to see exact SQL error
with open(r'src\repository\sequence_repository.go', 'r', encoding='utf-8') as f:
    repo_content = f.read()

# Enhance error logging in CreateSequenceStep
old_error_log = '''if err != nil {
		logrus.Errorf("Failed to create sequence step: %v", err)
	}'''

new_error_log = '''if err != nil {
		logrus.Errorf("Failed to create sequence step: %v", err)
		logrus.Errorf("Step details - ID: %s, SequenceID: %s, DayNumber: %d, MessageType: %s", 
			step.ID, step.SequenceID, dayNumber, step.MessageType)
		logrus.Errorf("Content length: %d, MediaURL: %s, Caption: %s", 
			len(step.Content), step.MediaURL, step.Caption)
	}'''

repo_content = repo_content.replace(old_error_log, new_error_log)

with open(r'src\repository\sequence_repository.go', 'w', encoding='utf-8') as f:
    f.write(repo_content)

print("Enhanced error logging in repository")
