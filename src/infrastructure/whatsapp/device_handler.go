package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/log"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/src/infrastructure/whatsapp/multidevice"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/src/ui/websocket"
	"go.mau.fi/whatsmeow/types/events"
)

// ConnectionSession stores information about ongoing connections
type ConnectionSession struct {
	DeviceID string
	UserID   string
	Phone    string
}

var (
	connectionSessions = make(map[string]*ConnectionSession) // userID -> session
	sessionMutex       sync.RWMutex
	
	deviceQRChannels   = make(map[string]<-chan string) // deviceID -> QR channel
	qrMutex            sync.RWMutex
)

// SetConnectionSession stores a connection session
func SetConnectionSession(userID string, session *ConnectionSession) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	connectionSessions[userID] = session
}

// GetConnectionSession gets a connection session
func GetConnectionSession(userID string) *ConnectionSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	return connectionSessions[userID]
}

// ClearConnectionSession removes a connection session
func ClearConnectionSession(userID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	delete(connectionSessions, userID)
}

// GetAllConnectionSessions returns all active sessions
func GetAllConnectionSessions() map[string]*ConnectionSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	
	sessions := make(map[string]*ConnectionSession)
	for k, v := range connectionSessions {
		sessions[k] = v
	}
	return sessions
}

// SetDeviceQRChannel stores QR channel for a device
func SetDeviceQRChannel(deviceID string, qrChan <-chan string) {
	qrMutex.Lock()
	defer qrMutex.Unlock()
	deviceQRChannels[deviceID] = qrChan
	
	// Start goroutine to handle QR updates
	go func() {
		for qr := range qrChan {
			// Broadcast QR update via websocket
			websocket.Broadcast <- websocket.BroadcastMessage{
				Code:    "QR_UPDATE",
				Message: "QR code updated",
				Result: map[string]interface{}{
					"deviceId": deviceID,
					"qr":       qr,
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
	case qr, ok := <-qrChan:
		if !ok {
			return "", fmt.Errorf("QR channel closed")
		}
		return qr, nil
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
		// Handle messages per device
		// You can add message handling here
	}
}

// handleDevicePairSuccess handles successful QR pairing for a device
func handleDevicePairSuccess(ctx context.Context, deviceID string, evt *events.PairSuccess) {
	log.Infof("Device %s paired successfully with %s", deviceID, evt.ID.String())
	
	// Get device connection
	dm := multidevice.GetDeviceManager()
	conn, err := dm.GetDeviceConnection(deviceID)
	if err != nil {
		log.Errorf("Failed to get device connection: %v", err)
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
	log.Infof("Device %s fully connected", deviceID)
	
	// Get device connection
	dm := multidevice.GetDeviceManager()
	conn, err := dm.GetDeviceConnection(deviceID)
	if err != nil {
		log.Errorf("Failed to get device connection: %v", err)
		return
	}
	
	if conn.Client == nil || !conn.Client.IsLoggedIn() {
		log.Warnf("Device %s connected event but client not logged in", deviceID)
		return
	}
	
	// Get WhatsApp info
	var phoneNumber, jid string
	if conn.Client.Store.ID != nil {
		jid = conn.Client.Store.ID.String()
		phoneNumber = conn.Client.Store.ID.User
		log.Infof("Device %s connected as: %s (Phone: %s)", deviceID, jid, phoneNumber)
	}
	
	// Update device in database
	userRepo := repository.GetUserRepository()
	err = userRepo.UpdateDeviceStatus(deviceID, "online", phoneNumber, jid)
	if err != nil {
		log.Errorf("Failed to update device status: %v", err)
	} else {
		log.Infof("Successfully updated device %s to online status", deviceID)
	}
	
	// Update device manager
	dm.UpdateDeviceStatus(deviceID, true, phoneNumber)
	
	// Register with client manager for broadcasts
	cm := GetClientManager()
	cm.AddClient(deviceID, conn.Client)
	log.Infof("Registered device %s with client manager", deviceID)
	
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
			log.Errorf("Failed to sync chats for device %s: %v", deviceID, err)
		} else {
			log.Infof("Successfully synced %d chats for device %s", len(chats), deviceID)
		}
	}()
}

// handleDeviceLoggedOut handles device logout
func handleDeviceLoggedOut(ctx context.Context, deviceID string) {
	log.Infof("Device %s logged out", deviceID)
	
	// Update device status
	userRepo := repository.GetUserRepository()
	err := userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
	if err != nil {
		log.Errorf("Failed to update device status: %v", err)
	}
	
	// Update device manager
	dm := multidevice.GetDeviceManager()
	dm.UpdateDeviceStatus(deviceID, false, "")
	
	// Remove from client manager
	cm := GetClientManager()
	cm.RemoveClient(deviceID)
	
	// Clear QR channel
	ClearDeviceQRChannel(deviceID)
	
	// Broadcast logout
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_LOGGED_OUT",
		Message: "Device logged out",
		Result: map[string]interface{}{
			"deviceId": deviceID,
		},
	}
}
