package tracker

import (
	"sync"
	"time"
)

// LoggedOutDevices tracks devices that were intentionally logged out
// to prevent auto-reconnection
type LoggedOutTracker struct {
	mu           sync.RWMutex
	loggedOut    map[string]time.Time // deviceID -> logout time
	cleanupTimer *time.Timer
}

var (
	logoutTracker     *LoggedOutTracker
	logoutTrackerOnce sync.Once
)

// GetLogoutTracker returns singleton instance
func GetLogoutTracker() *LoggedOutTracker {
	logoutTrackerOnce.Do(func() {
		logoutTracker = &LoggedOutTracker{
			loggedOut: make(map[string]time.Time),
		}
		// Start cleanup routine
		logoutTracker.startCleanup()
	})
	return logoutTracker
}

// MarkLoggedOut marks a device as intentionally logged out
func (lt *LoggedOutTracker) MarkLoggedOut(deviceID string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.loggedOut[deviceID] = time.Now()
}

// RemoveLoggedOut removes the logged out flag (when user wants to reconnect)
func (lt *LoggedOutTracker) RemoveLoggedOut(deviceID string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	delete(lt.loggedOut, deviceID)
}

// IsLoggedOut checks if device was intentionally logged out
func (lt *LoggedOutTracker) IsLoggedOut(deviceID string) bool {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	
	logoutTime, exists := lt.loggedOut[deviceID]
	if !exists {
		return false
	}
	
	// Keep logout state for 24 hours, then allow reconnection
	if time.Since(logoutTime) > 24*time.Hour {
		delete(lt.loggedOut, deviceID)
		return false
	}
	
	return true
}

// startCleanup periodically cleans up old logout entries
func (lt *LoggedOutTracker) startCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			lt.cleanupOldEntries()
		}
	}()
}

// cleanupOldEntries removes entries older than 24 hours
func (lt *LoggedOutTracker) cleanupOldEntries() {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	now := time.Now()
	for deviceID, logoutTime := range lt.loggedOut {
		if now.Sub(logoutTime) > 24*time.Hour {
			delete(lt.loggedOut, deviceID)
		}
	}
}
