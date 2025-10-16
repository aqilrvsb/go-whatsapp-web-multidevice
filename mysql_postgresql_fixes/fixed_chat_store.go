package whatsapp

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// StoreChat stores a single chat in the database
func StoreChat(deviceID, chatJID, name string, lastMessageTime time.Time) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Validate inputs
	if deviceID == "" || chatJID == "" {
		return fmt.Errorf("invalid input: deviceID or chatJID is empty")
	}

	// Parse JID to ensure it's valid
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid JID format: %v", err)
	}

	// Get a better name if empty
	if name == "" {
		if jid.Server == types.DefaultUserServer {
			// For personal chats, use the phone number
			name = jid.User
		} else {
			name = "Unknown"
		}
		logrus.Warnf("Empty name for chat %s, using fallback: %s", chatJID, name)
	}

	// Check if we're using MySQL or PostgreSQL
	dbType := "mysql"
	if dbURI := os.Getenv("MYSQL_URI"); dbURI == "" {
		dbURI = os.Getenv("DB_URI")
		if dbURI == "" || strings.Contains(dbURI, "postgres") {
			dbType = "postgres"
		}
	}

	var query string
	if dbType == "mysql" {
		// MySQL syntax - fixed without line break issues
		query = `INSERT INTO whatsapp_chats (device_id, chat_jid, chat_name, last_message_time) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE chat_name = VALUES(chat_name), last_message_time = VALUES(last_message_time)`
	} else {
		// PostgreSQL syntax
		query = `INSERT INTO whatsapp_chats (device_id, chat_jid, chat_name, last_message_time) VALUES ($1, $2, $3, $4) ON CONFLICT (device_id, chat_jid) DO UPDATE SET chat_name = EXCLUDED.chat_name, last_message_time = EXCLUDED.last_message_time`
	}

	_, err = db.Exec(query, deviceID, chatJID, name, lastMessageTime)
	if err != nil {
		logrus.Errorf("Failed to store chat: %v", err)
	} else {
		logrus.Debugf("Stored/updated chat: %s (%s)", name, chatJID)
	}
	return err
}
