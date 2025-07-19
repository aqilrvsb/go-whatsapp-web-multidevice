package whatsapp

import (
	"context"
	"database/sql"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"sync"
	"time"
)

// AutoReconnectService handles automatic reconnection of devices
type AutoReconnectService struct {
	db                   *sql.DB
	clientManager        *ClientManager
	reconnectRoutineOnce sync.Once
}

var (
	autoReconnectInstance *AutoReconnectService
	autoReconnectOnce     sync.Once
)

// GetAutoReconnectService returns the singleton instance of auto reconnect service
func GetAutoReconnectService(db *sql.DB) *AutoReconnectService {
	autoReconnectOnce.Do(func() {
		autoReconnectInstance = &AutoReconnectService{
			db:            db,
			clientManager: GetClientManager(),
		}
	})
	return autoReconnectInstance
}

// TryReconnectOfflineDevices attempts to reconnect devices that have stored sessions
func (ar *AutoReconnectService) TryReconnectOfflineDevices() {
	logrus.Info("=== Starting Auto-Reconnect for Offline Devices ===")
	
	// Query devices that are offline but have JID (meaning they have been connected before)
	query := `
		SELECT id, device_name, phone, jid, user_id, platform
		FROM user_devices
		WHERE status = 'offline' 
		AND jid IS NOT NULL 
		AND jid != ''
		AND (platform IS NULL OR platform = '')
		ORDER BY created_at ASC
		LIMIT 20
	`
	
	rows, err := ar.db.Query(query)
	if err != nil {
		logrus.Errorf("Failed to query offline devices: %v", err)
		return
	}
	defer rows.Close()
	
	attemptCount := 0
	successCount := 0
	
	for rows.Next() {
		var deviceID, name, phone, jid, userID, platform string
		err := rows.Scan(&deviceID, &name, &phone, &jid, &userID, &platform)
		if err != nil {
			logrus.Warnf("Failed to scan device row: %v", err)
			continue
		}
		
		// Skip platform devices - this should not happen due to WHERE clause, but double check
		if platform != "" {
			continue
		}
		
		attemptCount++
		logrus.Infof("[%d] Attempting to reconnect device %s (%s) with JID %s", attemptCount, name, deviceID, jid)
		
		// Try to reconnect this device
		go func(devID, devName, devJID, devPhone string) {
			// Small delay to prevent overwhelming the system
			time.Sleep(time.Duration(attemptCount) * 2 * time.Second)
			
			if err := ar.reconnectDevice(devID, devJID); err != nil {
				logrus.Errorf("Failed to reconnect %s: %v", devName, err)
			} else {
				successCount++
				logrus.Infof("✓ Successfully reconnected %s", devName)
			}
		}(deviceID, name, jid, phone)
	}
	
	logrus.Infof("Auto-reconnect initiated for %d devices", attemptCount)
}

// reconnectDevice attempts to reconnect a specific device using its JID
func (ar *AutoReconnectService) reconnectDevice(deviceID, jid string) error {
	// Try to get WhatsApp device by JID
	waDevice, err := GetContainer().GetDevice(context.Background(), parseJID(jid))
	if err != nil {
		return err
	}
	
	if waDevice == nil {
		return fmt.Errorf("no stored session found for JID %s", jid)
	}
	
	// Create WhatsApp client
	client := whatsmeow.NewClient(waDevice, GetWaLog())
	
	// Set up event handlers
	SetupDeviceEventHandlers(client, deviceID)
	
	// Connect
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	
	// Register with client manager
	ar.clientManager.AddClient(deviceID, client)
	
	// Update status in database
	userRepo := repository.GetUserRepository()
	if err := userRepo.UpdateDeviceStatus(deviceID, "online", "", jid); err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	return nil
}

// parseJID converts string JID to JID object
func parseJID(jidStr string) whatsmeow.JID {
	// Simple parsing - more robust parsing might be needed
	parts := strings.Split(jidStr, "@")
	if len(parts) == 2 {
		userParts := strings.Split(parts[0], ":")
		if len(userParts) == 2 {
			return whatsmeow.JID{
				User:   userParts[0],
				Agent:  userParts[1],
				Device: 0,
				Server: parts[1],
			}
		}
	}
	return whatsmeow.JID{}
}

// StartAutoReconnectRoutine starts a routine that periodically tries to reconnect devices
func (ar *AutoReconnectService) StartAutoReconnectRoutine() {
	ar.reconnectRoutineOnce.Do(func() {
		go func() {
			// Initial delay to let system start up
			time.Sleep(30 * time.Second)
			
			// Run initial reconnection attempt
			ar.TryReconnectOfflineDevices()
			
			// Then run periodically
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			
			for range ticker.C {
				ar.TryReconnectOfflineDevices()
			}
		}()
		
		logrus.Info("Auto-reconnect routine started (runs every 5 minutes)")
	})
}

// Add this to your init/startup
func StartAutoReconnectRoutine(db *sql.DB) {
	service := GetAutoReconnectService(db)
	service.StartAutoReconnectRoutine()
}