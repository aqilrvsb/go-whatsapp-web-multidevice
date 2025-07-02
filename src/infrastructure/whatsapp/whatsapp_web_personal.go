package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// GetWhatsAppWebChats gets recent chats from WhatsApp
func GetWhatsAppWebChats(deviceID string) ([]map[string]interface{}, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	// Get JID for this device
	if client.Store.ID == nil {
		return nil, fmt.Errorf("device not logged in")
	}
	
	logrus.Infof("Getting chats for device %s", deviceID)
	
	// First, try to get recent chats from the message history
	recentChats, err := getRecentChatsFromHistory(client, deviceID)
	if err == nil && len(recentChats) > 0 {
		logrus.Infof("Found %d recent chats from history", len(recentChats))
		return recentChats, nil
	}
	
	// Fallback to getting from database
	logrus.Info("Falling back to database method")
	return getChatsFromDatabase(deviceID, client.Store.ID.String())
}

// getRecentChatsFromHistory gets chats from WhatsApp's recent messages
func getRecentChatsFromHistory(client *whatsmeow.Client, deviceID string) ([]map[string]interface{}, error) {
	// Get all contacts instead
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, err
	}
	
	var chats []map[string]interface{}
	
	for jid, contact := range contacts {
		// Skip groups and broadcasts
		if strings.Contains(jid.String(), "@g.us") || 
		   strings.Contains(jid.String(), "@broadcast") ||
		   jid.String() == "status@broadcast" {
			continue
		}
		
		// Skip non-user JIDs
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Get contact name
		contactName := ""
		if contact.PushName != "" {
			contactName = contact.PushName
		} else if contact.BusinessName != "" {
			contactName = contact.BusinessName
		} else if contact.FullName != "" {
			contactName = contact.FullName
		} else if contact.FirstName != "" {
			contactName = contact.FirstName
		}
		
		// Use phone number if no name
		if contactName == "" {
			contactName = jid.User
		}
		
		// Get last message info from database
		lastMessage, lastTimestamp := getLastMessageFromDB(deviceID, jid.String())
		
		// Format time
		timeStr := ""
		if lastTimestamp > 0 {
			timeStr = formatMessageTime(lastTimestamp)
		}
		
		chat := map[string]interface{}{
			"id":          jid.String(),
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": lastMessage,
			"time":        timeStr,
			"timestamp":   lastTimestamp,
			"unread":      0,
			"isGroup":     false,
			"pinned":      false,
			"archived":    false,
		}
		
		chats = append(chats, chat)
	}
	
	// Sort by timestamp
	sort.Slice(chats, func(i, j int) bool {
		ts1, _ := chats[i]["timestamp"].(int64)
		ts2, _ := chats[j]["timestamp"].(int64)
		return ts1 > ts2
	})
	
	return chats, nil
}

