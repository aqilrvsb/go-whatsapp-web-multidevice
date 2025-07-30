#!/usr/bin/env python3
"""
Simple fix: Add device ID to QR broadcasts to prevent cross-user QR popups
"""

import os
import re

def simple_qr_fix():
    """Add device ID to QR code messages for filtering"""
    
    # Update app.go to include device ID in QR broadcasts
    app_file = "src/domains/app/app.go"
    
    if os.path.exists(app_file):
        with open(app_file, 'r') as f:
            content = f.read()
        
        # Find QR code broadcast and add device ID
        qr_pattern = r'(case "code":\s*qrCode[^}]+websocket\.Broadcast <- websocket\.BroadcastMessage\{[^}]+\})'
        
        def add_device_info(match):
            broadcast = match.group(0)
            # Check if Result already has deviceId
            if '"deviceId"' not in broadcast:
                # Add deviceId to the Result
                broadcast = re.sub(
                    r'(Result:\s*map\[string\]any\{)',
                    r'\1\n\t\t\t\t"deviceId": deviceJid,',
                    broadcast
                )
            return broadcast
        
        content = re.sub(qr_pattern, add_device_info, content, flags=re.DOTALL)
        
        with open(app_file, 'w') as f:
            f.write(content)
        
        print(f"[OK] Updated {app_file} to include device ID in QR broadcasts")

def update_frontend_qr_filter():
    """Update frontend to only show QR for current device"""
    
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
        
        # Find where QR code is displayed and add device check
        qr_display_pattern = r'(case \'QR_CODE\':[^}]*if \(data\.result && data\.result\.qrCode\) \{)'
        
        qr_display_update = r'''\1
                        // Check if QR is for current device being connected
                        const connectingDeviceId = sessionStorage.getItem('connectingDeviceId');
                        if (data.result.deviceId && connectingDeviceId && data.result.deviceId !== connectingDeviceId) {
                            console.log('QR code for different device, ignoring');
                            return;
                        }'''
        
        content = re.sub(qr_display_pattern, qr_display_update, content)
        
        # Store device ID when connecting
        connect_pattern = r'(function connectDevice\(deviceId\) \{)'
        connect_update = r'''\1
    // Store device ID being connected
    sessionStorage.setItem('connectingDeviceId', deviceId);'''
        
        content = re.sub(connect_pattern, connect_update, content)
        
        # Clear device ID when modal closes
        modal_close_pattern = r'(qrModal\.hide\(\);)'
        modal_close_update = r'''\1
            sessionStorage.removeItem('connectingDeviceId');'''
        
        content = re.sub(modal_close_pattern, modal_close_update, content)
        
        with open(dashboard_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"[OK] Updated {dashboard_file} with QR device filtering")

def main():
    print("Applying simple QR code filtering fix...")
    
    os.chdir(r"C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main")
    
    simple_qr_fix()
    update_frontend_qr_filter()
    
    print("\n[SUCCESS] QR filtering implemented!")
    print("\nWhat's fixed:")
    print("1. QR codes include device ID")
    print("2. Frontend only shows QR for device being connected")
    print("3. Other users won't see QR popups for devices they're not connecting")

if __name__ == "__main__":
    main()
