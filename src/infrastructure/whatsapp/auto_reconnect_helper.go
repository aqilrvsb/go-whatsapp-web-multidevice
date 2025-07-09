package whatsapp

import (
	"context"
	"time"
	
	"github.com/sirupsen/logrus"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// AutoReconnectOnlineDevices attempts to reconnect all previously online devices
func AutoReconnectOnlineDevices() {
	logrus.Info("=== Starting Auto-Reconnect for Online Devices ===")
	
	// Initialize WhatsApp container using PostgreSQL
	waLogger := waLog.Stdout("Database", config.WhatsappLogLevel, true)
	
	// Use PostgreSQL connection string
	container, err := sqlstore.New(context.Background(), "postgres", config.DBURI, waLogger)
	if err != nil {
		logrus.Errorf("Failed to create WhatsApp container: %v", err)
		return
	}
	
	// Call the original function with container
	AutoReconnectDevices(container)
}

// RefreshDeviceConnectionByID attempts to refresh a specific device connection
func RefreshDeviceConnectionByID(deviceID string) error {
	logrus.Infof("🔄 Starting device refresh for %s", deviceID)
	
	// Get user repo
	userRepo := repository.GetUserRepository()
	
	// Get device info
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get device info: %v", err)
		return err
	}
	
	// Check if device has JID
	if device.JID == "" {
		logrus.Warnf("Device %s has no JID, cannot reconnect", device.DeviceName)
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
		return nil
	}
	
	// Initialize WhatsApp container
	waLogger := waLog.Stdout("Database", config.WhatsappLogLevel, true)
	container, err := sqlstore.New(context.Background(), "postgres", config.DBURI, waLogger)
	if err != nil {
		logrus.Errorf("Failed to create WhatsApp container: %v", err)
		return err
	}
	
	// Try to get the device from store
	waDevice, err := container.GetDevice(context.Background(), parseJID(device.JID))
	if err != nil {
		logrus.Errorf("Failed to get device from store: %v", err)
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
		return err
	}
	
	// Create client
	client := whatsmeow.NewClient(waDevice, waLogger)
	
	// Add event handler
	client.AddEventHandler(func(evt interface{}) {
		HandleDeviceEvent(context.Background(), deviceID, evt)
	})
	
	// Try to connect
	logrus.Infof("Connecting device %s...", device.DeviceName)
	err = client.Connect()
	if err != nil {
		logrus.Errorf("Failed to connect device %s: %v", device.DeviceName, err)
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
		return err
	}
	
	// Wait for connection
	time.Sleep(3 * time.Second)
	
	// Verify connection status
	if !client.IsConnected() {
		logrus.Warnf("Device %s failed to establish connection", device.DeviceName)
		client.Disconnect()
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
		return nil
	}
	
	// Check if logged in
	if !client.IsLoggedIn() {
		logrus.Warnf("Device %s connected but not logged in - session expired", device.DeviceName)
		client.Disconnect()
		userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
		return nil
	}
	
	// Success! Register with client manager
	cm := GetClientManager()
	cm.AddClient(deviceID, client)
	
	// Update device status
	actualPhone := device.Phone
	actualJID := device.JID
	if client.Store.ID != nil {
		actualPhone = client.Store.ID.User
		actualJID = client.Store.ID.String()
	}
	
	err = userRepo.UpdateDeviceStatus(deviceID, "online", actualPhone, actualJID)
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	logrus.Infof("✅ Successfully reconnected device %s (%s)", device.DeviceName, deviceID)
	
	return nil
}

// parseJID converts a string JID to types.JID
func parseJID(jid string) types.JID {
	parsed, _ := types.ParseJID(jid)
	return parsed
}
