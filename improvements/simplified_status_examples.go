// Simplified device checking for campaigns and broadcasts

// For Campaign Trigger - BEFORE:
if device.Status == "connected" || device.Status == "Connected" || 
   device.Status == "online" || device.Status == "Online" {
    // Use device
}

// For Campaign Trigger - AFTER:
if device.Status == "online" {
    // Use device
}

// For Broadcast Processor - BEFORE:
if device.Status != "online" && device.Status != "Online" && 
   device.Status != "connected" && device.Status != "Connected" {
    // Skip device
}

// For Broadcast Processor - AFTER:
if device.Status != "online" {
    // Skip device
}

// For Sequence Processor - BEFORE:
WHERE d.status = 'online'

// For Sequence Processor - AFTER (no change needed):
WHERE d.status = 'online'

// For Device Health Check:
func checkDeviceHealth(deviceID string, client *whatsmeow.Client) {
    newStatus := "offline"
    if client != nil && client.IsConnected() {
        newStatus = "online"
    }
    
    userRepo.UpdateDeviceStatus(deviceID, newStatus, "", "")
}