package whatsapp

import (
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// DeviceStatusNormalizer ensures all devices have proper online/offline status
type DeviceStatusNormalizer struct {
	mu       sync.Mutex
	running  bool
	userRepo *repository.UserRepository
}

var statusNormalizer *DeviceStatusNormalizer

// StartDeviceStatusNormalizer starts the status normalizer
func StartDeviceStatusNormalizer() {
	if statusNormalizer != nil && statusNormalizer.running {
		return
	}
	
	statusNormalizer = &DeviceStatusNormalizer{
		userRepo: repository.GetUserRepository(),
		running:  true,
	}
	
	go statusNormalizer.run()
	logrus.Info("Device status normalizer started")
}

// run continuously checks and normalizes device statuses
func (n *DeviceStatusNormalizer) run() {
	// Initial check after 30 seconds
	time.Sleep(30 * time.Second)
	n.normalizeAllDevices()
	
	// Then check every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for n.running {
		select {
		case <-ticker.C:
			n.normalizeAllDevices()
		}
	}
}

// normalizeAllDevices ensures all devices have proper status
func (n *DeviceStatusNormalizer) normalizeAllDevices() {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	logrus.Debug("Running device status normalization...")
	
	// Get all devices
	devices, err := n.userRepo.GetAllDevices()
	if err != nil {
		logrus.Errorf("Failed to get devices for normalization: %v", err)
		return
	}
	
	normalized := 0
	for _, device := range devices {
		// Check if status needs normalization
		if device.Status != "online" && device.Status != "offline" {
			logrus.Warnf("Normalizing device %s status from '%s' to 'offline'", device.DeviceName, device.Status)
			
			// Set to offline
			err := n.userRepo.UpdateDeviceStatus(device.ID, "offline", device.Phone, device.JID)
			if err != nil {
				logrus.Errorf("Failed to normalize device %s: %v", device.DeviceName, err)
			} else {
				normalized++
			}
		}
	}
	
	if normalized > 0 {
		logrus.Infof("Normalized %d device statuses to offline", normalized)
	}
}

// StopDeviceStatusNormalizer stops the normalizer
func StopDeviceStatusNormalizer() {
	if statusNormalizer != nil {
		statusNormalizer.running = false
		logrus.Info("Device status normalizer stopped")
	}
}
