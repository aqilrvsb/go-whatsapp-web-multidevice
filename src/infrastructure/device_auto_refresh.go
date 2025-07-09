package infrastructure

import (
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// DeviceAutoRefreshHandler handles device auto-refresh globally
type DeviceAutoRefreshHandler struct {
	refreshing map[string]bool
	userRepo   *repository.UserRepository
}

var globalRefreshHandler *DeviceAutoRefreshHandler

// InitDeviceAutoRefresh initializes the global refresh handler
func InitDeviceAutoRefresh() {
	globalRefreshHandler = &DeviceAutoRefreshHandler{
		refreshing: make(map[string]bool),
		userRepo:   repository.GetUserRepository(),
	}
	logrus.Info("Device auto-refresh handler initialized")
}

// TriggerDeviceRefresh triggers auto-refresh for a device
func TriggerDeviceRefresh(deviceID string) {
	if globalRefreshHandler == nil {
		InitDeviceAutoRefresh()
	}
	
	// Check if already refreshing
	if globalRefreshHandler.refreshing[deviceID] {
		logrus.Debugf("Device %s is already being refreshed", deviceID)
		return
	}
	
	globalRefreshHandler.refreshing[deviceID] = true
	
	// Run refresh in background
	go func() {
		defer func() {
			delete(globalRefreshHandler.refreshing, deviceID)
		}()
		
		logrus.Infof("🔄 Auto-refresh triggered for device %s", deviceID)
		
		// Get device info
		device, err := globalRefreshHandler.userRepo.GetDeviceByID(deviceID)
		if err != nil {
			logrus.Errorf("Failed to get device info for auto-refresh: %v", err)
			return
		}
		
		// Log the refresh attempt
		logrus.Infof("Attempting to refresh device %s (%s)", device.DeviceName, device.JID)
		
		// Mark device as refreshing in database (optional)
		globalRefreshHandler.userRepo.UpdateDeviceStatus(deviceID, "refreshing", device.Phone, device.JID)
		
		// The actual reconnection will be handled by the main connection logic
		// when it detects the "refreshing" status
		
		// After a delay, if still offline, mark as offline
		time.Sleep(30 * time.Second)
		
		// Check if device is now online
		updatedDevice, err := globalRefreshHandler.userRepo.GetDeviceByID(deviceID)
		if err == nil && updatedDevice.Status != "online" {
			globalRefreshHandler.userRepo.UpdateDeviceStatus(deviceID, "offline", device.Phone, device.JID)
			logrus.Warnf("Auto-refresh timeout for device %s - marked as offline", device.DeviceName)
		}
	}()
}

// IsDeviceRefreshing checks if a device is currently being refreshed
func IsDeviceRefreshing(deviceID string) bool {
	if globalRefreshHandler == nil {
		return false
	}
	return globalRefreshHandler.refreshing[deviceID]
}
