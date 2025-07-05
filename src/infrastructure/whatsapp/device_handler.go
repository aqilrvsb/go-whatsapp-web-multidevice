package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/sirupsen/logrus"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp/multidevice"
	websocket "github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	deviceQRChannels   = make(map[string]<-chan whatsmeow.QRChannelItem) // deviceID -> QR channel
	qrMutex            sync.RWMutex
)

// SetDeviceQRChannel stores QR channel for a device
func SetDeviceQRChannel(deviceID string, qrChan <-chan whatsmeow.QRChannelItem) {
	qrMutex.Lock()
	defer qrMutex.Unlock()
	deviceQRChannels[deviceID] = qrChan
	
	// Start goroutine to handle QR updates
	go func() {
		for qrItem := range qrChan {
			// Broadcast QR update via websocket
			websocket.Broadcast <- websocket.BroadcastMessage{
				Code:    "QR_UPDATE",
				Message: "QR code updated",
				Result: map[string]interface{}{
					"deviceId": deviceID,
					"qr":       qrItem.Code,
					"timeout":  qrItem.Timeout,
				},
			}
		}
		// Channel closed, remove it
		ClearDeviceQRChannel(deviceID)
	}()
}

// GetDeviceQR gets the current QR from channel
func GetDeviceQR(deviceID string) (string, error) {
	qrMutex.RLock()
	qrChan, exists := deviceQRChannels[deviceID]
	qrMutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("no QR channel for device %s", deviceID)
	}
	
	// Try to get QR with timeout
	select {
	case qrItem, ok := <-qrChan:
		if !ok {
			return "", fmt.Errorf("QR channel closed")
		}
		return qrItem.Code, nil
	case <-time.After(1 * time.Second):
		return "", fmt.Errorf("no QR available")
	}
}

// ClearDeviceQRChannel removes QR channel for a device
func ClearDeviceQRChannel(deviceID string) {
	qrMutex.Lock()
	defer qrMutex.Unlock()
	delete(deviceQRChannels, deviceID)
}

// HandleDeviceEvent handles WhatsApp events for a specific device
func HandleDeviceEvent(ctx context.Context, deviceID string, rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.PairSuccess:
		handleDevicePairSuccess(ctx, deviceID, evt)
	case *events.Connected:
		handleDeviceConnected(ctx, deviceID)
	case *events.PushNameSetting:
		handleDeviceConnected(ctx, deviceID)
	case *events.LoggedOut:
		handleDeviceLoggedOut(ctx, deviceID)
	case *events.Message:
		// Skip - already handled in main handler (init.go)
		// This prevents duplicate message storage
		logrus.Debugf("Message event for device %s handled by main handler", deviceID)
	case *events.HistorySync:
		// Process history sync to get recent messages
		HandleHistorySyncForWebView(deviceID, evt)
		
		// Also update chat list
		cm := GetClientManager()
		if client, err := cm.GetClient(deviceID); err == nil {
			HandleHistorySyncForChats(deviceID, client, evt)
		}
	}
}

// handleDevicePairSuccess handles successful QR pairing for a device
func handleDevicePairSuccess(ctx context.Context, deviceID string, evt *events.PairSuccess) {
	logrus.Infof("Device %s paired successfully with %s", deviceID, evt.ID.String())
	
	// Get device connection
	dm := multidevice.GetDeviceManager()
	conn, err := dm.GetDeviceConnection(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get device connection: %v", err)
		return
	}
	
	// Update connection info
	conn.Phone = evt.ID.User
	
	// Broadcast pairing success
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_PAIRED",
		Message: fmt.Sprintf("Device paired with %s", evt.ID.String()),
		Result: map[string]interface{}{
			"deviceId": deviceID,
			"phone":    evt.ID.User,
		},
	}
}

