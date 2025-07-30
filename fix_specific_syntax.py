import os
import re

print("Fixing specific Go syntax errors in infrastructure/whatsapp...")

# Fix 1: auto_connection_monitor_15min.go - SELECT should be select
file_path = r'src\infrastructure\whatsapp\auto_connection_monitor_15min.go'
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

content = content.replace('SELECT {', 'select {')

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)
print("Fixed auto_connection_monitor_15min.go - SELECT -> select")

# Fix 2: auto_reconnect.go - fix SQL syntax
file_path = r'src\infrastructure\whatsapp\auto_reconnect.go'
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Fix backticks around SQL keywords
content = content.replace('`status`', 'status')  # status is a column name, keep it
content = content.replace('`order`', 'ORDER')    # order is SQL keyword
content = content.replace('from user_devices', 'FROM user_devices')
content = content.replace('limit 20', 'LIMIT 20')

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)
print("Fixed auto_reconnect.go - SQL syntax")

# Fix 3: chat_store.go - check line 339
file_path = r'src\infrastructure\whatsapp\chat_store.go'
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Replace lowercase SQL keywords with uppercase
content = re.sub(r'\bfrom\s+whatsapp_chats\b', 'FROM whatsapp_chats', content)
content = re.sub(r'\border\s+by\b', 'ORDER BY', content, flags=re.IGNORECASE)
content = re.sub(r'\blimit\s+(\d+)', r'LIMIT \1', content)
content = re.sub(r'\bselect\s+', 'SELECT ', content, flags=re.IGNORECASE)
content = re.sub(r'\bwhere\s+', 'WHERE ', content, flags=re.IGNORECASE)
content = re.sub(r'\band\s+', 'AND ', content, flags=re.IGNORECASE)

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)
print("Fixed chat_store.go - SQL keywords")

# Fix 4: chat_to_leads.go - fix backtick around limit
file_path = r'src\infrastructure\whatsapp\chat_to_leads.go'
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

# Fix the specific backtick issue
content = content.replace('`limit` 1', 'LIMIT 1')
content = content.replace('`type`', 'type')  # If type is used as column name
content = re.sub(r'\bfrom\s+leads\b', 'FROM leads', content)
content = re.sub(r'\blimit\s+(\d+)', r'LIMIT \1', content)

with open(file_path, 'w', encoding='utf-8') as f:
    f.write(content)
print("Fixed chat_to_leads.go - limit backticks")

# Fix 5: connection_manager.go - case issue
file_path = r'src\infrastructure\whatsapp\connection_manager.go'
if os.path.exists(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Make sure SELECT in Go code is lowercase select
    content = re.sub(r'(\s+)SELECT\s*{', r'\1select {', content)
    
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print("Fixed connection_manager.go - select statement")

print("\nAll specific syntax errors fixed!")
