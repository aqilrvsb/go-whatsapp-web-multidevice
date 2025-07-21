import re
import os

base_path = r'C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main'

# Read the sequence file
seq_file = os.path.join(base_path, 'src', 'usecase', 'sequence_trigger_processor.go')
with open(seq_file, 'r', encoding='utf-8') as f:
    content = f.read()

# Count changes
changes = 0

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
    (r'logrus\.Infof\("Step 1: PENDING', 'logrus.Debugf("Step 1: PENDING'),
    (r'logrus\.Infof\("Step %d: PENDING', 'logrus.Debugf("Step %d: PENDING'),
    (r'logrus\.Infof\("\[ENROLLMENT\]', 'logrus.Debugf("[ENROLLMENT]'),
    
    # Performance metrics
    (r'logrus\.Infof\("Performance:', 'logrus.Debugf("Performance:'),
    
    # Summary logs
    (r'logrus\.Infof\("✅ Enrollment complete', 'logrus.Debugf("✅ Enrollment complete'),
]

# Apply replacements
for old, new in replacements:
    new_content, count = re.subn(old, new, content, flags=re.MULTILINE | re.DOTALL)
    if count > 0:
        content = new_content
        changes += count
        print(f"  Replaced {count} occurrence(s) of: {old[:50]}...")

# Write back
with open(seq_file, 'w', encoding='utf-8') as f:
    f.write(content)

print(f"\n✅ Reduced sequence logs - {changes} changes made")

# Now handle webhook logs
webhook_file = os.path.join(base_path, 'src', 'ui', 'rest', 'webhook_lead.go')
if os.path.exists(webhook_file):
    with open(webhook_file, 'r', encoding='utf-8') as f:
        webhook_content = f.read()

    webhook_changes = 0
    # Change most webhook Info logs to Debug
    webhook_replacements = [
        (r'logrus\.Info\("Webhook received', 'logrus.Debug("Webhook received'),
        (r'logrus\.Info\("Webhook: Processing', 'logrus.Debug("Webhook: Processing'),
        (r'logrus\.Info\("Webhook: Creating', 'logrus.Debug("Webhook: Creating'),
        (r'logrus\.Info\("Webhook: Found', 'logrus.Debug("Webhook: Found'),
        (r'logrus\.Info\("Webhook: Updated', 'logrus.Debug("Webhook: Updated'),
        (r'logrus\.Info\("Webhook lead created', 'logrus.Debug("Webhook lead created'),
        # Keep error logs as they are
    ]

    for old, new in webhook_replacements:
        new_content, count = re.subn(old, new, webhook_content)
        if count > 0:
            webhook_content = new_content
            webhook_changes += count

    with open(webhook_file, 'w', encoding='utf-8') as f:
        f.write(webhook_content)

    print(f"✅ Reduced webhook logs - {webhook_changes} changes made")

# Also reduce broadcast processor logs
broadcast_file = os.path.join(base_path, 'src', 'usecase', 'ultra_optimized_broadcast_processor.go')
if os.path.exists(broadcast_file):
    with open(broadcast_file, 'r', encoding='utf-8') as f:
        broadcast_content = f.read()
    
    # Change to Debug
    broadcast_content = broadcast_content.replace(
        'logrus.Infof("Queued %d messages to broadcast pools"',
        'logrus.Debugf("Queued %d messages to broadcast pools"'
    )
    
    with open(broadcast_file, 'w', encoding='utf-8') as f:
        f.write(broadcast_content)
    
    print("✅ Reduced broadcast processor logs")

print("\n📝 Summary: Changed verbose Info logs to Debug level")
print("   - Sequence processing logs")
print("   - Webhook processing logs")
print("   - Broadcast queue logs")
print("\nℹ️  Important logs (success/failure) remain at Info level")
