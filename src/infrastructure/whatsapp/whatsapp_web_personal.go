package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// GetWhatsAppWebChats gets recent chats based on messages
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
	
	ourJID := client.Store.ID.String()
	logrus.Infof("Getting recent chats for device %s", deviceID)
	
	// Get recent chats from messages
	return getRecentChatsFromMessages(client, deviceID, ourJID)
}

// getRecentChatsFromMessages gets chats based on recent messages
func getRecentChatsFromMessages(client *whatsmeow.Client, deviceID, ourJID string) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Query to get recent chats with last message
	// First try whatsmeow_messages table
	query := `
		WITH recent_chats AS (
			SELECT DISTINCT ON (chat) 
				chat as chat_jid,
				text as message_text,
				timestamp / 1000 as timestamp,
				sender
			FROM whatsmeow_messages
			WHERE chat NOT LIKE '%@g.us'
			AND chat NOT LIKE '%@broadcast'
			AND chat != 'status@broadcast'
			AND text IS NOT NULL
			AND text != ''
			ORDER BY chat, timestamp DESC
		)
		SELECT 
			rc.chat_jid,
			rc.message_text,
			rc.timestamp,
			rc.sender,
			COALESCE(c.push_name, c.business_name, c.full_name, c.first_name, '') as contact_name
		FROM recent_chats rc
		LEFT JOIN whatsmeow_contacts c ON c.our_jid = $1 AND c.their_jid = rc.chat_jid
		ORDER BY rc.timestamp DESC
		LIMIT 50
	`
	
	rows, err := db.Query(query, ourJID)
	if err != nil {
		logrus.Warnf("Failed to query whatsmeow_messages, trying fallback: %v", err)
		
		// Fallback to whatsapp_messages table
		query = `
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
			SELECT 
				chat_jid,
				message_text,
				timestamp,
				sender_jid,
				'' as contact_name
			FROM recent_chats
			ORDER BY timestamp DESC
			LIMIT 50
		`
		
		rows, err = db.Query(query, deviceID)
		if err != nil {
			logrus.Errorf("Failed to get recent chats: %v", err)
			return []map[string]interface{}{}, nil
		}
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	seenChats := make(map[string]bool)
	
	for rows.Next() {
		var chatJID string
		var lastMessage sql.NullString
		var timestamp int64
		var senderJID string
		var contactName string
		
		err := rows.Scan(&chatJID, &lastMessage, &timestamp, &senderJID, &contactName)
		if err != nil {
			logrus.Warnf("Error scanning row: %v", err)
			continue
		}
		
		// Skip if already processed
		if seenChats[chatJID] {
			continue
		}
		seenChats[chatJID] = true
		
		// Parse JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			continue
		}
		
		// Skip non-user chats
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Get contact name if not available
		if contactName == "" {
			// Try to get from contact store
			contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
			if err == nil && contact.Found {
				if contact.PushName != "" {
					contactName = contact.PushName
				} else if contact.BusinessName != "" {
					contactName = contact.BusinessName
				} else if contact.FullName != "" {
					contactName = contact.FullName
				} else if contact.FirstName != "" {
					contactName = contact.FirstName
				}
			}
		}
		
		// Use phone number as fallback
		if contactName == "" {
			contactName = jid.User
		}
		
		// Format message
		message := ""
		if lastMessage.Valid {
			message = lastMessage.String
			// Truncate long messages
			if len(message) > 50 {
				message = message[:47] + "..."
			}
		}
		
		// Format time
		timeStr := formatMessageTime(timestamp)
		
		// Check if this is a sent message
		isSent := senderJID == ourJID || senderJID == client.Store.ID.User
		if isSent && message != "" {
			message = "You: " + message
		}
		
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
	
	logrus.Infof("Found %d recent chats with messages", len(chats))
	
	// If no chats found, try to get at least some recent contacts with any interaction
	if len(chats) == 0 {
		logrus.Info("No recent chats found, trying to get any recent interactions")
		chats = getAnyRecentInteractions(client, deviceID, ourJID)
	}
	
	return chats, nil
}