// handleDeviceConnected handles full connection for a device
func handleDeviceConnected(ctx context.Context, deviceID string) {
	logrus.Infof("Device %s fully connected", deviceID)
	
	// Get device connection
	dm := multidevice.GetDeviceManager()
	conn, err := dm.GetDeviceConnection(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get device connection: %v", err)
		return
	}
	
	if conn.Client == nil || !conn.Client.IsLoggedIn() {
		logrus.Warnf("Device %s connected event but client not logged in", deviceID)
		return
	}
	
	// Get WhatsApp info
	var phoneNumber, jid string
	if conn.Client.Store.ID != nil {
		jid = conn.Client.Store.ID.String()
		phoneNumber = conn.Client.Store.ID.User
		logrus.Infof("Device %s connected as: %s (Phone: %s)", deviceID, jid, phoneNumber)
	}
	
	// Update device in database
	userRepo := repository.GetUserRepository()
	err = userRepo.UpdateDeviceStatus(deviceID, "online", phoneNumber, jid)
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	} else {
		logrus.Infof("Successfully updated device %s to online status", deviceID)
	}
	
	// Update device manager
	dm.UpdateDeviceStatus(deviceID, true, phoneNumber)
	
	// Register with client manager for broadcasts
	cm := GetClientManager()
	cm.AddClient(deviceID, conn.Client)
	logrus.Infof("Registered device %s with client manager", deviceID)
	
	// Clear QR channel
	ClearDeviceQRChannel(deviceID)
	
	// Broadcast connection success
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_CONNECTED",
		Message: "WhatsApp fully connected and logged in",
		Result: map[string]interface{}{
			"deviceId": deviceID,
			"phone":    phoneNumber,
			"jid":      jid,
		},
	}
	
	// Trigger initial sync after connection
	go func() {
		time.Sleep(3 * time.Second)
		chats, err := GetChatsForDevice(deviceID)
		if err != nil {
			logrus.Errorf("Failed to sync chats for device %s: %v", deviceID, err)
		} else {
			logrus.Infof("Successfully synced %d chats for device %s", len(chats), deviceID)
		}
	}()
}

// handleDeviceLoggedOut handles device logout
func handleDeviceLoggedOut(ctx context.Context, deviceID string) {
	logrus.Infof("Device %s logged out", deviceID)
	
	// Get phone number before clearing everything
	phoneNumber := ""
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err == nil && client != nil && client.Store != nil && client.Store.ID != nil {
		phoneNumber = client.Store.ID.User
	}
	
	// Clear WhatsApp session data first
	err = ClearWhatsAppSessionData(deviceID)
	if err != nil {
		logrus.Errorf("Failed to clear WhatsApp session data: %v", err)
		// Continue with logout even if session clear fails
	}
	
	// Update device status
	userRepo := repository.GetUserRepository()
	err = userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	// Update device manager
	dm := multidevice.GetDeviceManager()
	dm.UpdateDeviceStatus(deviceID, false, "")
	
	// Remove from client manager
	cm.RemoveClient(deviceID)
	
	// Clear QR channel
	ClearDeviceQRChannel(deviceID)
	
	// Broadcast logout with phone number (like QR scan does)
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_LOGGED_OUT",
		Message: "Device logged out",
		Result: map[string]interface{}{
			"deviceId": deviceID,
			"phone":    phoneNumber, // Include phone number for updateDeviceStatusByPhone
		},
	}
}

// ClearWhatsAppSessionData clears all WhatsApp session data for a device
func ClearWhatsAppSessionData(deviceID string) error {
	// Get repository to access DB
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// First, try to delete from whatsmeow_device as it's the parent table
	// This should cascade delete all related records
	_, err = tx.Exec(`DELETE FROM whatsmeow_device WHERE jid = $1`, deviceID)
	if err != nil {
		// If that fails, try without specifying JID (might be stored differently)
		logrus.Warnf("Failed to delete by JID, trying alternative approach: %v", err)
		
		// Try to find and delete by any matching device
		_, err = tx.Exec(`
			DELETE FROM whatsmeow_device 
			WHERE jid = $1 
			   OR jid LIKE '%' || $1 || '%'
			   OR EXISTS (
			       SELECT 1 FROM user_devices 
			       WHERE user_devices.id = $1 
			       AND whatsmeow_device.jid = user_devices.jid
			   )
		`, deviceID)
		
		if err != nil {
			logrus.Errorf("Failed to clear WhatsApp device: %v", err)
			// Don't return error, try to clean other tables
		}
	}
	
	// Commit transaction
	return tx.Commit()
}
