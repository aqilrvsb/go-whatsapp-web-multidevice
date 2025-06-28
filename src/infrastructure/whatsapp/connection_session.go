package whatsapp

import (
	"sync"
)

// ConnectionSession stores information about a device connection attempt
type ConnectionSession struct {
	UserID   string
	DeviceID string
}

var (
	connectionSessions = make(map[string]*ConnectionSession) // userID -> session
	sessionMutex       sync.RWMutex
)

// StoreConnectionSession stores a connection session
func StoreConnectionSession(userID string, session *ConnectionSession) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	connectionSessions[userID] = session
}

// GetConnectionSession retrieves a connection session
func GetConnectionSession(userID string) *ConnectionSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	return connectionSessions[userID]
}

// GetAllConnectionSessions returns all active sessions
func GetAllConnectionSessions() map[string]*ConnectionSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	
	// Create a copy
	sessions := make(map[string]*ConnectionSession)
	for k, v := range connectionSessions {
		sessions[k] = v
	}
	return sessions
}

// ClearConnectionSession removes a connection session
func ClearConnectionSession(userID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	delete(connectionSessions, userID)
}
