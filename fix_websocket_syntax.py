#!/usr/bin/env python3
import os

# Fix the syntax error in websocket.go
websocket_file = "src/ui/websocket/websocket.go"

with open(websocket_file, 'r') as f:
    content = f.read()

# Remove the extra closing brace
content = content.replace('}\n}\n\nfunc closeConnection', '}\n\nfunc closeConnection')

with open(websocket_file, 'w') as f:
    f.write(content)

print("[OK] Fixed syntax error in websocket.go")
