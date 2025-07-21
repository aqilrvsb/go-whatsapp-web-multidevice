import re
import os

base_path = r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main'

# Read the sequence file
seq_file = os.path.join(base_path, 'src', 'usecase', 'sequence_trigger_processor.go')
with open(seq_file, 'r', encoding='utf-8') as f:
    content = f.read()

# Apply replacements
print("Reducing sequence logs...")

# Main logs to reduce
content = re.sub(r'logrus\.Infof\("Sequence processing completed:', 'logrus.Debugf("Sequence processing completed:', content)
content = re.sub(r'\s*// Debug logging for name issue\s*\n\s*logrus\.Infof\("\[SEQUENCE-NAME\] Contact:.*?\n', '\n', content, flags=re.DOTALL)
content = re.sub(r'logrus\.Infof\("\S+ Step %d for %s not ready', 'logrus.Debugf("Step %d for %s not ready', content)
content = re.sub(r'logrus\.Infof\("Enrolling contact', 'logrus.Debugf("Enrolling contact', content)
content = re.sub(r'logrus\.Infof\("Step \d+: PENDING', 'logrus.Debugf("Step pending', content)
content = re.sub(r'logrus\.Infof\("\[ENROLLMENT\]', 'logrus.Debugf("[ENROLLMENT]', content)
content = re.sub(r'logrus\.Infof\("Performance:', 'logrus.Debugf("Performance:', content)

# Write back
with open(seq_file, 'w', encoding='utf-8') as f:
    f.write(content)

print("Done with sequence logs")

# Handle webhook logs
webhook_file = os.path.join(base_path, 'src', 'ui', 'rest', 'webhook_lead.go')
if os.path.exists(webhook_file):
    print("Reducing webhook logs...")
    with open(webhook_file, 'r', encoding='utf-8') as f:
        webhook_content = f.read()

    # Change webhook Info to Debug
    webhook_content = re.sub(r'logrus\.Info\("Webhook received', 'logrus.Debug("Webhook received', webhook_content)
    webhook_content = re.sub(r'logrus\.Info\("Webhook: Processing', 'logrus.Debug("Webhook: Processing', webhook_content)
    webhook_content = re.sub(r'logrus\.Info\("Webhook: Creating', 'logrus.Debug("Webhook: Creating', webhook_content)
    webhook_content = re.sub(r'logrus\.Info\("Webhook: Found', 'logrus.Debug("Webhook: Found', webhook_content)
    webhook_content = re.sub(r'logrus\.Info\("Webhook: Updated', 'logrus.Debug("Webhook: Updated', webhook_content)
    webhook_content = re.sub(r'logrus\.Info\("Webhook lead created', 'logrus.Debug("Webhook lead created', webhook_content)

    with open(webhook_file, 'w', encoding='utf-8') as f:
        f.write(webhook_content)
    
    print("Done with webhook logs")

# Broadcast processor
broadcast_file = os.path.join(base_path, 'src', 'usecase', 'ultra_optimized_broadcast_processor.go')
if os.path.exists(broadcast_file):
    print("Reducing broadcast logs...")
    with open(broadcast_file, 'r', encoding='utf-8') as f:
        broadcast_content = f.read()
    
    broadcast_content = broadcast_content.replace(
        'logrus.Infof("Queued %d messages to broadcast pools"',
        'logrus.Debugf("Queued %d messages to broadcast pools"'
    )
    
    with open(broadcast_file, 'w', encoding='utf-8') as f:
        f.write(broadcast_content)
    
    print("Done with broadcast logs")

print("\nAll logs reduced successfully!")
print("Info level logs changed to Debug level")
