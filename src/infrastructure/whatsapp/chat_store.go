package whatsapp

import (
	"fmt"
	"time"
	"context"
	
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// CreateChatTable ensures the chats table exists
func CreateChatTable() error {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	query := `
		CREATE TABLE IF NOT EXISTS whatsapp_chats (
			id SERIAL PRIMARY KEY,
			device_id TEXT NOT NULL,
			chat_jid TEXT NOT NULL,
			chat_name TEXT NOT NULL,
			last_message_time TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(device_id, chat_jid)
		);
		
		CREATE INDEX IF NOT EXISTS idx_chats_device_time ON whatsapp_chats(device_id, last_message_time DESC);
	`
	
	_, err := db.Exec(query)
	return err
}

// StoreChat stores or updates a chat in the database
func StoreChat(deviceID, chatJID, name string, lastMessageTime time.Time) error {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	query := `
		INSERT INTO whatsapp_chats (device_id, chat_jid, chat_name, last_message_time)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, chat_jid) 
		DO UPDATE SET 
			chat_name = EXCLUDED.chat_name,
			last_message_time = EXCLUDED.last_message_time
	`
	
	_, err := db.Exec(query, deviceID, chatJID, name, lastMessageTime)
	if err != nil {
		logrus.Errorf("Failed to store chat: %v", err)
	} else {
		logrus.Debugf("Stored/updated chat: %s (%s)", name, chatJID)
	}
	return err
}

// GetChatName determines the appropriate name for a chat
func GetChatName(client *whatsmeow.Client, jid types.JID, chatJID string) string {
	// For groups
	if jid.Server == types.GroupServer {
		groupInfo, err := client.GetGroupInfo(jid)
		if err == nil && groupInfo.Name != "" {
			return groupInfo.Name
		}
		return fmt.Sprintf("Group %s", jid.User)
	}
	
	// For individual contacts
	contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
	if err == nil && contact.Found {
		if contact.PushName != "" {
			return contact.PushName
		}
		if contact.BusinessName != "" {
			return contact.BusinessName
		}
		if contact.FullName != "" {
			return contact.FullName
		}
	}
	
	// Fallback to phone number
	return jid.User
}

// HandleMessageForChats processes messages and updates chat list
func HandleMessageForChats(deviceID string, client *whatsmeow.Client, evt *events.Message) {
	// Skip non-personal chats for WhatsApp Web
	if evt.Info.Chat.Server != types.DefaultUserServer {
		return
	}
	
	// Get chat name
	chatJID := evt.Info.Chat.String()
	name := GetChatName(client, evt.Info.Chat, chatJID)
	
	// Store/update chat
	err := StoreChat(deviceID, chatJID, name, evt.Info.Timestamp)
	if err != nil {
		logrus.Warnf("Failed to store chat from message: %v", err)
	}
	
	// Also store the message (existing function)
	HandleMessageForWebView(deviceID, evt)
}

// HandleHistorySyncForChats processes history sync and updates chat list
func HandleHistorySyncForChats(deviceID string, client *whatsmeow.Client, evt *events.HistorySync) {
	logrus.Infof("Processing history sync for chats - device %s", deviceID)
	
	chatCount := 0
	for _, conv := range evt.Data.GetConversations() {
		if conv.GetId() == "" {
			continue
		}
		
		// Parse chat JID
		chatJID, err := types.ParseJID(conv.GetId())
		if err != nil {
			continue
		}
		
		// Skip non-personal chats
		if chatJID.Server != types.DefaultUserServer {
			continue
		}
		
		// Get chat name
		name := GetChatName(client, chatJID, conv.GetId())
		
		// Get last message time from conversation
		var lastMessageTime time.Time
		if len(conv.GetMessages()) > 0 {
			// Get timestamp from first message (they're in reverse order)
			firstMsg := conv.GetMessages()[0]
			if firstMsg != nil && firstMsg.GetMessage() != nil {
				timestamp := firstMsg.GetMessage().GetMessageTimestamp()
				if timestamp > 0 {
					lastMessageTime = time.Unix(int64(timestamp), 0)
				}
			}
		}
		
		// If no message time, use current time
		if lastMessageTime.IsZero() {
			lastMessageTime = time.Now()
		}
		
		// Store chat
		err = StoreChat(deviceID, conv.GetId(), name, lastMessageTime)
		if err == nil {
			chatCount++
		}
	}
	
	logrus.Infof("Stored %d chats from history sync", chatCount)
	
	// Also process messages
	HandleHistorySyncForWebView(deviceID, evt)
}

// GetChatsFromDatabase retrieves ONLY chats with recent messages (within last month)
func GetChatsFromDatabase(deviceID string) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Ensure table exists
	CreateChatTable()
	
	// Modified query to only show chats with recent messages
	query := `
		SELECT 
			c.chat_jid,
			c.chat_name,
			c.last_message_time,
			m.message_text,
			m.timestamp,
			m.message_count
		FROM whatsapp_chats c
		INNER JOIN (
			SELECT 
				chat_jid,
				MAX(message_text) as message_text,
				MAX(timestamp) as timestamp,
				COUNT(*) as message_count
			FROM (
				SELECT 
					chat_jid,
					message_text,
					timestamp,
					ROW_NUMBER() OVER (PARTITION BY chat_jid ORDER BY timestamp DESC) as rn
				FROM whatsapp_messages
				WHERE device_id = $1
					AND timestamp > EXTRACT(EPOCH FROM NOW() - INTERVAL '30 days')::BIGINT
			) ranked_messages
			WHERE rn = 1
			GROUP BY chat_jid
		) m ON c.chat_jid = m.chat_jid
		WHERE c.device_id = $1
		ORDER BY m.timestamp DESC
	`
	
	rows, err := db.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %v", err)
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var chatJID, name string
		var lastMessageTime time.Time
		var messageText string
		var messageTimestamp int64
		var messageCount int
		
		err := rows.Scan(&chatJID, &name, &lastMessageTime, &messageText, &messageTimestamp, &messageCount)
		if err != nil {
			logrus.Warnf("Failed to scan chat row: %v", err)
			continue
		}
		
		// Parse JID to get phone number
		jid, _ := types.ParseJID(chatJID)
		phone := jid.User
		
		// Get timestamp
		timestamp := messageTimestamp
		if timestamp == 0 {
			timestamp = lastMessageTime.Unix()
		}
		
		// Format time
		timeStr := formatMessageTime(timestamp)
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        name,
			"phone":       phone,
			"lastMessage": messageText,
			"time":        timeStr,
			"timestamp":   timestamp,
			"unread":      0,
			"isGroup":     false,
			"messageCount": messageCount,
		}
		
		chats = append(chats, chat)
	}
	
	logrus.Infof("Retrieved %d recent chats (with messages in last 30 days) for device %s", len(chats), deviceID)
	return chats, nil
}

