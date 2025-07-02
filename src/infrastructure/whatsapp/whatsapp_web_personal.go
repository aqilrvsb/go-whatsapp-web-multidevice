package whatsapp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/appstate"
)

// GetWhatsAppWebChats gets personal chats from the WhatsApp client
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
	
	// Get all contacts from the store
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		logrus.Warnf("Failed to get contacts: %v", err)
		// Try alternative method
		return getChatsFromDatabase(deviceID, client.Store.ID.String())
	}
	
	var chats []map[string]interface{}
	
	// Process each contact
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
		} else {
			// Use phone number as fallback
			contactName = jid.User
		}
		
		// Get last message from database
		lastMessage, lastTimestamp := getLastMessage(deviceID, jid.String())
		
		// Format time
		timeStr := ""
		if lastTimestamp > 0 {
			t := time.Unix(lastTimestamp, 0)
			now := time.Now()
			if t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year() {
				timeStr = t.Format("15:04")
			} else if t.Year() == now.Year() {
				timeStr = t.Format("Jan 2")
			} else {
				timeStr = t.Format("2006-01-02")
			}
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
		}
		
		chats = append(chats, chat)
	}
	
	// Sort by timestamp (most recent first)
	sort.Slice(chats, func(i, j int) bool {
		ts1, _ := chats[i]["timestamp"].(int64)
		ts2, _ := chats[j]["timestamp"].(int64)
		return ts1 > ts2
	})
	
	return chats, nil
}

// getLastMessage gets the last message for a chat from the database
func getLastMessage(deviceID, chatJID string) (string, int64) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
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
	if err != nil {
		// No messages found
		return "", 0
	}
	
	return message, timestamp
}

// Fallback method using database
func getChatsFromDatabase(deviceID, ourJID string) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Get unique chats from messages table
	query := `
		SELECT DISTINCT 
			m.chat_jid,
			m.message_text,
			m.timestamp,
			COALESCE(c.push_name, c.business_name, c.full_name, c.first_name, '') as contact_name
		FROM (
			SELECT DISTINCT ON (chat_jid) 
				chat_jid, 
				message_text, 
				timestamp
			FROM whatsapp_messages
			WHERE device_id = $1
			AND chat_jid NOT LIKE '%@g.us'
			AND chat_jid NOT LIKE '%@broadcast'
			AND chat_jid != 'status@broadcast'
			ORDER BY chat_jid, timestamp DESC
		) m
		LEFT JOIN whatsmeow_contacts c ON c.our_jid = $2 AND c.their_jid = m.chat_jid
		ORDER BY m.timestamp DESC
	`
	
	rows, err := db.Query(query, deviceID, ourJID)
	if err != nil {
		logrus.Warnf("Failed to query chats from database: %v", err)
		return []map[string]interface{}{}, nil
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var chatJID string
		var lastMessage string
		var timestamp int64
		var contactName string
		
		err := rows.Scan(&chatJID, &lastMessage, &timestamp, &contactName)
		if err != nil {
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
		
		// Format time
		timeStr := ""
		if timestamp > 0 {
			t := time.Unix(timestamp, 0)
			now := time.Now()
			if t.Day() == now.Day() && t.Month() == now.Month() && t.Year() == now.Year() {
				timeStr = t.Format("15:04")
			} else if t.Year() == now.Year() {
				timeStr = t.Format("Jan 2")
			} else {
				timeStr = t.Format("2006-01-02")
			}
		}
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": lastMessage,
			"time":        timeStr,
			"timestamp":   timestamp,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
	}
	
	return chats, nil
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
	
	// Get messages from database
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	query := `
		SELECT message_id, sender_jid, message_text, message_type, timestamp
		FROM whatsapp_messages
		WHERE device_id = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`
	
	rows, err := db.Query(query, deviceID, chatJID, limit)
	if err != nil {
		logrus.Warnf("Failed to query messages: %v", err)
		return []map[string]interface{}{}, nil
	}
	defer rows.Close()
	
	var messages []map[string]interface{}
	
	for rows.Next() {
		var messageID, senderJID, messageText, messageType string
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
		
		message := map[string]interface{}{
			"id":        messageID,
			"text":      messageText,
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
	
	return messages, nil
}

// RefreshWhatsAppChats triggers a sync of contacts
func RefreshWhatsAppChats(deviceID string) error {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Request contact refresh
	err = client.FetchAppState(context.Background(), appstate.WAPatchCriticalUnblockLow, false, false)
	if err != nil {
		logrus.Warnf("Failed to fetch app state: %v", err)
	}
	
	return nil
}
