package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp/multidevice"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// DeviceHealthMonitor monitors device health and handles reconnections
type DeviceHealthMonitor struct {
	mu              sync.RWMutex
	monitorInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	db              *sqlstore.Container
	deviceStates    map[string]*DeviceState // Track device connection states
}

// DeviceState tracks the connection state of a device
type DeviceState struct {
	LastSeen         time.Time
	ConsecutiveFails int
	IsReconnecting   bool
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
			deviceStates:    make(map[string]*DeviceState),
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
func (dhm *DeviceHealthMonitor) checkDeviceHealth(deviceID string, client *whatsmeow.Client, userRepo *repository.UserRepository) {
	// First check if this is a platform device
	device, err := userRepo.GetDeviceByID(deviceID)
	if err == nil && device.Platform != "" {
		// Skip health check for platform devices (Wablas, Whacenter, etc)
		return
	}
	
	// Get or create device state first
	dhm.mu.Lock()
	state, exists := dhm.deviceStates[deviceID]
	if !exists {
		state = &DeviceState{
			LastSeen:         time.Now(),
			ConsecutiveFails: 0,
			IsReconnecting:   false,
		}
		dhm.deviceStates[deviceID] = state
	}
	dhm.mu.Unlock()
	
	if client == nil {
		logrus.Warnf("Device %s has nil client, attempting to restore connection", deviceID)
		// Try to get client from DeviceManager
		dm := multidevice.GetDeviceManager()
		if conn, err := dm.GetDeviceConnection(deviceID); err == nil && conn.Client != nil {
			client = conn.Client
			logrus.Infof("Successfully restored client for device %s from DeviceManager", deviceID)
		} else {
			// Mark for reconnection but DON'T remove
			logrus.Warnf("Could not restore client for device %s, will retry later", deviceID)
			state.ConsecutiveFails = 3 // Trigger reconnect attempt
			return
		}
	}
	
	// Check if client is connected
	if !client.IsConnected() {
		state.ConsecutiveFails++
		
		// Only log after multiple failures with better messages
		if state.ConsecutiveFails == 1 {
			logrus.Debugf("Device %s disconnected (check %d/1440 before removal)", deviceID, state.ConsecutiveFails)
		} else if state.ConsecutiveFails == 3 {
			logrus.Warnf("Device %s disconnected for %d checks, will attempt reconnection", deviceID, state.ConsecutiveFails)
		} else if state.ConsecutiveFails == 10 {
			logrus.Warnf("Device %s still disconnected after %d checks (5 minutes)", deviceID, state.ConsecutiveFails)
		} else if state.ConsecutiveFails == 120 {
			logrus.Errorf("Device %s disconnected for 1 hour (%d checks)", deviceID, state.ConsecutiveFails)
		} else if state.ConsecutiveFails == 720 {
			logrus.Errorf("Device %s disconnected for 6 hours (%d checks) - will remove after 12 hours", deviceID, state.ConsecutiveFails)
		}
		
		// Give device 12 hours (1440 checks) before removing
		// But try to reconnect after 3 failures (1.5 minutes)
		if state.ConsecutiveFails == 3 {
			logrus.Infof("Device %s disconnected for 3 checks, attempting reconnection", deviceID)
			go func() {
				if err := client.Connect(); err != nil {
					logrus.Warnf("Failed to reconnect device %s: %v", deviceID, err)
				} else {
					logrus.Infof("Successfully reconnected device %s", deviceID)
					// Update status to online
					userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, device.JID)
				}
			}()
		} else if state.ConsecutiveFails == 10 {
			logrus.Warnf("Device %s disconnected for 5 minutes, trying reconnect again", deviceID)
			go func() {
				client.Disconnect() // Force disconnect first
				time.Sleep(2 * time.Second)
				if err := client.Connect(); err != nil {
					logrus.Errorf("Second reconnect attempt failed for device %s: %v", deviceID, err)
				}
			}()
		} else if state.ConsecutiveFails >= 1440 { // 12 hours
			timeSinceLastSeen := time.Since(state.LastSeen)
			logrus.Errorf("Device %s has been disconnected for %v (12 hours), removing from system", deviceID, timeSinceLastSeen)
			
			// NOW we remove after 12 hours
			cm := GetClientManager()
			cm.RemoveClient(deviceID)
			userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
			
			// Stop keepalive
			km := GetKeepaliveManager()
			km.StopKeepalive(deviceID)
		}
	} else if !client.IsLoggedIn() {
		logrus.Warnf("Device %s is connected but not logged in", deviceID)
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
	} else {
		// Device is healthy, reset failure count
		if state.ConsecutiveFails > 0 {
			logrus.Infof("Device %s reconnected after %d failed checks", deviceID, state.ConsecutiveFails)
		}
		state.ConsecutiveFails = 0
		state.LastSeen = time.Now()
		
		// Ensure status is correct
		device, err := userRepo.GetDeviceByID(deviceID)
		if err == nil && device.Status != "online" {
			userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, device.JID)
		}
	}
}

// reconnectDevice - DISABLED - We only check status now
/*
func (dhm *DeviceHealthMonitor) reconnectDevice(deviceID string) error {
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}
	
	// Skip reconnection for platform devices
	if device.Platform != "" {
		return nil
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
		// Process asynchronously
		go HandleDeviceEvent(context.Background(), deviceID, evt)
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
*/

// ManualReconnectDevice manually triggers device reconnection
func (dhm *DeviceHealthMonitor) ManualReconnectDevice(deviceID string) error {
	logrus.Infof("Manual reconnection DISABLED - only status check available")
	return fmt.Errorf("reconnection disabled - use refresh button instead")
}

// ReconnectAllOfflineDevices - DISABLED - only status check available
func (dhm *DeviceHealthMonitor) ReconnectAllOfflineDevices() (int, int) {
	logrus.Info("Reconnect all devices DISABLED - only status check available")
	return 0, 0
}