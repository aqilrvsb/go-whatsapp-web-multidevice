package whatsapp

import (
	"context"
	"fmt"
	"time"
	
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// GetWhatsAppWebChats gets chats - SIMPLE APPROACH
func GetWhatsAppWebChats(deviceID string) ([]map[string]interface{}, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	if client.Store.ID == nil {
		return nil, fmt.Errorf("device not logged in")
	}
	
	// Simple approach - get ALL contacts first
	contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		logrus.Errorf("Failed to get contacts: %v", err)
		return []map[string]interface{}{}, nil
	}
	
	var chats []map[string]interface{}
	count := 0
	
	// Process each contact
	for jid, contact := range contacts {
		// Only personal chats
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		// Limit to 100 for now
		if count >= 100 {
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
			"lastMessage": "Click to view messages",
			"time":        "",
			"timestamp":   0,
			"unread":      0,
			"isGroup":     false,
		}
		
		chats = append(chats, chat)
		count++
	}
	
	logrus.Infof("Got %d contacts for WhatsApp Web", len(chats))
	return chats, nil
}

// GetWhatsAppWebMessages gets messages - SIMPLE APPROACH
func GetWhatsAppWebMessages(deviceID, chatJID string, limit int) ([]map[string]interface{}, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected: %v", err)
	}
	
	if client.Store.ID == nil {
		return nil, fmt.Errorf("device not logged in")
	}
	
	// For now, return empty - we'll implement this next
	return []map[string]interface{}{
		{
			"id":        "1",
			"text":      "This is a test message",
			"type":      "text",
			"sent":      false,
			"time":      time.Now().Format("15:04"),
			"timestamp": time.Now().Unix(),
		},
	}, nil
}

// RefreshWhatsAppChats does nothing for now
func RefreshWhatsAppChats(deviceID string) error {
	return nil
}
