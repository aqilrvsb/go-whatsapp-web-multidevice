package whatsapp

import (
	"context"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// DeviceConnectionManager prevents multiple connections to same device
type DeviceConnectionManager struct {
	mu               sync.RWMutex
	activeConnections map[string]*ConnectionInfo
	connectionLocks   map[string]*sync.Mutex
}

type ConnectionInfo struct {
	Client       *whatsmeow.Client
	ConnectedAt  time.Time
	Phone        string
	JID          string
	IsConnecting bool
}

var (
	dcm  *DeviceConnectionManager
	dcmOnce sync.Once
)

// GetDeviceConnectionManager returns singleton instance
func GetDeviceConnectionManager() *DeviceConnectionManager {
	dcmOnce.Do(func() {
		dcm = &DeviceConnectionManager{
			activeConnections: make(map[string]*ConnectionInfo),
			connectionLocks:   make(map[string]*sync.Mutex),
		}
	})
	return dcm
}

// PreventDuplicateConnection checks if device is already connecting/connected
func (dcm *DeviceConnectionManager) PreventDuplicateConnection(deviceID string) bool {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	// Get or create device lock
	lock, exists := dcm.connectionLocks[deviceID]
	if !exists {
		lock = &sync.Mutex{}
		dcm.connectionLocks[deviceID] = lock
	}

	// Check if already connected or connecting
	if info, exists := dcm.activeConnections[deviceID]; exists {
		if info.IsConnecting {
			logrus.Warnf("Device %s is already connecting, preventing duplicate connection", deviceID)
			return false
		}
		if info.Client != nil && info.Client.IsConnected() {
			logrus.Warnf("Device %s is already connected, preventing duplicate connection", deviceID)
			return false
		}
	}

	// Mark as connecting
	dcm.activeConnections[deviceID] = &ConnectionInfo{
		IsConnecting: true,
		ConnectedAt:  time.Now(),
	}

	return true
}

// RegisterConnection registers successful connection
func (dcm *DeviceConnectionManager) RegisterConnection(deviceID string, client *whatsmeow.Client, phone, jid string) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	dcm.activeConnections[deviceID] = &ConnectionInfo{
		Client:       client,
		ConnectedAt:  time.Now(),
		Phone:        phone,
		JID:          jid,
		IsConnecting: false,
	}

	logrus.Infof("Registered connection for device %s", deviceID)
}

// RemoveConnection removes device from active connections
func (dcm *DeviceConnectionManager) RemoveConnection(deviceID string) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	delete(dcm.activeConnections, deviceID)
	logrus.Infof("Removed connection for device %s", deviceID)
}

// HandleConnectionSuccess sends proper WebSocket notification for QR modal close
func HandleConnectionSuccess(deviceID string, phone string, jid string) {
	// Send multiple notifications to ensure frontend receives it
	messages := []websocket.BroadcastMessage{
		{
			Code:    "LOGIN_SUCCESS",
			Message: "Successfully connected to WhatsApp",
			Result: map[string]interface{}{
				"deviceId": deviceID,
				"phone":    phone,
				"jid":      jid,
			},
		},
		{
			Code:    "DEVICE_CONNECTED",
			Message: "WhatsApp device is now online",
			Result: map[string]interface{}{
				"deviceId": deviceID,
				"phone":    phone,
				"jid":      jid,
				"status":   "online",
			},
		},
		{
			Code:    "QR_CONNECTED",
			Message: "QR code scan successful",
			Result: map[string]interface{}{
				"deviceId": deviceID,
				"success":  true,
			},
		},
	}

	// Send all messages with small delay
	for _, msg := range messages {
		websocket.Broadcast <- msg
		time.Sleep(100 * time.Millisecond)
	}

	// Update device status
	userRepo := repository.GetUserRepository()
	if userRepo != nil {
		userRepo.UpdateDeviceStatus(deviceID, "online", phone, jid)
	}
}

// HandleStreamReplaced handles when another client connects with same credentials
func HandleStreamReplaced(ctx context.Context, deviceID string, evt *events.StreamReplaced) {
	logrus.Warnf("Stream replaced for device %s - another client connected with same credentials", deviceID)
	
	dcm := GetDeviceConnectionManager()
	dcm.RemoveConnection(deviceID)
	
	// Don't try to reconnect immediately to avoid loop
	go func() {
		time.Sleep(5 * time.Second)
		
		// Check if device should reconnect
		userRepo := repository.GetUserRepository()
		device, err := userRepo.GetDeviceByID(deviceID)
		if err != nil || device == nil {
			return
		}
		
		// Only reconnect if device status is not offline
		if device.Status == "offline" {
			logrus.Debugf("Device %s is marked offline - not reconnecting", deviceID)
			return
		}
		
		// Only reconnect if device is supposed to be online
		if device.Status == "online" {
			logrus.Infof("Attempting to reclaim connection for device %s", deviceID)
			// The health monitor will handle reconnection
		}
	}()
}
