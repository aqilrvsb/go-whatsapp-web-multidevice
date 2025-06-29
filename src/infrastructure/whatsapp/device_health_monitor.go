package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// DeviceHealthMonitor monitors device health and handles reconnections
type DeviceHealthMonitor struct {
	mu              sync.RWMutex
	monitorInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	db              *sqlstore.Container
}

var (
	healthMonitor     *DeviceHealthMonitor
	healthMonitorOnce sync.Once
)

// GetDeviceHealthMonitor returns singleton instance
func GetDeviceHealthMonitor(db *sqlstore.Container) *DeviceHealthMonitor {
	healthMonitorOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		healthMonitor = &DeviceHealthMonitor{
			monitorInterval: 30 * time.Second,
			ctx:             ctx,
			cancel:          cancel,
			db:              db,
		}
	})
	return healthMonitor
}

// Start begins monitoring device health
func (dhm *DeviceHealthMonitor) Start() {
	go dhm.monitorLoop()
	logrus.Info("Device health monitor started")
}

// Stop stops the health monitor
func (dhm *DeviceHealthMonitor) Stop() {
	if dhm.cancel != nil {
		dhm.cancel()
	}
	logrus.Info("Device health monitor stopped")
}

// monitorLoop continuously checks device health
func (dhm *DeviceHealthMonitor) monitorLoop() {
	ticker := time.NewTicker(dhm.monitorInterval)
	defer ticker.Stop()
	
	// Initial check after 10 seconds
	time.Sleep(10 * time.Second)
	dhm.checkAllDevices()
	
	for {
		select {
		case <-dhm.ctx.Done():
			return
		case <-ticker.C:
			dhm.checkAllDevices()
		}
	}
}

// checkAllDevices checks health of all registered devices
func (dhm *DeviceHealthMonitor) checkAllDevices() {
	cm := GetClientManager()
	allClients := cm.GetAllClients()
	
	userRepo := repository.GetUserRepository()
	
	for deviceID, client := range allClients {
		go dhm.checkDeviceHealth(deviceID, client, userRepo)
	}
}

// checkDeviceHealth checks health of a single device
func (dhm *DeviceHealthMonitor) checkDeviceHealth(deviceID string, client *whatsmeow.Client, userRepo repository.IUserRepository) {
	if client == nil {
		logrus.Warnf("Device %s has nil client, removing from manager", deviceID)
		cm := GetClientManager()
		cm.RemoveClient(deviceID)
		userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
		return
	}
	
	// Check if client is connected
	if !client.IsConnected() {
		logrus.Warnf("Device %s is disconnected, attempting reconnection", deviceID)
		
		// Update status to disconnected
		userRepo.UpdateDeviceStatus(deviceID, "disconnected", "", "")
		
		// Try to reconnect
		if err := dhm.reconnectDevice(deviceID); err != nil {
			logrus.Errorf("Failed to reconnect device %s: %v", deviceID, err)
			userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
		}
	} else if !client.IsLoggedIn() {
		logrus.Warnf("Device %s is connected but not logged in", deviceID)
		userRepo.UpdateDeviceStatus(deviceID, "disconnected", "", "")
	} else {
		// Device is healthy, ensure status is correct
		device, err := userRepo.GetDeviceByID(deviceID)
		if err == nil && device.Status != "online" {
			userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, device.JID)
		}
	}
}

// reconnectDevice attempts to reconnect a disconnected device
func (dhm *DeviceHealthMonitor) reconnectDevice(deviceID string) error {
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}
	
	if device.JID == "" {
		return fmt.Errorf("device has no JID, needs QR scan")
	}
	
	// Find WhatsApp device by JID
	devices, err := dhm.db.GetAllDevices(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get devices: %v", err)
	}
	
	var waDevice *store.Device
	for _, d := range devices {
		if d.ID.String() == device.JID {
			waDevice = d
			break
		}
	}
	
	if waDevice == nil {
		return fmt.Errorf("session not found, needs QR scan")
	}
	
	// Create new client
	client := whatsmeow.NewClient(waDevice, waLog.Stdout("Device_"+deviceID, config.WhatsappLogLevel, true))
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true
	
	// Add event handlers
	client.AddEventHandler(func(evt interface{}) {
		HandleDeviceEvent(context.Background(), deviceID, evt)
	})
	
	// Connect
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	
	// Wait for connection
	connected := false
	for i := 0; i < 30; i++ { // 3 seconds timeout
		if client.IsConnected() && client.IsLoggedIn() {
			connected = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	if !connected {
		client.Disconnect()
		return fmt.Errorf("connection timeout")
	}
	
	// Register with ClientManager
	cm := GetClientManager()
	cm.AddClient(deviceID, client)
	
	// Update status
	userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, device.JID)
	logrus.Infof("Device %s reconnected successfully", deviceID)
	
	return nil
}

// ManualReconnectDevice manually triggers device reconnection
func (dhm *DeviceHealthMonitor) ManualReconnectDevice(deviceID string) error {
	logrus.Infof("Manual reconnection requested for device %s", deviceID)
	return dhm.reconnectDevice(deviceID)
}

// ReconnectAllOfflineDevices attempts to reconnect all offline devices
func (dhm *DeviceHealthMonitor) ReconnectAllOfflineDevices() (int, int) {
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetAllDevices()
	if err != nil {
		logrus.Errorf("Failed to get devices: %v", err)
		return 0, 0
	}
	
	successful := 0
	failed := 0
	
	for _, device := range devices {
		if device.Status == "offline" || device.Status == "disconnected" {
			err := dhm.reconnectDevice(device.ID)
			if err != nil {
				failed++
				logrus.Errorf("Failed to reconnect device %s: %v", device.ID, err)
			} else {
				successful++
			}
		}
	}
	
	return successful, failed
}