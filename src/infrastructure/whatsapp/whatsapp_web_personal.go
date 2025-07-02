package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/appstate"
)

// GetWhatsAppWebChats gets recent chats based on messages in our database
func GetWhatsAppWebChats(deviceID string) ([]map[string]interface{}, error) {
	logrus.Infof("=== GetWhatsAppWebChats called for device: %s ===", deviceID)
	
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		logrus.Errorf("Failed to get client: %v", err)
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	if client.Store.ID == nil {
		logrus.Error("Client store ID is nil")
		return nil, fmt.Errorf("device not logged in")
	}
	
	logrus.Infof("Client connected, JID: %s", client.Store.ID.String())
	
	// Get database connection
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// First, let's check if the table exists and has data
	var tableExists bool
	checkTableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'whatsapp_messages'
		)
	`
	err = db.QueryRow(checkTableQuery).Scan(&tableExists)
	logrus.Infof("Table 'whatsapp_messages' exists: %v", tableExists)
	
	if !tableExists {
		// Create table
		createTableQuery := `
			CREATE TABLE IF NOT EXISTS whatsapp_messages (
				id SERIAL PRIMARY KEY,
				device_id TEXT NOT NULL,
				chat_jid TEXT NOT NULL,
				message_id TEXT NOT NULL,
				sender_jid TEXT NOT NULL,
				message_text TEXT,
				message_type TEXT DEFAULT 'text',
				timestamp BIGINT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(device_id, message_id)
			);
			
			CREATE INDEX IF NOT EXISTS idx_device_chat_time ON whatsapp_messages(device_id, chat_jid, timestamp DESC);
		`
		
		_, err = db.Exec(createTableQuery)
		if err != nil {
			logrus.Errorf("Failed to create table: %v", err)
		} else {
			logrus.Info("Created whatsapp_messages table")
		}
	}
	
	// Count total messages for this device
	var messageCount int
	countQuery := `SELECT COUNT(*) FROM whatsapp_messages WHERE device_id = $1`
	err = db.QueryRow(countQuery, deviceID).Scan(&messageCount)
	if err != nil {
		logrus.Errorf("Failed to count messages: %v", err)
	} else {
		logrus.Infof("Total messages for device %s: %d", deviceID, messageCount)
	}
	
	// If no messages, let's check whatsmeow tables
	if messageCount == 0 {
		logrus.Info("No messages in whatsapp_messages table, checking whatsmeow tables...")
		
		// Check if whatsmeow_messages exists
		var whatsmeowExists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = 'whatsmeow_messages'
			)
		`).Scan(&whatsmeowExists)
		
		if whatsmeowExists {
			var whatsmeowCount int
			err = db.QueryRow(`SELECT COUNT(*) FROM whatsmeow_messages`).Scan(&whatsmeowCount)
			if err == nil {
				logrus.Infof("Found %d messages in whatsmeow_messages table", whatsmeowCount)
				
				// Try to copy some messages
				if whatsmeowCount > 0 {
					copyQuery := `
						INSERT INTO whatsapp_messages (device_id, chat_jid, message_id, sender_jid, message_text, message_type, timestamp)
						SELECT 
							$1 as device_id,
							chat,
							id,
							sender,
							text,
							'text',
							timestamp / 1000
						FROM whatsmeow_messages
						WHERE chat NOT LIKE '%@g.us'
						AND chat NOT LIKE '%@broadcast'
						AND text IS NOT NULL
						AND text != ''
						LIMIT 100
						ON CONFLICT (device_id, message_id) DO NOTHING
					`
					
					result, err := db.Exec(copyQuery, deviceID)
					if err != nil {
						logrus.Errorf("Failed to copy messages: %v", err)
					} else {
						rowsAffected, _ := result.RowsAffected()
						logrus.Infof("Copied %d messages from whatsmeow_messages", rowsAffected)
					}
				}
			}
		}
	}
	
	// Query recent chats with last message
	query := `
		WITH recent_chats AS (
			SELECT DISTINCT ON (chat_jid) 
				chat_jid,
				message_text,
				timestamp,
				sender_jid
			FROM whatsapp_messages
			WHERE device_id = $1
			AND chat_jid NOT LIKE '%@g.us'
			AND chat_jid NOT LIKE '%@broadcast'
			AND chat_jid != 'status@broadcast'
			ORDER BY chat_jid, timestamp DESC
		)
		SELECT * FROM recent_chats
		ORDER BY timestamp DESC
		LIMIT 50
	`
	
	rows, err := db.Query(query, deviceID)
	if err != nil {
		logrus.Errorf("Failed to query recent chats: %v", err)
		return []map[string]interface{}{}, nil
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	ourJID := client.Store.ID.String()
	chatCount := 0
	
	for rows.Next() {
		var chatJID string
		var messageText sql.NullString
		var timestamp int64
		var senderJID string
		
		err := rows.Scan(&chatJID, &messageText, &timestamp, &senderJID)
		if err != nil {
			logrus.Errorf("Error scanning row: %v", err)
			continue
		}
		
		chatCount++
		logrus.Debugf("Processing chat: %s", chatJID)
		
		// Parse JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			logrus.Warnf("Failed to parse JID %s: %v", chatJID, err)
			continue
		}
		
		if jid.Server != types.DefaultUserServer {
			logrus.Debugf("Skipping non-user chat: %s", chatJID)
			continue
		}
		
		// Get contact name
		contactName := jid.User
		contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
		if err == nil && contact.Found {
			if contact.PushName != "" {
				contactName = contact.PushName
			} else if contact.BusinessName != "" {
				contactName = contact.BusinessName
			}
		}
		
		// Format message
		message := ""
		if messageText.Valid {
			message = messageText.String
			if len(message) > 50 {
				message = message[:47] + "..."
			}
			
			// Add "You: " prefix for sent messages
			if senderJID == ourJID {
				message = "You: " + message
			}
		}
		
		// Format time
		timeStr := formatMessageTime(timestamp)
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": message,
			"time":        timeStr,
			"timestamp":   timestamp,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
	}
	
	logrus.Infof("=== Found %d chats from %d rows for device %s ===", len(chats), chatCount, deviceID)
	
	return chats, nil
}

