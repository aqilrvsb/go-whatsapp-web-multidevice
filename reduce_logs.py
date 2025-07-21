import re

# Read the file
with open('src/usecase/sequence_trigger_processor.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Define replacements
replacements = [
    # Main sequence processing log
    (r'logrus\.Infof\("Sequence processing completed:', 'logrus.Debugf("Sequence processing completed:'),
    
    # Remove SEQUENCE-NAME debug log entirely (delete the line)
    (r'\s*// Debug logging for name issue\s*\n\s*logrus\.Infof\("\[SEQUENCE-NAME\] Contact:.*?\n', '\n'),
    
    # Step not ready log
    (r'logrus\.Infof\("⏰ Step %d for %s not ready', 'logrus.Debugf("⏰ Step %d for %s not ready'),
    
    # Enrolling contact log
    (r'logrus\.Infof\("Enrolling contact %s in sequence', 'logrus.Debugf("Enrolling contact %s in sequence'),
    
    # Step creation logs
    (r'logrus\.Infof\("Step \d+: PENDING', 'logrus.Debugf("Step %d: PENDING'),
    (r'logrus\.Infof\("\[ENROLLMENT\]', 'logrus.Debugf("[ENROLLMENT]'),
    
    # Performance metrics
    (r'logrus\.Infof\("Performance:', 'logrus.Debugf("Performance:'),
]

# Apply replacements
for old, new in replacements:
    content = re.sub(old, new, content, flags=re.MULTILINE | re.DOTALL)

# Write back
with open('src/usecase/sequence_trigger_processor.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("✅ Reduced sequence logs")

# Now handle webhook logs
with open('src/ui/rest/webhook_lead.go', 'r', encoding='utf-8') as f:
    webhook_content = f.read()

# Change most webhook Info logs to Debug
webhook_replacements = [
    (r'logrus\.Info\("Webhook received', 'logrus.Debug("Webhook received'),
    (r'logrus\.Info\("Webhook: Processing', 'logrus.Debug("Webhook: Processing'),
    (r'logrus\.Info\("Webhook: Creating', 'logrus.Debug("Webhook: Creating'),
    (r'logrus\.Info\("Webhook: Found', 'logrus.Debug("Webhook: Found'),
    (r'logrus\.Info\("Webhook: Updated', 'logrus.Debug("Webhook: Updated'),
    # Keep error logs as they are
]

for old, new in webhook_replacements:
    webhook_content = re.sub(old, new, webhook_content)

with open('src/ui/rest/webhook_lead.go', 'w', encoding='utf-8') as f:
    f.write(webhook_content)

print("✅ Reduced webhook logs")