// GetRecentChatsOnly retrieves only chats with recent activity (configurable time period)
func GetRecentChatsOnly(deviceID string, days int) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	if days <= 0 {
		days = 30 // Default to 30 days
	}
	
	// Query that only returns chats with messages in the specified time period
	query := `
		WITH recent_messages AS (
			SELECT DISTINCT
				chat_jid,
				FIRST_VALUE(message_text) OVER (PARTITION BY chat_jid ORDER BY timestamp DESC) as last_message,
				MAX(timestamp) OVER (PARTITION BY chat_jid) as last_timestamp
			FROM whatsapp_messages
			WHERE device_id = $1
				AND timestamp > EXTRACT(EPOCH FROM NOW() - INTERVAL '%d days')::BIGINT
		)
		SELECT 
			c.chat_jid,
			c.chat_name,
			rm.last_message,
			rm.last_timestamp
		FROM whatsapp_chats c
		INNER JOIN recent_messages rm ON c.chat_jid = rm.chat_jid
		WHERE c.device_id = $1
		ORDER BY rm.last_timestamp DESC
	`
	
	formattedQuery := fmt.Sprintf(query, days)
	rows, err := db.Query(formattedQuery, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent chats: %v", err)
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var chatJID, name, lastMessage string
		var timestamp int64
		
		err := rows.Scan(&chatJID, &name, &lastMessage, &timestamp)
		if err != nil {
			continue
		}
		
		// Parse JID to get phone number
		jid, _ := types.ParseJID(chatJID)
		phone := jid.User
		
		// Format time
		timeStr := formatMessageTime(timestamp)
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        name,
			"phone":       phone,
			"lastMessage": lastMessage,
			"time":        timeStr,
			"timestamp":   timestamp,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
	}
	
	logrus.Infof("Retrieved %d chats with activity in last %d days for device %s", len(chats), days, deviceID)
	return chats, nil
}