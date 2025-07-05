package whatsapp

import (
	"context"
	"time"
	
	"github.com/sirupsen/logrus"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// AutoReconnectDevices attempts to reconnect devices that were previously connected
func AutoReconnectDevices(container *sqlstore.Container) {
	logrus.Info("Starting auto-reconnect for previously connected devices...")
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// First, set all devices to offline to ensure clean state
	_, err := db.Exec(`UPDATE user_devices SET status = 'offline' WHERE status = 'online'`)
	if err != nil {
		logrus.Warnf("Failed to reset device statuses: %v", err)
	}
	
	// Find all devices that have a JID (were previously connected)
	rows, err := db.Query(`
		SELECT id, device_name, phone, jid 
		FROM user_devices 
		WHERE jid IS NOT NULL AND jid != ''
		ORDER BY last_seen DESC
	`)
	if err != nil {
		logrus.Errorf("Failed to query devices for auto-reconnect: %v", err)
		return
	}
	defer rows.Close()
	
	reconnectCount := 0
	attemptCount := 0
	
	for rows.Next() {
		var deviceID, deviceName, phone, jid string
		err := rows.Scan(&deviceID, &deviceName, &phone, &jid)
		if err != nil {
			logrus.Warnf("Failed to scan device row: %v", err)
			continue
		}
		
		attemptCount++
		logrus.Infof("[%d] Attempting to reconnect device %s (%s) with JID %s", attemptCount, deviceName, deviceID, jid)
		
		// Try to reconnect this device
		go func(devID, devName, devJID, devPhone string) {
			// Small delay to prevent overwhelming the system
			time.Sleep(time.Duration(attemptCount) * 2 * time.Second)
			
			// Try to get existing device from store
			devices, err := container.GetAllDevices(context.Background())
			if err != nil {
				logrus.Errorf("Failed to get devices from store: %v", err)
				userRepo.UpdateDeviceStatus(devID, "offline", devPhone, devJID)
				return
			}
			
			// Find the device with matching JID
			var device *store.Device
			for _, d := range devices {
				if d.ID != nil && d.ID.String() == devJID {
					device = d
					break
				}
			}
			
			if device == nil {
				logrus.Warnf("No stored session found for device %s with JID %s", devName, devJID)
				userRepo.UpdateDeviceStatus(devID, "offline", devPhone, devJID)
				return
			}
			
			// Create client with proper logging
			client := whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
			
			// Add device-specific event handler
			client.AddEventHandler(func(evt interface{}) {
				HandleDeviceEvent(context.Background(), devID, evt)
			})
			
			// Try to connect
			logrus.Infof("Connecting device %s...", devName)
			err = client.Connect()
			if err != nil {
				logrus.Errorf("Failed to connect device %s: %v", devName, err)
				userRepo.UpdateDeviceStatus(devID, "offline", devPhone, devJID)
				return
			}
			
			// Wait for connection to establish properly
			time.Sleep(5 * time.Second)
			
			// Verify connection status
			if !client.IsConnected() {
				logrus.Warnf("Device %s failed to establish connection", devName)
				client.Disconnect()
				userRepo.UpdateDeviceStatus(devID, "offline", devPhone, devJID)
				return
			}
			
			// Check if logged in
			if !client.IsLoggedIn() {
				logrus.Warnf("Device %s connected but not logged in - session expired", devName)
				client.Disconnect()
				userRepo.UpdateDeviceStatus(devID, "offline", devPhone, devJID)
				return
			}
			
			// Success! Register with client manager
			cm := GetClientManager()
			cm.AddClient(devID, client)
			
			// Update device status
			actualPhone := devPhone
			actualJID := devJID
			if client.Store.ID != nil {
				actualPhone = client.Store.ID.User
				actualJID = client.Store.ID.String()
			}
			
			err = userRepo.UpdateDeviceStatus(devID, "online", actualPhone, actualJID)
			if err != nil {
				logrus.Errorf("Failed to update device status: %v", err)
			}
			
			logrus.Infof("✅ Successfully reconnected device %s (%s)", devName, devID)
			
			// Send WebSocket notification
			websocket.Broadcast <- websocket.BroadcastMessage{
				Code:    "DEVICE_RECONNECTED",
				Message: "Device auto-reconnected after restart",
				Result: map[string]interface{}{
					"deviceId": devID,
					"phone":    actualPhone,
					"name":     devName,
					"status":   "online",
				},
			}
			
			reconnectCount++
			
		}(deviceID, deviceName, jid, phone)
	}
	
	// Wait a bit for all goroutines to start
	time.Sleep(2 * time.Second)
	
	logrus.Infof("Auto-reconnect initiated for %d devices", attemptCount)
}

// StartAutoReconnectRoutine starts a routine that periodically checks and reconnects devices
func StartAutoReconnectRoutine(container *sqlstore.Container, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			logrus.Debug("Running periodic auto-reconnect check...")
			AutoReconnectDevices(container)
		}
	}()
	
	logrus.Infof("Started auto-reconnect routine with %v interval", interval)
}