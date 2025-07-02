package whatsapp

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// GetWhatsAppWebChats gets personal chats only (no groups) from WhatsMeow's store
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
	
	// Get database connection
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Query to get all personal chats with contact info and last message
	// This joins whatsmeow_chat_settings with whatsmeow_contacts and our messages
	// EXCLUDING groups (chat_jid NOT LIKE '%@g.us')
	query := `
		SELECT 
			cs.chat_jid,
			cs.muted_until,
			cs.pinned,
			cs.archived,
			COALESCE(c.push_name, c.business_name, c.full_name, c.first_name, '') as contact_name,
			m.last_message,
			m.last_timestamp
		FROM whatsmeow_chat_settings cs
		LEFT JOIN whatsmeow_contacts c ON c.our_jid = cs.our_jid AND c.their_jid = cs.chat_jid
		LEFT JOIN (
			SELECT 
				chat_jid, 
				message_text as last_message,
				timestamp as last_timestamp
			FROM (
				SELECT 
					chat_jid, 
					message_text, 
					timestamp,
					ROW_NUMBER() OVER (PARTITION BY chat_jid ORDER BY timestamp DESC) as rn
				FROM whatsapp_messages
				WHERE device_id = $1
			) t WHERE rn = 1
		) m ON m.chat_jid = cs.chat_jid
		WHERE cs.our_jid = $1 
		AND cs.chat_jid NOT LIKE '%@g.us'
		AND cs.chat_jid NOT LIKE '%@broadcast'
		AND cs.chat_jid != 'status@broadcast'
		ORDER BY cs.pinned DESC, m.last_timestamp DESC NULLS LAST
	`
	
	rows, err := db.Query(query, ourJID)
	if err != nil {
		logrus.Warnf("Failed to query chats: %v", err)
		// If table doesn't exist, return empty
		return []map[string]interface{}{}, nil
	}
	defer rows.Close()
	
	var chats []map[string]interface{}
	
	for rows.Next() {
		var chatJID string
		var mutedUntil sql.NullInt64
		var pinned sql.NullBool
		var archived sql.NullBool
		var contactName string
		var lastMessage sql.NullString
		var lastTimestamp sql.NullInt64
		
		err := rows.Scan(&chatJID, &mutedUntil, &pinned, &archived, &contactName, &lastMessage, &lastTimestamp)
		if err != nil {
			logrus.Warnf("Error scanning row: %v", err)
			continue
		}
		
		// Parse JID to get phone number
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			continue
		}
		
		// Skip non-personal chats
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Format contact name
		if contactName == "" {
			// Use phone number if no name
			contactName = formatPhoneNumberWA(jid.User)
		}
		
		// Check if muted
		isMuted := false
		if mutedUntil.Valid && mutedUntil.Int64 > 0 {
			mutedTime := time.Unix(mutedUntil.Int64, 0)
			isMuted = mutedTime.After(time.Now())
		}
		
		// Format last message time
		timeStr := ""
		if lastTimestamp.Valid && lastTimestamp.Int64 > 0 {
			msgTime := time.Unix(lastTimestamp.Int64, 0)
			timeStr = formatWhatsAppTime(msgTime)
		}
		
		chat := map[string]interface{}{
			"id":          chatJID,
			"name":        contactName,
			"phone":       jid.User,
			"lastMessage": lastMessage.String,
			"time":        timeStr,
			"timestamp":   lastTimestamp.Int64,
			"unread":      0, // TODO: Implement unread count
			"isGroup":     false,
			"isMuted":     isMuted,
			"isArchived":  archived.Valid && archived.Bool,
			"isPinned":    pinned.Valid && pinned.Bool,
		}
		
		chats = append(chats, chat)
	}
	
	// Sort: Pinned first, then by last message time
	sort.Slice(chats, func(i, j int) bool {
		// Pinned chats first
		if chats[i]["isPinned"].(bool) != chats[j]["isPinned"].(bool) {
			return chats[i]["isPinned"].(bool)
		}
		// Then by last message timestamp
		t1, _ := chats[i]["timestamp"].(int64)
		t2, _ := chats[j]["timestamp"].(int64)
		return t1 > t2
	})
	
	logrus.Infof("Found %d personal chats for device %s", len(chats), deviceID)
	return chats, nil
}

// GetWhatsAppWebMessages gets messages for a specific chat
// Returns last 20 messages like WhatsApp Web
func GetWhatsAppWebMessages(deviceID string, chatJID string, limit int) ([]map[string]interface{}, error) {
	// Validate chat JID
	_, err := types.ParseJID(chatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat JID: %v", err)
	}
	
	// Get messages from our message store
	messages, err := GetMessagesForChatWeb(deviceID, chatJID)
	if err != nil {
		logrus.Warnf("Failed to get messages: %v", err)
		// Return empty instead of error
		return []map[string]interface{}{}, nil
	}
	
	return messages, nil
}

// formatPhoneNumberWA formats phone number for display
func formatPhoneNumberWA(phone string) string {
	if phone == "" {
		return "Unknown"
	}
	
	// Add + if not present
	if !strings.HasPrefix(phone, "+") {
		return "+" + phone
	}
	
	return phone
}

// RefreshWhatsAppChats triggers a contact sync for a device
func RefreshWhatsAppChats(deviceID string) error {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Request presence subscription to trigger contact sync
	err = client.SubscribePresence(types.JID{User: "status", Server: types.BroadcastServer})
	if err != nil {
		logrus.Warnf("Failed to subscribe presence: %v", err)
	}
	
	logrus.Infof("Contact refresh triggered for device %s", deviceID)
	return nil
}

// formatWhatsAppTime formats time like WhatsApp Web
func formatWhatsAppTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	
	if t.After(today) {
		// Today - show time
		return t.Format("3:04 PM")
	} else if t.After(yesterday) {
		return "Yesterday"
	} else if t.After(today.AddDate(0, 0, -7)) {
		// Last week - show day name
		return t.Format("Monday")
	} else if t.Year() == now.Year() {
		// This year - show date
		return t.Format("1/2")
	} else {
		// Older - show full date
		return t.Format("1/2/06")
	}
}