import re

# Read the rest.go file
file_path = r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\cmd\rest.go"
with open(file_path, 'r') as f:
    content = f.read()

# Find and comment out the campaign trigger section
pattern = r'(\t// Start campaign trigger processor.*?}\(\)\))'
replacement = r'''	// DISABLED: Campaign trigger processor - Now handled by sequence processor
	/*
\1
	*/'''

content = re.sub(pattern, replacement, content, flags=re.DOTALL)

# Update the sequence processor comment
content = content.replace(
    '// Start sequence trigger processor for trigger-based flow\n\tgo usecase.StartSequenceTriggerProcessor()\n\tlogrus.Info("Sequence trigger processor started")',
    '// Start sequence trigger processor - NOW HANDLES BOTH SEQUENCES AND CAMPAIGNS\n\tgo usecase.StartSequenceTriggerProcessor()\n\tlogrus.Info("Unified processor started (handles both sequences and campaigns)")'
)

# Write back
with open(file_path, 'w') as f:
    f.write(content)

print("Modified rest.go to disable campaign trigger and use unified processor")
