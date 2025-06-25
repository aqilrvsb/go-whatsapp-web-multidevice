package whatsapp

import (
	"fmt"
	"sync"
	"time"
)

// Global variables to track current connection session
var (
	// Map of WhatsApp JID to user device info
	connectionSessions = make(map[string]*ConnectionSession)
	sessionMutex       sync.RWMutex
)

// ConnectionSession tracks a device being connected
type ConnectionSession struct {
	UserID     string
	DeviceID   string
	DeviceName string
	StartTime  int64
}

// StartConnectionSession starts tracking a new connection attempt
func StartConnectionSession(userID, deviceID, deviceName string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	
	// Store session info that we'll use when connection succeeds
	connectionSessions[userID] = &ConnectionSession{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		StartTime:  time.Now().Unix(),
	}
	
	// Log for debugging
	fmt.Printf("Started connection session: UserID=%s, DeviceID=%s, Total sessions=%d\n", 
		userID, deviceID, len(connectionSessions))
}

// GetConnectionSession retrieves session info for a connected device
func GetConnectionSession(jid string) (*ConnectionSession, bool) {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	
	// Try to find session by checking all users
	// In production, we'd map JID -> session directly
	for _, session := range connectionSessions {
		return session, true
	}
	
	return nil, false
}

// GetAllConnectionSessions returns all active connection sessions (for debugging)
func GetAllConnectionSessions() map[string]*ConnectionSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	
	// Create a copy to avoid concurrent access issues
	sessionsCopy := make(map[string]*ConnectionSession)
	for k, v := range connectionSessions {
		sessionsCopy[k] = v
	}
	return sessionsCopy
}

// ClearConnectionSession removes a session after successful connection
func ClearConnectionSession(userID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	
	delete(connectionSessions, userID)
}
