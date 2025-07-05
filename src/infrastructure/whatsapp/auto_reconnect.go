package whatsapp

import (
	"context"
	"time"
	
	"github.com/sirupsen/logrus"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// AutoReconnectDevices attempts to reconnect devices that were previously connected
func AutoReconnectDevices(container *sqlstore.Container) {
	logrus.Info("Starting auto-reconnect for previously connected devices...")
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Find all devices that have a JID (were previously connected)
	rows, err := db.Query(`
		SELECT id, name, phone, jid 
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
	for rows.Next() {
		var deviceID, name, phone, jid string
		err := rows.Scan(&deviceID, &name, &phone, &jid)
		if err != nil {
			logrus.Warnf("Failed to scan device row: %v", err)
			continue
		}
		
		logrus.Infof("Attempting to reconnect device %s (%s) with JID %s", name, deviceID, jid)
		
		// Try to reconnect this device
		go func(devID, devName, devJID string) {
			// Try to get existing device from store
			devices, err := container.GetAllDevices(context.Background())
			if err != nil {
				logrus.Warnf("Failed to get devices from store: %v", err)
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
				// Update status to offline
				userRepo.UpdateDeviceStatus(devID, "offline", "", "")
				return
			}
			
			// Create client
			client := whatsmeow.NewClient(device, nil)
			
			// Add event handler
			client.AddEventHandler(CreateDeviceEventHandlerForReconnect(devID))
			
			// Try to connect
			err = client.Connect()
			if err != nil {
				logrus.Warnf("Failed to reconnect device %s: %v", devName, err)
				// Update status to offline
				userRepo.UpdateDeviceStatus(devID, "offline", "", "")
				return
			}
			
			// Wait a bit for connection to establish
			time.Sleep(3 * time.Second)
			
			// Check if logged in
			if client.IsLoggedIn() && client.IsConnected() {
				logrus.Infof("Successfully reconnected device %s", devName)
				
				// Register with client manager
				cm := GetClientManager()
				cm.AddClient(devID, client)
				
				// Update status
				userRepo.UpdateDeviceStatus(devID, "online", client.Store.ID.User, client.Store.ID.String())
				
				reconnectCount++
			} else {
				logrus.Warnf("Device %s session expired or not connected, needs QR scan", devName)
				client.Disconnect()
				// Update status to offline but keep JID/phone
				userRepo.UpdateDeviceStatus(devID, "offline", phone, devJID)
			}
		}(deviceID, name, jid)
		
		// Small delay between reconnection attempts
		time.Sleep(2 * time.Second)
	}
	
	logrus.Infof("Auto-reconnect completed. Successfully reconnected %d devices", reconnectCount)
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
}

// CreateDeviceEventHandlerForReconnect creates an event handler for reconnected devices
func CreateDeviceEventHandlerForReconnect(deviceID string) func(evt interface{}) {
	// Return the device-specific event handler
	return func(evt interface{}) {
		// Let the device handler process events
		HandleDeviceEvent(context.Background(), deviceID, evt)
	}
}
