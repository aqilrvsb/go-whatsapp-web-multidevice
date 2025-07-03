package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"strings"
	"reflect"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// GetWhatsAppWebChats gets recent chats from chat settings
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
	
	var chats []map[string]interface{}
	
	// First check if we have messages in our table
	var messageCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM whatsapp_messages WHERE device_id = $1`, deviceID).Scan(&messageCount)
	if err == nil && messageCount > 0 {
		logrus.Infof("Found %d messages in our table", messageCount)
		
		// Get chats from our table
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
		if err == nil {
			defer rows.Close()
			
			for rows.Next() {
				var chatJID string
				var messageText sql.NullString
				var timestamp int64
				var senderJID string
				
				err := rows.Scan(&chatJID, &messageText, &timestamp, &senderJID)
				if err != nil {
					continue
				}
				
				// Parse JID
				jid, err := types.ParseJID(chatJID)
				if err != nil || jid.Server != types.DefaultUserServer {
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
			
			logrus.Infof("Found %d chats from our table", len(chats))
			if len(chats) > 0 {
				return chats, nil
			}
		}
	}
	
	// If no messages in our table, use chat settings with recent contacts
	logrus.Info("Using chat settings to get recent chats")
	
	// Get recent chats from chat settings (these are actually active chats)
	query := `
		SELECT 
			cs.chat_jid,
			cs.muted_until,
			cs.pinned,
			cs.archived,
			COALESCE(c.push_name, c.business_name, c.full_name, c.first_name, '') as contact_name
		FROM whatsmeow_chat_settings cs
		LEFT JOIN whatsmeow_contacts c ON c.our_jid = cs.our_jid AND c.their_jid = cs.chat_jid
		WHERE cs.our_jid = $1
		AND cs.chat_jid NOT LIKE '%@g.us'
		AND cs.chat_jid NOT LIKE '%@broadcast'
		AND cs.chat_jid != 'status@broadcast'
		ORDER BY cs.pinned DESC, cs.chat_jid
		LIMIT 100
	`
	
	rows, err := db.Query(query, ourJID)
	if err != nil {
		logrus.Errorf("Failed to query chat settings: %v", err)
		// Fallback to contacts only
		return getRecentContactsAsChats(client, deviceID)
	}
	defer rows.Close()
	
	chatCount := 0
	for rows.Next() {
		var chatJID string
		var mutedUntil sql.NullInt64
		var pinned sql.NullBool
		var archived sql.NullBool
		var contactName string
		
		err := rows.Scan(&chatJID, &mutedUntil, &pinned, &archived, &contactName)
		if err != nil {
			continue
		}
		
		// Parse JID
		jid, err := types.ParseJID(chatJID)
		if err != nil || jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Use phone number if no name
		if contactName == "" {
			contactName = jid.User
		}
		
		// Get last message from our table if exists
		var lastMessage string
		var lastTimestamp int64
		msgQuery := `
			SELECT message_text, timestamp 
			FROM whatsapp_messages 
			WHERE device_id = $1 AND chat_jid = $2 
			ORDER BY timestamp DESC 
			LIMIT 1
		`
		db.QueryRow(msgQuery, deviceID, chatJID).Scan(&lastMessage, &lastTimestamp)
		
		timeStr := ""
		if lastTimestamp > 0 {
			timeStr = formatMessageTime(lastTimestamp)
		}
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": lastMessage,
			"time":        timeStr,
			"timestamp":   lastTimestamp,
			"unread":      0,
			"isGroup":     false,
			"pinned":      pinned.Valid && pinned.Bool,
			"archived":    archived.Valid && archived.Bool,
		}
		
		chats = append(chats, chat)
		chatCount++
	}
	
	logrus.Infof("=== Found %d chats for device %s ===", len(chats), deviceID)
	
	return chats, nil
}

// getRecentContactsAsChats gets recent contacts as a fallback
func getRecentContactsAsChats(client *whatsmeow.Client, deviceID string) ([]map[string]interface{}, error) {
	logrus.Info("Falling back to contact list")
	
	// Get ALL contacts and return them sorted by name
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		logrus.Errorf("Failed to get contacts: %v", err)
		return []map[string]interface{}{}, nil
	}
	
	var chats []map[string]interface{}
	
	for jid, contact := range contacts {
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Skip broadcast lists
		if strings.Contains(jid.String(), "@broadcast") {
			continue
		}
		
		contactName := contact.PushName
		if contactName == "" {
			contactName = contact.BusinessName
		}
		if contactName == "" {
			contactName = contact.FullName
		}
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
			"lastMessage": "",
			"time":        "",
			"timestamp":   0,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
		
		// Limit to prevent UI overload
		if len(chats) >= 100 {
			break
		}
	}
	
	logrus.Infof("Found %d contacts", len(chats))
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
	
	// Ensure table exists
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
	
	db.Exec(createTableQuery)
	
	// Query messages
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
	
	// If no messages found, create a welcome message
	if len(messages) == 0 {
		messages = append(messages, map[string]interface{}{
			"id":        "welcome",
			"text":      "Send a message to start the conversation",
			"type":      "text",
			"sent":      false,
			"time":      time.Now().Format("15:04"),
			"timestamp": time.Now().Unix(),
		})
	}
	
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

// RefreshWhatsAppChats does nothing to prevent contact list changes
func RefreshWhatsAppChats(deviceID string) error {
	logrus.Infof("=== RefreshWhatsAppChats called for device: %s ===", deviceID)
	
	// Just send presence, don't change anything
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	if client.Store.ID != nil {
		client.SendPresence(types.PresenceAvailable)
		logrus.Info("Sent presence update")
	}
	
	// Don't reload or change the contact list
	return nil
}
// StoreHistorySyncMessage stores a message from history sync with proper timestamp
func StoreHistorySyncMessage(deviceID, chatJID, messageID, senderJID, messageText, messageType string, timestamp int64) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Ensure table exists
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
	
	db.Exec(createTableQuery)
	
	// Insert or update message
	query := `
		INSERT INTO whatsapp_messages (device_id, chat_jid, message_id, sender_jid, message_text, message_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, message_id) 
		DO UPDATE SET 
			message_text = EXCLUDED.message_text,
			message_type = EXCLUDED.message_type,
			timestamp = EXCLUDED.timestamp
	`
	
	_, err := db.Exec(query, deviceID, chatJID, messageID, senderJID, messageText, messageType, timestamp)
	if err != nil {
		logrus.Errorf("Failed to store history sync message: %v", err)
	} else {
		logrus.Debugf("Stored history message: %s in chat %s", messageID, chatJID)
	}
}

// SyncWhatsAppHistory triggers a history sync for the device
func SyncWhatsAppHistory(deviceID string) error {
	logrus.Infof("=== Starting WhatsApp history sync for device: %s ===", deviceID)
	
	if deviceID == "" {
		return fmt.Errorf("device ID is empty")
	}
	
	cm := GetClientManager()
	if cm == nil {
		return fmt.Errorf("client manager is nil")
	}
	
	// Try to get the client
	client, err := cm.GetClient(deviceID)
	if err != nil {
		// Log all available clients for debugging
		allClients := cm.GetAllClients()
		logrus.Warnf("Failed to get client for device %s. Available clients: %d", deviceID, len(allClients))
		for id := range allClients {
			logrus.Warnf("Available client ID: %s", id)
		}
		return fmt.Errorf("device not connected: %v", err)
	}
	
	if client == nil {
		return fmt.Errorf("client is nil")
	}
	
	if client.Store == nil || client.Store.ID == nil {
		return fmt.Errorf("device not logged in or store is nil")
	}
	
	// Check if client is connected
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected to WhatsApp")
	}
	
	// The proper way to request history sync in whatsmeow
	// is to use the client.SendMessage with a protocol message
	
	// Create the history sync notification
	notification := &waProto.HistorySyncNotification{
		FileSHA256:     []byte{},
		FileLength:     proto.Uint64(0),
		MediaKey:       []byte{},
		FileEncSHA256:  []byte{},
		DirectPath:     proto.String(""),
		SyncType:       waProto.HistorySyncNotification_RECENT.Enum(),
		ChunkOrder:     proto.Uint32(1),
	}
	
	// Wrap in protocol message
	protocolMsg := &waProto.ProtocolMessage{
		Type:                    waProto.ProtocolMessage_HISTORY_SYNC_NOTIFICATION.Enum(),
		HistorySyncNotification: notification,
	}
	
	// Create the message
	msg := &waProto.Message{
		ProtocolMessage: protocolMsg,
	}
	
	// Send to our own JID to trigger history sync
	ctx := context.Background()
	resp, err := client.SendMessage(ctx, *client.Store.ID, msg)
	if err != nil {
		// If that fails, try alternative approach
		logrus.Warnf("Failed to send history sync via protocol message: %v, trying alternative", err)
		
		// Alternative: Send presence which often triggers sync
		err = client.SendPresence(types.PresenceAvailable)
		if err != nil {
			return fmt.Errorf("failed to trigger sync: %v", err)
		}
		
		logrus.Info("Sent presence update to trigger sync")
		return nil
	}
	
	logrus.Infof("History sync requested successfully. Response ID: %s", resp.ID)
	return nil
}
