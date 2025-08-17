package whatsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp/multidevice"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// EnhancedLogout performs complete logout with all cleanup steps
func EnhancedLogout(deviceID string) error {
	logrus.Infof("=== STARTING ENHANCED LOGOUT FOR DEVICE %s ===", deviceID)
	
	// Step 1: Get device info
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}
	
	var phoneNumber string
	var jidStr string
	
	// Step 2: Get client and perform WhatsApp logout
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err == nil && client != nil {
		// Get phone and JID before logout
		if client.Store != nil && client.Store.ID != nil {
			phoneNumber = client.Store.ID.User
			jidStr = client.Store.ID.String()
			logrus.Infof("Device info - Phone: %s, JID: %s", phoneNumber, jidStr)
		}
		
		// Logout from WhatsApp
		if client.IsConnected() {
			logrus.Info("Logging out from WhatsApp...")
			err := client.Logout(context.Background())
			if err != nil {
				logrus.Errorf("WhatsApp logout error: %v", err)
			} else {
				logrus.Info("Successfully logged out from WhatsApp")
			}
			
			// Wait for logout to process
			time.Sleep(2 * time.Second)
		}
		
		// Disconnect client
		client.Disconnect()
		
		// Remove from managers
		cm.RemoveClient(deviceID)
		
		// Remove from device connection manager
		dcm := GetDeviceConnectionManager()
		dcm.RemoveConnection(deviceID)
	}
	
	// Step 3: Clear from multidevice manager
	dm := multidevice.GetDeviceManager()
	if dm != nil {
		dm.UpdateDeviceStatus(deviceID, false, phoneNumber)
	}
	
	// Step 4: Clear device store in WhatsApp database
	if device.JID != "" {
		err = ClearDeviceFromStore(device.JID)
		if err != nil {
			logrus.Errorf("Error clearing device from store: %v", err)
		}
	}
	
	// Step 5: Clear WhatsApp session data
	err = ClearWhatsAppSessionData(deviceID)
	if err != nil {
		logrus.Errorf("Error clearing WhatsApp session: %v", err)
	}
	
	// Step 6: Update device status to offline
	if phoneNumber == "" && device.Phone != "" {
		phoneNumber = device.Phone
	}
	if jidStr == "" && device.JID != "" {
		jidStr = device.JID
	}
	
	err = userRepo.UpdateDeviceStatus(deviceID, "offline", phoneNumber, jidStr)
	if err != nil {
		logrus.Errorf("Error updating device status: %v", err)
	}
	
	// Step 7: Clear any QR channels
	ClearDeviceQRChannel(deviceID)
	
	// Step 8: Clear connection sessions
	ClearConnectionSession("")
	
	// Step 9: Send logout notification
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_LOGGED_OUT",
		Message: "Device successfully logged out",
		Result: map[string]interface{}{
			"deviceId": deviceID,
			"phone":    phoneNumber,
			"status":   "offline",
		},
	}
	
	logrus.Infof("=== ENHANCED LOGOUT COMPLETED FOR DEVICE %s ===", deviceID)
	return nil
}

// ClearDeviceFromStore removes device from WhatsApp store
func ClearDeviceFromStore(jidStr string) error {
	// Parse JID
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	
	// Get WhatsApp database
	dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)
	container, err := sqlstore.New(context.Background(), "sqlite3", 
		fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", config.DBURI), dbLog)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	
	// Get device from store
	device, err := container.GetDevice(context.Background(), jid)
	if err != nil {
		return fmt.Errorf("failed to get device from store: %v", err)
	}
	
	if device == nil {
		logrus.Warn("Device not found in store")
		return nil
	}
	
	// Delete device
	err = device.Delete(context.Background())
	if err != nil {
		return fmt.Errorf("failed to delete device from store: %v", err)
	}
	
	logrus.Info("Successfully deleted device from WhatsApp store")
	return nil
}

// ForceLogoutAllDevices logs out all devices for a user
func ForceLogoutAllDevices(userID string) error {
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetUserDevices(userID)
	if err != nil {
		return err
	}
	
	for _, device := range devices {
		if device.Status != "offline" {
			err = EnhancedLogout(device.ID)
			if err != nil {
				logrus.Errorf("Error logging out device %s: %v", device.ID, err)
			}
		}
	}
	
	return nil
}

// VerifyDeviceLoggedOut checks if device is truly logged out
func VerifyDeviceLoggedOut(deviceID string) bool {
	// Check client manager
	cm := GetClientManager()
	if client, err := cm.GetClient(deviceID); err == nil && client != nil {
		if client.IsConnected() || client.IsLoggedIn() {
			logrus.Warnf("Device %s still connected/logged in", deviceID)
			return false
		}
	}
	
	// Check device connection manager
	dcm := GetDeviceConnectionManager()
	dcm.mu.RLock()
	if info, exists := dcm.activeConnections[deviceID]; exists {
		dcm.mu.RUnlock()
		if info.Client != nil && (info.Client.IsConnected() || info.Client.IsLoggedIn()) {
			logrus.Warnf("Device %s found in connection manager", deviceID)
			return false
		}
	} else {
		dcm.mu.RUnlock()
	}
	
	// Check database status
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err == nil && device != nil && device.Status != "offline" {
		logrus.Warnf("Device %s status is not offline: %s", deviceID, device.Status)
		return false
	}
	
	return true
}
