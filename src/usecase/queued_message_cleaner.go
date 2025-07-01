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
	// Mark them as failed instead of pending to avoid infinite loops
	result, err := db.Exec(`
		UPDATE broadcast_messages 
		SET status = 'failed', 
		    updated_at = CURRENT_TIMESTAMP,
		    error_message = 'Message timeout - device was not available'
		WHERE status = 'queued' 
		AND updated_at < (CURRENT_TIMESTAMP - INTERVAL '5 minutes')
	`)
	
	if err != nil {
		logrus.Errorf("Failed to clean stuck messages: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logrus.Warnf("Marked %d stuck queued messages as failed after timeout", rowsAffected)
	}
}