// getAnyRecentInteractions gets any recent WhatsApp interactions
func getAnyRecentInteractions(client *whatsmeow.Client, deviceID, ourJID string) []map[string]interface{} {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Try to get any recent message senders/receivers
	query := `
		SELECT DISTINCT 
			CASE 
				WHEN sender = $1 THEN chat
				ELSE sender
			END as contact_jid
		FROM whatsmeow_messages
		WHERE (sender = $1 OR chat = $1)
		AND chat NOT LIKE '%@g.us'
		AND chat NOT LIKE '%@broadcast'
		ORDER BY contact_jid
		LIMIT 20
	`
	
	rows, err := db.Query(query, ourJID)
	if err != nil {
		logrus.Warnf("Failed to get any interactions: %v", err)
		return []map[string]interface{}{}
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var contactJID string
		err := rows.Scan(&contactJID)
		if err != nil {
			continue
		}
		
		// Parse JID
		jid, err := types.ParseJID(contactJID)
		if err != nil {
			continue
		}
		
		// Skip invalid
		if jid.Server != types.DefaultUserServer || jid.User == "" {
			continue
		}
		
		// Get contact info
		contactName := jid.User
		contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
		if err == nil && contact.Found && contact.PushName != "" {
			contactName = contact.PushName
		}
		
		chat := map[string]interface{}{
			"id":          contactJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": "Click to view messages",
			"time":        "",
			"timestamp":   0,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
	}
	
	return chats
}

// formatMessageTime formats timestamp to readable time
func formatMessageTime(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	
	t := time.Unix(timestamp, 0)
	now := time.Now()
	
	// Today - show time
	if t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year() {
		return t.Format("15:04")
	}
	
	// Yesterday
	yesterday := now.AddDate(0, 0, -1)
	if t.Day() == yesterday.Day() && t.Month() == yesterday.Month() && t.Year() == yesterday.Year() {
		return "Yesterday"
	}
	
	// This week - show day name
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Monday")
	}
	
	// This year - show date
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	
	// Older - show full date
	return t.Format("02/01/2006")
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
		SELECT 
			id, 
			sender, 
			text, 
			timestamp / 1000 as timestamp,
			CASE WHEN sender = $3 THEN true ELSE false END as is_sent
		FROM whatsmeow_messages
		WHERE chat = $1
		AND text IS NOT NULL
		ORDER BY timestamp DESC
		LIMIT $2
	`
	
	rows, err := db.Query(query, chatJID, limit, ourJID)
	if err != nil {
		logrus.Warnf("Failed to query whatsmeow_messages: %v", err)
		
		// Try whatsapp_messages
		query = `
			SELECT 
				message_id, 
				sender_jid, 
				message_text, 
				timestamp,
				CASE WHEN sender_jid = $3 THEN true ELSE false END as is_sent
			FROM whatsapp_messages
			WHERE device_id = $1 AND chat_jid = $2
			ORDER BY timestamp DESC
			LIMIT $4
		`
		
		rows, err = db.Query(query, deviceID, chatJID, ourJID, limit)
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
		var isSent bool
		
		err := rows.Scan(&messageID, &senderJID, &messageText, &timestamp, &isSent)
		if err != nil {
			logrus.Warnf("Error scanning message: %v", err)
			continue
		}
		
		// Skip empty messages
		if !messageText.Valid || messageText.String == "" {
			continue
		}
		
		// Format time
		t := time.Unix(timestamp, 0)
		timeStr := t.Format("15:04")
		
		message := map[string]interface{}{
			"id":        messageID,
			"text":      messageText.String,
			"type":      "text",
			"sent":      isSent,
			"time":      timeStr,
			"timestamp": timestamp,
		}
		
		messages = append(messages, message)
	}
	
	// Reverse to show oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	
	logrus.Infof("Found %d messages for chat %s", len(messages), chatJID)
	return messages, nil
}

// RefreshWhatsAppChats triggers a sync of recent messages
func RefreshWhatsAppChats(deviceID string) error {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Request presence to trigger sync
	if client.Store.ID != nil {
		logrus.Info("Requesting presence update...")
		client.SendPresence(types.PresenceAvailable)
	}
	
	return nil
}
