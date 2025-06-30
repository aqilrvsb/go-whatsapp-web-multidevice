package usecase

import (
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/sirupsen/logrus"
)

// StartQueuedMessageCleaner handles stuck messages in queued state
func StartQueuedMessageCleaner() {
	logrus.Info("Starting queued message cleaner...")
	
	ticker := time.NewTicker(60 * time.Second) // Check every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cleanStuckMessages()
		}
	}
}

func cleanStuckMessages() {
	db := database.GetDB()
	
	// Find messages that have been queued for more than 5 minutes
	// These are likely stuck due to worker crash or other issues
	result, err := db.Exec(`
		UPDATE broadcast_messages 
		SET status = 'pending', 
		    updated_at = CURRENT_TIMESTAMP,
		    error_message = 'Reset from stuck queued state'
		WHERE status = 'queued' 
		AND updated_at < (CURRENT_TIMESTAMP - INTERVAL '5 minutes')
	`)
	
	if err != nil {
		logrus.Errorf("Failed to clean stuck messages: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logrus.Warnf("Reset %d stuck messages from 'queued' back to 'pending'", rowsAffected)
	}
}
