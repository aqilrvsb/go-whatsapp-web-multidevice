import re

print("Ensuring Caption is set for image messages...")

with open(r'src\usecase\sequence.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Find where we create the step and ensure Caption is set for images
# When MessageType is "image", Caption should be set from Content if Caption is empty

# Add logic after step creation
old_step_log = '''// Log incoming step request
		logrus.Infof("Processing step %d - Type: %s, Content: %s, MediaURL: %s, ImageURL: %s",
			i+1, stepReq.MessageType, stepReq.Content, stepReq.MediaURL, stepReq.ImageURL)'''

new_step_log = '''// Log incoming step request
		logrus.Infof("Processing step %d - Type: %s, Content: %s, MediaURL: %s, ImageURL: %s",
			i+1, stepReq.MessageType, stepReq.Content, stepReq.MediaURL, stepReq.ImageURL)
		
		// For image messages, ensure Caption is set
		if stepReq.MessageType == "image" && stepReq.Caption == "" && stepReq.Content != "" {
			stepReq.Caption = stepReq.Content
			logrus.Infof("Setting Caption from Content for image message")
		}'''

content = content.replace(old_step_log, new_step_log)

with open(r'src\usecase\sequence.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("Added Caption handling for image messages")

# Also fix the frontend to send caption properly
print("\nChecking frontend...")

with open(r'src\views\sequences.html', 'r', encoding='utf-8') as f:
    html_content = f.read()

# Find the step creation in frontend and add caption
old_step_js = '''const step = {
                    day: index + 1,
                    day_number: index + 1,
                    message_type: stepEl.querySelector('.step-image-url').value ? 'image' : 'text',
                    content: stepEl.querySelector('.step-content').value,
                    image_url: stepEl.querySelector('.step-image-url').value || '',
                    media_url: stepEl.querySelector('.step-image-url').value || '',
                    min_delay_seconds: parseInt(stepEl.querySelector('.step-min-delay').value) || 5,
                    max_delay_seconds: parseInt(stepEl.querySelector('.step-max-delay').value) || 15,
                    send_time: document.getElementById('sequenceTimeSchedule').value || '09:00',
                    time_schedule: document.getElementById('sequenceTimeSchedule').value || '09:00'
                };'''

new_step_js = '''const step = {
                    day: index + 1,
                    day_number: index + 1,
                    message_type: stepEl.querySelector('.step-image-url').value ? 'image' : 'text',
                    content: stepEl.querySelector('.step-content').value,
                    image_url: stepEl.querySelector('.step-image-url').value || '',
                    media_url: stepEl.querySelector('.step-image-url').value || '',
                    caption: stepEl.querySelector('.step-content').value || '', // Add caption for images
                    min_delay_seconds: parseInt(stepEl.querySelector('.step-min-delay').value) || 5,
                    max_delay_seconds: parseInt(stepEl.querySelector('.step-max-delay').value) || 15,
                    send_time: document.getElementById('sequenceTimeSchedule').value || '09:00',
                    time_schedule: document.getElementById('sequenceTimeSchedule').value || '09:00'
                };'''

html_content = html_content.replace(old_step_js, new_step_js)

with open(r'src\views\sequences.html', 'w', encoding='utf-8') as f:
    f.write(html_content)

print("Updated frontend to send caption field")
print("\nAll fixes applied!")
