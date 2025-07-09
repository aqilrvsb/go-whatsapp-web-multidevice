package whatsapp

import (
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure"
	"github.com/sirupsen/logrus"
)

// ErrorMonitor monitors for specific errors and triggers auto-fixes
type ErrorMonitor struct {
	lastErrors map[string]time.Time
}

var errorMonitor *ErrorMonitor

// InitErrorMonitor initializes the error monitoring system
func InitErrorMonitor() {
	errorMonitor = &ErrorMonitor{
		lastErrors: make(map[string]time.Time),
	}
	
	// Set up logrus hook to monitor all error logs
	logrus.AddHook(&errorHook{monitor: errorMonitor})
	
	logrus.Info("Error monitor initialized - will auto-refresh devices on connection errors")
}

// errorHook implements logrus.Hook
type errorHook struct {
	monitor *ErrorMonitor
}

func (hook *errorHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}

func (hook *errorHook) Fire(entry *logrus.Entry) error {
	message := entry.Message
	
	// Check for device connection error
	if strings.Contains(message, "Failed to get device connection: no device connection found for device") {
		// Extract device ID
		parts := strings.Split(message, "device ")
		if len(parts) >= 2 {
			deviceID := strings.TrimSpace(parts[len(parts)-1])
			
			// Remove any trailing quotes or punctuation
			deviceID = strings.Trim(deviceID, "\"'")
			
			// Check if we already triggered refresh recently (within 5 minutes)
			if lastTime, exists := hook.monitor.lastErrors[deviceID]; exists {
				if time.Since(lastTime) < 5*time.Minute {
					// Don't trigger again too soon
					return nil
				}
			}
			
			// Record this error
			hook.monitor.lastErrors[deviceID] = time.Now()
			
			// Trigger auto-refresh
			logrus.Infof("🔄 Auto-refresh triggered by error monitor for device: %s", deviceID)
			infrastructure.TriggerDeviceRefresh(deviceID)
			
			// Clean up old entries (older than 1 hour)
			for id, timestamp := range hook.monitor.lastErrors {
				if time.Since(timestamp) > time.Hour {
					delete(hook.monitor.lastErrors, id)
				}
			}
		}
	}
	
	return nil
}

// MonitorDeviceErrors starts monitoring for device connection errors
func MonitorDeviceErrors() {
	if errorMonitor == nil {
		InitErrorMonitor()
	}
}
