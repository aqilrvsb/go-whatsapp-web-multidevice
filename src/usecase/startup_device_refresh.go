package usecase

import (
	"database/sql"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
)

// StartupDeviceRefresh refreshes all non-platform devices on server startup
func StartupDeviceRefresh() {
	logrus.Info("=== STARTUP DEVICE REFRESH STARTED ===")
	
	// Wait a bit for all services to initialize
	time.Sleep(10 * time.Second)
	
	db := database.GetDB()
	
	// Get all non-platform devices
	rows, err := db.Query(`
		SELECT id, device_name, phone, jid, status
		FROM user_devices 
		WHERE (platform IS NULL OR platform = '')
		AND jid IS NOT NULL
		ORDER BY device_name
	`)
	if err != nil {
		logrus.Errorf("Failed to get devices for refresh: %v", err)
		return
	}
	defer rows.Close()
	
	var devices []struct {
		ID         string
		DeviceName string
		Phone      string
		JID        string
		Status     string
	}
	
	for rows.Next() {
		var device struct {
			ID         string
			DeviceName string
			Phone      string
			JID        string
			Status     string
		}
		
		var phone, jid, status sql.NullString
		err := rows.Scan(&device.ID, &device.DeviceName, &phone, &jid, &status)
		if err != nil {
			logrus.Errorf("Failed to scan device: %v", err)
			continue
		}
		
		device.Phone = phone.String
		device.JID = jid.String
		device.Status = status.String
		
		devices = append(devices, device)
	}
	
	logrus.Infof("Found %d WhatsApp Web devices to refresh", len(devices))
	
	// Create wait group for concurrent refresh
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent refreshes
	
	onlineCount := 0
	offlineCount := 0
	mu := sync.Mutex{}
	
	for _, device := range devices {
		wg.Add(1)
		go func(dev struct {
			ID         string
			DeviceName string
			Phone      string
			JID        string
			Status     string
		}) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			logrus.Infof("Checking device: %s (current status: %s)", dev.DeviceName, dev.Status)
			
			// Try to get client from ClientManager
			clientManager := whatsapp.GetClientManager()
			client, err := clientManager.GetClient(dev.ID)
			
			newStatus := "offline"
			
			if err != nil {
				logrus.Debugf("Device %s: No client found - marking offline", dev.DeviceName)
			} else if client == nil {
				logrus.Debugf("Device %s: Client is nil - marking offline", dev.DeviceName)
			} else {
				// Check if client is logged in and connected
				if client.IsLoggedIn() {
					if client.IsConnected() {
						newStatus = "online"
						logrus.Infof("Device %s: Connected and online ✓", dev.DeviceName)
					} else {
						// Try to reconnect
						logrus.Infof("Device %s: Logged in but disconnected, attempting reconnect...", dev.DeviceName)
						err := client.Connect()
						if err != nil {
							logrus.Errorf("Device %s: Reconnect failed - %v", dev.DeviceName, err)
						} else {
							// Wait a bit for connection to establish
							time.Sleep(3 * time.Second)
							if client.IsConnected() {
								newStatus = "online"
								logrus.Infof("Device %s: Reconnected successfully ✓", dev.DeviceName)
							} else {
								logrus.Warnf("Device %s: Still disconnected after reconnect attempt", dev.DeviceName)
							}
						}
					}
				} else {
					logrus.Debugf("Device %s: Not logged in - marking offline", dev.DeviceName)
				}
			}
			
			// Update device status in database
			_, err = db.Exec(`
				UPDATE user_devices 
				SET status = $1, last_seen = NOW(), updated_at = NOW()
				WHERE id = $2
			`, newStatus, dev.ID)
			
			if err != nil {
				logrus.Errorf("Failed to update device %s status: %v", dev.DeviceName, err)
			} else {
				logrus.Debugf("Device %s status updated to: %s", dev.DeviceName, newStatus)
			}
			
			// Update counters
			mu.Lock()
			if newStatus == "online" {
				onlineCount++
			} else {
				offlineCount++
			}
			mu.Unlock()
			
		}(device)
	}
	
	// Wait for all refreshes to complete
	wg.Wait()
	
	logrus.Info("=== STARTUP DEVICE REFRESH COMPLETED ===")
	logrus.Infof("Summary: %d devices online, %d devices offline", onlineCount, offlineCount)
	logrus.Info("You can now send messages without clicking refresh buttons!")
}