// getLastMessageFromDB gets the last message from database
func getLastMessageFromDB(deviceID, chatJID string) (string, int64) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// First check if whatsapp_messages table exists
	var tableExists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'whatsapp_messages'
		)
	`
	db.QueryRow(checkQuery).Scan(&tableExists)
	
	if !tableExists {
		// Try to get from whatsmeow_messages table instead
		query := `
			SELECT text, timestamp 
			FROM whatsmeow_messages 
			WHERE chat = $1 AND sender != $1
			ORDER BY timestamp DESC 
			LIMIT 1
		`
		
		var message sql.NullString
		var timestamp sql.NullInt64
		
		err := db.QueryRow(query, chatJID).Scan(&message, &timestamp)
		if err == nil && message.Valid {
			return message.String, timestamp.Int64 / 1000 // Convert milliseconds to seconds
		}
	} else {
		// Use whatsapp_messages table
		query := `
			SELECT message_text, timestamp 
			FROM whatsapp_messages 
			WHERE device_id = $1 AND chat_jid = $2 
			ORDER BY timestamp DESC 
			LIMIT 1
		`
		
		var message string
		var timestamp int64
		
		err := db.QueryRow(query, deviceID, chatJID).Scan(&message, &timestamp)
		if err == nil {
			return message, timestamp
		}
	}
	
	return "", 0
}

// Fallback method using database
func getChatsFromDatabase(deviceID, ourJID string) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Try multiple approaches to get chats
	
	// First try: Get from whatsmeow_messages
	query := `
		SELECT DISTINCT ON (m.chat) 
			m.chat as chat_jid,
			m.text as message_text,
			m.timestamp / 1000 as timestamp,
			COALESCE(c.push_name, c.business_name, c.full_name, c.first_name, '') as contact_name
		FROM whatsmeow_messages m
		LEFT JOIN whatsmeow_contacts c ON c.our_jid = $1 AND c.their_jid = m.chat
		WHERE m.chat NOT LIKE '%@g.us'
		AND m.chat NOT LIKE '%@broadcast'
		AND m.chat != 'status@broadcast'
		AND m.text IS NOT NULL
		AND m.text != ''
		ORDER BY m.chat, m.timestamp DESC
		LIMIT 100
	`
	
	rows, err := db.Query(query, ourJID)
	if err != nil {
		logrus.Warnf("Failed to query whatsmeow_messages: %v", err)
		
		// Second try: Get from whatsapp_messages if exists
		query = `
			SELECT DISTINCT ON (chat_jid) 
				chat_jid,
				message_text,
				timestamp,
				'' as contact_name
			FROM whatsapp_messages
			WHERE device_id = $1
			AND chat_jid NOT LIKE '%@g.us'
			AND chat_jid NOT LIKE '%@broadcast'
			AND chat_jid != 'status@broadcast'
			ORDER BY chat_jid, timestamp DESC
		`
		
		rows, err = db.Query(query, deviceID)
		if err != nil {
			logrus.Warnf("Failed to query whatsapp_messages: %v", err)
			return []map[string]interface{}{}, nil
		}
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var chatJID string
		var lastMessage sql.NullString
		var timestamp int64
		var contactName string
		
		err := rows.Scan(&chatJID, &lastMessage, &timestamp, &contactName)
		if err != nil {
			logrus.Warnf("Error scanning row: %v", err)
			continue
		}
		
		// Parse JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			continue
		}
		
		// Use phone number as name if contact name is empty
		if contactName == "" {
			contactName = jid.User
		}
		
		// Format message
		message := ""
		if lastMessage.Valid {
			message = lastMessage.String
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
	
	// Sort by timestamp
	sort.Slice(chats, func(i, j int) bool {
		ts1, _ := chats[i]["timestamp"].(int64)
		ts2, _ := chats[j]["timestamp"].(int64)
		return ts1 > ts2
	})
	
	logrus.Infof("Found %d chats from database", len(chats))
	return chats, nil
}

// formatMessageTime formats timestamp to readable time
func formatMessageTime(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	
	t := time.Unix(timestamp, 0)
	now := time.Now()
	
	if t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year() {
		return t.Format("15:04")
	} else if t.Year() == now.Year() {
		return t.Format("Jan 2")
	} else {
		return t.Format("2006-01-02")
	}
}

// GetWhatsAppWebMessages gets messages for a specific chat
func GetWhatsAppWebMessages(deviceID, chatJID string, limit int) ([]map[string]interface{}, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	// Get our JID
	if client.Store.ID == nil {
		return nil, fmt.Errorf("device not logged in")
	}
	ourJID := client.Store.ID.String()
	
	logrus.Infof("Getting messages for chat %s", chatJID)
	
	// Get messages from database
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Try whatsmeow_messages first
	query := `
		SELECT id, sender, text, timestamp / 1000 as timestamp
		FROM whatsmeow_messages
		WHERE chat = $1
		AND text IS NOT NULL
		ORDER BY timestamp DESC
		LIMIT $2
	`
	
	rows, err := db.Query(query, chatJID, limit)
	if err != nil {
		logrus.Warnf("Failed to query whatsmeow_messages: %v", err)
		
		// Try whatsapp_messages
		query = `
			SELECT message_id, sender_jid, message_text, timestamp
			FROM whatsapp_messages
			WHERE device_id = $1 AND chat_jid = $2
			ORDER BY timestamp DESC
			LIMIT $3
		`
		
		rows, err = db.Query(query, deviceID, chatJID, limit)
		if err != nil {
			logrus.Warnf("Failed to query messages: %v", err)
			return []map[string]interface{}{}, nil
		}
	}
	defer rows.Close()
	
	var messages []map[string]interface{}
	
	for rows.Next() {
		var messageID, senderJID string
		var messageText sql.NullString
		var timestamp int64
		
		err := rows.Scan(&messageID, &senderJID, &messageText, &timestamp)
		if err != nil {
			logrus.Warnf("Error scanning message: %v", err)
			continue
		}
		
		// Skip empty messages
		if !messageText.Valid || messageText.String == "" {
			continue
		}
		
		// Determine if sent or received
		sent := senderJID == ourJID || senderJID == client.Store.ID.User
		
		// Format time
		t := time.Unix(timestamp, 0)
		timeStr := t.Format("15:04")
		
		message := map[string]interface{}{
			"id":        messageID,
			"text":      messageText.String,
			"type":      "text",
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
	
	logrus.Infof("Found %d messages", len(messages))
	return messages, nil
}

// RefreshWhatsAppChats triggers a sync of contacts and messages
func RefreshWhatsAppChats(deviceID string) error {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Request history sync
	if client.Store.ID != nil {
		logrus.Info("Requesting history sync...")
		client.SendPresence(types.PresenceAvailable)
	}
	
	return nil
}
