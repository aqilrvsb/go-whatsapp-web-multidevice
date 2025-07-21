package whatsapp

import (
	"strings"
	"sync"
	
	"github.com/sirupsen/logrus"
)

// DeviceIDValidator ensures only valid device IDs are processed
type DeviceIDValidator struct {
	mu         sync.RWMutex
	validIDs   map[string]bool
}

var (
	validator     *DeviceIDValidator
	validatorOnce sync.Once
)

// GetDeviceIDValidator returns singleton validator
func GetDeviceIDValidator() *DeviceIDValidator {
	validatorOnce.Do(func() {
		validator = &DeviceIDValidator{
			validIDs: make(map[string]bool),
		}
	})
	return validator
}

// IsValidDeviceID checks if a device ID is valid UUID format
func IsValidDeviceID(deviceID string) bool {
	// Check basic length (UUID is 36 chars)
	if len(deviceID) != 36 {
		return false
	}
	
	// Check format: 8-4-4-4-12
	parts := strings.Split(deviceID, "-")
	if len(parts) != 5 {
		return false
	}
	
	// Check each part length
	if len(parts[0]) != 8 || len(parts[1]) != 4 || 
	   len(parts[2]) != 4 || len(parts[3]) != 4 || 
	   len(parts[4]) != 12 {
		return false
	}
	
	// Check if it contains invalid prefixes
	if strings.Contains(deviceID, "check-connection") ||
	   strings.Contains(deviceID, "unitiesb-") ||
	   strings.Contains(deviceID, "/") {
		return false
	}
	
	return true
}

// RegisterValidDevice registers a valid device ID
func (v *DeviceIDValidator) RegisterValidDevice(deviceID string) {
	if !IsValidDeviceID(deviceID) {
		logrus.Warnf("Attempted to register invalid device ID: %s", deviceID)
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validIDs[deviceID] = true
	logrus.Debugf("Registered valid device ID: %s", deviceID)
}

// IsKnownValidDevice checks if device was previously validated
func (v *DeviceIDValidator) IsKnownValidDevice(deviceID string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.validIDs[deviceID]
}

// CleanCorruptedDeviceID attempts to extract valid UUID from corrupted ID
func CleanCorruptedDeviceID(deviceID string) string {
	// If already valid, return as is
	if IsValidDeviceID(deviceID) {
		return deviceID
	}
	
	// Try to find UUID pattern in the string
	// UUID pattern: 8-4-4-4-12 hex characters
	if idx := strings.Index(deviceID, "-"); idx > 0 {
		// Check if we can find a valid UUID starting from different positions
		for i := 0; i < len(deviceID)-35; i++ {
			candidate := deviceID[i:]
			if len(candidate) >= 36 {
				candidate = candidate[:36]
				if IsValidDeviceID(candidate) {
					logrus.Infof("Extracted valid UUID %s from corrupted ID %s", candidate, deviceID)
					return candidate
				}
			}
		}
	}
	
	// Could not extract valid UUID
	return ""
}

// SafeClientManagerOperation performs operations only on valid device IDs
func SafeClientManagerOperation(deviceID string, operation func(string)) {
	// First try to clean the device ID
	cleanID := CleanCorruptedDeviceID(deviceID)
	if cleanID != "" {
		operation(cleanID)
		return
	}
	
	// If it's a known corrupted pattern, ignore it
	if strings.Contains(deviceID, "check-connection") {
		logrus.Debugf("Ignoring corrupted device ID from check-connection: %s", deviceID)
		return
	}
	
	// Log unknown invalid patterns for debugging
	logrus.Warnf("Skipping operation for invalid device ID: %s", deviceID)
}
