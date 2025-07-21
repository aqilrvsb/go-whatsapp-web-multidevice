package whatsapp

import (
	"sync"
	"time"
	
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	websocket "github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
)

// DeviceConnectionMonitor ensures devices stay connected
type DeviceConnectionMonitor struct {
	mu       sync.RWMutex
	running  bool
	interval time.Duration
}

var (
	deviceMonitor     *DeviceConnectionMonitor
	deviceMonitorOnce sync.Once
)

// GetDeviceConnectionMonitor returns singleton monitor
func GetDeviceConnectionMonitor() *DeviceConnectionMonitor {
	deviceMonitorOnce.Do(func() {
		deviceMonitor = &DeviceConnectionMonitor{
			interval: 30 * time.Second,
		}
	})
	return deviceMonitor
}

// Start begins monitoring all devices
func (dcm *DeviceConnectionMonitor) Start() {
	dcm.mu.Lock()
	if dcm.running {
		dcm.mu.Unlock()
		return
	}
	dcm.running = true
	dcm.mu.Unlock()
	
	go dcm.monitorLoop()
	logrus.Info("Started device connection monitor")
}

// Stop halts monitoring
func (dcm *DeviceConnectionMonitor) Stop() {
	dcm.mu.Lock()
	dcm.running = false
	dcm.mu.Unlock()
}

// monitorLoop runs the monitoring process
func (dcm *DeviceConnectionMonitor) monitorLoop() {
	// Initial check after 10 seconds
	time.Sleep(10 * time.Second)
	dcm.checkAllDevices()
	
	ticker := time.NewTicker(dcm.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			dcm.mu.RLock()
			if !dcm.running {
				dcm.mu.RUnlock()
				return
			}
			dcm.mu.RUnlock()
			
			dcm.checkAllDevices()
		}
	}
}

// checkAllDevices verifies all devices are properly connected
func (dcm *DeviceConnectionMonitor) checkAllDevices() {
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetAllDevices()
	if err != nil {
		logrus.Errorf("Failed to get devices for monitoring: %v", err)
		return
	}
	
	cm := GetClientManager()
	
	for _, device := range devices {
		// Skip platform devices
		if device.Platform != "" {
			continue
		}
		
		// Skip offline devices
		if device.Status != "online" {
			continue
		}
		
		// Check if client exists and is connected
		client, err := cm.GetClient(device.ID)
		if err != nil {
			logrus.Warnf("Device %s marked as online but not in ClientManager", device.ID)
			// Update status to offline
			userRepo.UpdateDeviceStatus(device.ID, "offline", device.Phone, device.JID)
			continue
		}
		
		if client == nil {
			logrus.Warnf("Device %s has nil client", device.ID)
			cm.RemoveClient(device.ID)
			userRepo.UpdateDeviceStatus(device.ID, "offline", device.Phone, device.JID)
			continue
		}
		
		// Check connection status
		if !client.IsConnected() || !client.IsLoggedIn() {
			logrus.Warnf("Device %s disconnected, updating status", device.ID)
			userRepo.UpdateDeviceStatus(device.ID, "offline", device.Phone, device.JID)
			
			// Notify via websocket
			websocket.Broadcast <- websocket.BroadcastMessage{
				Code:    "DEVICE_DISCONNECTED",
				Message: "Device disconnected",
				Result: map[string]interface{}{
					"deviceId": device.ID,
					"phone":    device.Phone,
					"status":   "offline",
				},
			}
		}
	}
}

// EnsureDeviceRegistered ensures a device is properly registered in ClientManager
func EnsureDeviceRegistered(deviceID string, client *whatsmeow.Client) {
	if client == nil {
		logrus.Errorf("Cannot register nil client for device %s", deviceID)
		return
	}
	
	cm := GetClientManager()
	
	// Check if already registered
	existingClient, _ := cm.GetClient(deviceID)
	if existingClient != nil && existingClient == client {
		// Already registered with same client
		return
	}
	
	// Register the client
	cm.AddClient(deviceID, client)
	logrus.Infof("Ensured device %s is registered in ClientManager", deviceID)
	
	// Start chat sync in background
	go func() {
		time.Sleep(2 * time.Second)
		_, err := GetChatsForDevice(deviceID)
		if err != nil {
			logrus.Errorf("Failed to sync chats for device %s: %v", deviceID, err)
		}
	}()
}
