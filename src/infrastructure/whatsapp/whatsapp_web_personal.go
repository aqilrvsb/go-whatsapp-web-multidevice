package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow"
)

// GetWhatsAppWebChats gets recent chats - simplified approach
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
	
	ourJID := client.Store.ID.String()
	logrus.Infof("Client connected, JID: %s", ourJID)
	
	// Get database connection
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Let's directly check whatsmeow tables since that's where WhatsApp stores messages
	var chats []map[string]interface{}
	
	// First, let's see what tables exist
	tableQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name LIKE 'whatsmeow_%'
		ORDER BY table_name
	`
	
	rows, err := db.Query(tableQuery)
	if err == nil {
		defer rows.Close()
		logrus.Info("Available whatsmeow tables:")
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err == nil {
				logrus.Infof("  - %s", tableName)
			}
		}
	}
	
	// Query directly from whatsmeow_messages
	query := `
		SELECT 
			COALESCE(chat, sender) as chat_jid,
			MAX(text) as last_message,
			MAX(timestamp) as last_timestamp,
			COUNT(*) as message_count
		FROM whatsmeow_messages
		WHERE (chat LIKE '%@s.whatsapp.net' OR sender LIKE '%@s.whatsapp.net')
		AND chat NOT LIKE '%@g.us'
		AND chat NOT LIKE '%@broadcast'
		AND text IS NOT NULL
		AND text != ''
		GROUP BY COALESCE(chat, sender)
		ORDER BY MAX(timestamp) DESC
		LIMIT 50
	`
	
	rows, err = db.Query(query)
	if err != nil {
		logrus.Errorf("Failed to query whatsmeow_messages: %v", err)
		
		// If whatsmeow_messages doesn't exist, let's check message_history
		query = `
			SELECT 
				jid as chat_jid,
				message as last_message,
				MAX(timestamp) as last_timestamp,
				COUNT(*) as message_count
			FROM whatsmeow_message_history
			WHERE jid LIKE '%@s.whatsapp.net'
			GROUP BY jid, message
			ORDER BY MAX(timestamp) DESC
			LIMIT 50
		`
		
		rows, err = db.Query(query)
		if err != nil {
			logrus.Errorf("Failed to query message_history: %v", err)
			
			// Last resort - get recent contacts
			return getRecentContactsAsChats(client, deviceID)
		}
	}
	defer rows.Close()
	
	chatCount := 0
	for rows.Next() {
		var chatJID string
		var lastMessage sql.NullString
		var timestamp sql.NullInt64
		var messageCount int
		
		err := rows.Scan(&chatJID, &lastMessage, &timestamp, &messageCount)
		if err != nil {
			logrus.Errorf("Error scanning row: %v", err)
			continue
		}
		
		chatCount++
		logrus.Debugf("Found chat: %s with %d messages", chatJID, messageCount)
		
		// Parse JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			logrus.Warnf("Failed to parse JID %s: %v", chatJID, err)
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
		if lastMessage.Valid {
			message = lastMessage.String
			if len(message) > 50 {
				message = message[:47] + "..."
			}
		}
		
		// Format time
		timeStr := ""
		if timestamp.Valid {
			// WhatsApp timestamps are in milliseconds
			t := time.Unix(timestamp.Int64/1000, 0)
			timeStr = formatMessageTime(t.Unix())
		}
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": message,
			"time":        timeStr,
			"timestamp":   timestamp.Int64 / 1000,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
	}
	
	logrus.Infof("=== Found %d chats for device %s ===", len(chats), deviceID)
	
	// Store these messages in our table for future use
	if len(chats) > 0 {
		go syncMessagesToOurTable(deviceID, ourJID)
	}
	
	return chats, nil
}

// getRecentContactsAsChats gets recent contacts as a fallback
func getRecentContactsAsChats(client *whatsmeow.Client, deviceID string) ([]map[string]interface{}, error) {
	logrus.Info("Falling back to contact list")
	
	// Get recent contacts
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		logrus.Errorf("Failed to get contacts: %v", err)
		return []map[string]interface{}{}, nil
	}
	
	var chats []map[string]interface{}
	count := 0
	
	for jid, contact := range contacts {
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		if count >= 20 { // Limit to 20 recent contacts
			break
		}
		
		contactName := contact.PushName
		if contactName == "" {
			contactName = contact.FirstName
		}
		if contactName == "" {
			contactName = jid.User
		}
		
		chat := map[string]interface{}{
			"id":          jid.String(),
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": "Click to load messages",
			"time":        "",
			"timestamp":   0,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
		count++
	}
	
	return chats, nil
}

// syncMessagesToOurTable copies messages from whatsmeow to our table
func syncMessagesToOurTable(deviceID, ourJID string) {
	logrus.Info("Starting background sync of messages to our table")
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Create table if not exists
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
	
	_, err := db.Exec(createTableQuery)
	if err != nil {
		logrus.Errorf("Failed to create table: %v", err)
		return
	}
	
	// Copy messages from whatsmeow_messages
	copyQuery := `
		INSERT INTO whatsapp_messages (device_id, chat_jid, message_id, sender_jid, message_text, message_type, timestamp)
		SELECT 
			$1 as device_id,
			COALESCE(chat, sender) as chat_jid,
			id,
			sender,
			text,
			'text',
			timestamp / 1000
		FROM whatsmeow_messages
		WHERE (chat LIKE '%@s.whatsapp.net' OR sender LIKE '%@s.whatsapp.net')
		AND text IS NOT NULL
		AND text != ''
		ON CONFLICT (device_id, message_id) DO NOTHING
	`
	
	result, err := db.Exec(copyQuery, deviceID)
	if err != nil {
		logrus.Errorf("Failed to sync messages: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		logrus.Infof("Synced %d messages to our table", rowsAffected)
	}
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
	
	// Try our table first
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
		logrus.Warnf("Failed to query our table, trying whatsmeow: %v", err)
		
		// Try whatsmeow_messages
		query = `
			SELECT 
				id,
				sender,
				text,
				'text' as message_type,
				timestamp / 1000 as timestamp
			FROM whatsmeow_messages
			WHERE (chat = $1 OR (chat IS NULL AND sender = $1))
			AND text IS NOT NULL
			ORDER BY timestamp DESC
			LIMIT $2
		`
		
		rows, err = db.Query(query, chatJID, limit)
		if err != nil {
			logrus.Errorf("Failed to query messages: %v", err)
			return []map[string]interface{}{}, nil
		}
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

// RefreshWhatsAppChats triggers a manual refresh
func RefreshWhatsAppChats(deviceID string) error {
	logrus.Infof("=== RefreshWhatsAppChats called for device: %s ===", deviceID)
	
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Just sync existing messages for now
	if client.Store.ID != nil {
		go syncMessagesToOurTable(deviceID, client.Store.ID.String())
		
		// Send presence
		client.SendPresence(types.PresenceAvailable)
		logrus.Info("Sent presence update")
	}
	
	return nil
}
