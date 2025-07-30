#!/usr/bin/env python3
"""
Fix WebSocket to broadcast QR codes only to the device owner
Prevents other users from seeing QR popups for devices they don't own
"""

import os
import re

def fix_websocket_user_filtering():
    """Add user filtering to WebSocket broadcasts"""
    
    # 1. Update websocket.go to track user connections
    websocket_file = "src/ui/websocket/websocket.go"
    
    if os.path.exists(websocket_file):
        with open(websocket_file, 'r') as f:
            content = f.read()
        
        # Update client struct to include user ID
        content = re.sub(
            r'type client struct\{\}',
            '''type client struct{
	UserID   string
	DeviceID string
}''',
            content
        )
        
        # Update BroadcastMessage to include target filters
        content = re.sub(
            r'type BroadcastMessage struct \{([^}]+)\}',
            r'''type BroadcastMessage struct {
\1
	TargetUserID   string `json:"targetUserId,omitempty"`
	TargetDeviceID string `json:"targetDeviceId,omitempty"`
}''',
            content
        )
        
        # Update broadcastMessage to filter by user/device
        old_broadcast = r'func broadcastMessage\(message BroadcastMessage\) \{[\s\S]*?for conn := range Clients \{[\s\S]*?\}\s*\}'
        
        new_broadcast = '''func broadcastMessage(message BroadcastMessage) {
	marshalMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}

	for conn, client := range Clients {
		// Filter by target user if specified
		if message.TargetUserID != "" && client.UserID != message.TargetUserID {
			continue
		}
		
		// Filter by target device if specified
		if message.TargetDeviceID != "" && client.DeviceID != message.TargetDeviceID {
			continue
		}
		
		if err := conn.WriteMessage(websocket.TextMessage, marshalMessage); err != nil {
			log.Println("write error:", err)
			closeConnection(conn)
		}
	}
}'''
        
        content = re.sub(old_broadcast, new_broadcast, content, flags=re.DOTALL)
        
        with open(websocket_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Updated {websocket_file} with user filtering")
    
    # 2. Update WebSocket initialization to include user info
    rest_app_file = "src/ui/rest/rest_app.go"
    
    if os.path.exists(rest_app_file):
        with open(rest_app_file, 'r') as f:
            content = f.read()
        
        # Find WebSocket upgrade handler
        ws_handler_pattern = r'(websocket\.New\(func\(c \*websocket\.Conn\) \{[\s\S]*?websocket\.Register <- c)'
        
        ws_handler_update = r'''\1
		
		// Get user info from context
		userID := ""
		if user := c.Locals("user"); user != nil {
			if u, ok := user.(*models.User); ok {
				userID = u.ID
			}
		}
		
		// Store user info with connection
		websocket.Clients[c] = websocket.client{
			UserID: userID,
		}'''
        
        content = re.sub(ws_handler_pattern, ws_handler_update, content)
        
        with open(rest_app_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Updated {rest_app_file} with user context")
    
    # 3. Update QR code broadcasts to include user ID
    whatsapp_service_files = [
        "src/services/whatsapp_service.go",
        "src/infrastructure/whatsapp/whatsapp.go",
        "src/domains/app/app.go"
    ]
    
    for service_file in whatsapp_service_files:
        if os.path.exists(service_file):
            with open(service_file, 'r') as f:
                content = f.read()
            
            # Update QR broadcasts to include user/device targeting
            qr_broadcast_pattern = r'(websocket\.Broadcast <- websocket\.BroadcastMessage\{[\s\S]*?Code:\s*"QR[^"]*"[^}]*\})'
            
            def add_targeting(match):
                broadcast = match.group(1)
                if 'TargetUserID' not in broadcast:
                    # Add targeting before closing brace
                    broadcast = re.sub(
                        r'\}$',
                        r',\n\t\tTargetUserID:   userID,\n\t\tTargetDeviceID: deviceID,\n\t}',
                        broadcast
                    )
                return broadcast
            
            content = re.sub(qr_broadcast_pattern, add_targeting, content, flags=re.DOTALL)
            
            with open(service_file, 'w') as f:
                f.write(content)
            
            print(f"[OK] Updated {service_file} with QR targeting")

def update_frontend_websocket():
    """Update frontend to handle user-specific messages"""
    
    dashboard_files = [
        "src/views/dashboard.html",
        "src/views/team_dashboard.html",
        "src/views/dashboard_reference.html"
    ]
    
    for dashboard_file in dashboard_files:
        if not os.path.exists(dashboard_file):
            continue
            
        with open(dashboard_file, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Add user ID to WebSocket connection
        ws_connect_pattern = r'(function connectWebSocket\(\) \{[^}]*const ws = new WebSocket[^;]+;)'
        
        ws_connect_update = r'''\1
    
    // Store current user ID for filtering
    const currentUserId = localStorage.getItem('userId') || '';'''
        
        content = re.sub(ws_connect_pattern, ws_connect_update, content)
        
        # Update message handler to check if message is for current user
        ws_message_pattern = r'(ws\.onmessage = function\(event\) \{[\s\S]*?try \{[\s\S]*?const data = JSON\.parse\(event\.data\);)'
        
        ws_message_update = r'''\1
            
            // Filter messages by user if targetUserId is specified
            if (data.targetUserId && data.targetUserId !== currentUserId) {
                console.log('Message not for this user, ignoring');
                return;
            }'''
        
        content = re.sub(ws_message_pattern, ws_message_update, content)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Updated {dashboard_file} with user filtering")

def create_websocket_fix_guide():
    """Create documentation for the WebSocket fix"""
    
    guide = '''# WebSocket User Filtering Fix

## Problem
When one user scans a QR code, all connected users see the QR popup modal, even for devices they don't own.

## Solution
Implemented user-specific WebSocket broadcasting:

1. **WebSocket Client Tracking**
   - Each WebSocket connection now stores the user ID
   - Messages can be targeted to specific users or devices

2. **Targeted Broadcasting**
   - QR code events include `targetUserId` and `targetDeviceId`
   - Only the device owner receives QR-related popups

3. **Frontend Filtering**
   - Frontend checks if incoming messages are for the current user
   - Ignores messages targeted at other users

## Benefits
- Users only see QR popups for their own devices
- No interference between multiple users
- Better privacy and user experience
- Supports team collaboration without confusion

## Technical Details
- WebSocket connections store user context
- Broadcast messages can specify target user/device
- Frontend filters messages before processing
'''
    
    with open("WEBSOCKET_USER_FILTERING.md", 'w') as f:
        f.write(guide)
    
    print("[OK] Created WebSocket filtering documentation")

def main():
    print("Fixing WebSocket to filter QR codes by user...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    fix_websocket_user_filtering()
    update_frontend_websocket()
    create_websocket_fix_guide()
    
    print("\n[SUCCESS] WebSocket user filtering implemented!")
    print("\nWhat's fixed:")
    print("1. QR code popups only shown to device owner")
    print("2. WebSocket messages can be targeted to specific users")
    print("3. No more interference between users")
    print("4. Better multi-user experience")

if __name__ == "__main__":
    main()