// GetWhatsAppWebMessages gets messages for a specific chat
func GetWhatsAppWebMessages(deviceID, chatJID string, limit int) ([]map[string]interface{}, error) {
	logrus.Infof("=== GetWhatsAppWebMessages called for device: %s, chat: %s ===", deviceID, chatJID)
	
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	if client.Store.ID == nil {
		return nil, fmt.Errorf("device not logged in")
	}
	
	ourJID := client.Store.ID.String()
	
	// Get messages from database
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// First count messages for this chat
	var count int
	countQuery := `SELECT COUNT(*) FROM whatsapp_messages WHERE device_id = $1 AND chat_jid = $2`
	err = db.QueryRow(countQuery, deviceID, chatJID).Scan(&count)
	if err != nil {
		logrus.Errorf("Failed to count messages for chat: %v", err)
	} else {
		logrus.Infof("Total messages in chat %s: %d", chatJID, count)
	}
	
	query := `
		SELECT 
			message_id,
			sender_jid,
			message_text,
			message_type,
			timestamp
		FROM whatsapp_messages
		WHERE device_id = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`
	
	rows, err := db.Query(query, deviceID, chatJID, limit)
	if err != nil {
		logrus.Errorf("Failed to query messages: %v", err)
		return []map[string]interface{}{}, nil
	}
	defer rows.Close()
	
	var messages []map[string]interface{}
	
	for rows.Next() {
		var messageID, senderJID, messageType string
		var messageText sql.NullString
		var timestamp int64
		
		err := rows.Scan(&messageID, &senderJID, &messageText, &messageType, &timestamp)
		if err != nil {
			logrus.Errorf("Error scanning message: %v", err)
			continue
		}
		
		// Determine if sent or received
		sent := senderJID == ourJID
		
		// Format time
		t := time.Unix(timestamp, 0)
		timeStr := t.Format("15:04")
		
		text := ""
		if messageText.Valid {
			text = messageText.String
		}
		
		message := map[string]interface{}{
			"id":        messageID,
			"text":      text,
			"type":      messageType,
			"sent":      sent,
			"time":      timeStr,
			"timestamp": timestamp,
		}
		
		messages = append(messages, message)
	}
	
	// Reverse to show oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	
	logrus.Infof("=== Found %d messages for chat %s ===", len(messages), chatJID)
	return messages, nil
}

// formatMessageTime formats timestamp to readable time
func formatMessageTime(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	
	t := time.Unix(timestamp, 0)
	now := time.Now()
	
	// Today
	if t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year() {
		return t.Format("15:04")
	}
	
	// Yesterday
	yesterday := now.AddDate(0, 0, -1)
	if t.Day() == yesterday.Day() && t.Month() == yesterday.Month() && t.Year() == yesterday.Year() {
		return "Yesterday"
	}
	
	// This week
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Monday")
	}
	
	// Older
	return t.Format("Jan 2")
}

// RefreshWhatsAppChats triggers a manual history sync
func RefreshWhatsAppChats(deviceID string) error {
	logrus.Infof("=== RefreshWhatsAppChats called for device: %s ===", deviceID)
	
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Send presence to trigger sync
	if client.Store.ID != nil {
		client.SendPresence(types.PresenceAvailable)
		logrus.Info("Sent presence update to trigger sync")
		
		// Request app state sync which includes chat history
		err = client.FetchAppState(context.Background(), appstate.WAPatchCriticalBlock, true, false)
		if err != nil {
			logrus.Errorf("Failed to fetch app state: %v", err)
		} else {
			logrus.Info("Requested app state sync")
		}
	}
	
	return nil
}
