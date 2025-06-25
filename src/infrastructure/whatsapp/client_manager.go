package whatsapp

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ClientManager manages multiple WhatsApp client instances
type ClientManager struct {
	clients map[string]*whatsmeow.Client
	mutex   sync.RWMutex
}

var (
	clientManager *ClientManager
	once          sync.Once
)

// GetClientManager returns the singleton client manager
func GetClientManager() *ClientManager {
	once.Do(func() {
		clientManager = &ClientManager{
			clients: make(map[string]*whatsmeow.Client),
		}
	})
	return clientManager
}

// AddClient adds a WhatsApp client for a device
func (cm *ClientManager) AddClient(deviceID string, client *whatsmeow.Client) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.clients[deviceID] = client
}

// GetClient retrieves a WhatsApp client for a device
func (cm *ClientManager) GetClient(deviceID string) (*whatsmeow.Client, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	client, exists := cm.clients[deviceID]
	if !exists {
		return nil, fmt.Errorf("no WhatsApp client found for device %s", deviceID)
	}
	
	if !client.IsConnected() {
		return nil, fmt.Errorf("WhatsApp client for device %s is not connected", deviceID)
	}
	
	return client, nil
}

// RemoveClient removes a WhatsApp client for a device
func (cm *ClientManager) RemoveClient(deviceID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.clients, deviceID)
}

// GetChatsForDevice fetches and saves chats for a specific device (personal chats only)
func GetChatsForDevice(deviceID string) ([]repository.WhatsAppChat, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	
	// Always try to get from database first
	repo := repository.GetWhatsAppRepository()
	savedChats, _ := repo.GetChats(deviceID)
	
	// Filter out groups - only keep personal chats
	var personalChats []repository.WhatsAppChat
	for _, chat := range savedChats {
		if !chat.IsGroup && !strings.Contains(chat.ChatJID, "@g.us") && !strings.Contains(chat.ChatJID, "@broadcast") {
			personalChats = append(personalChats, chat)
		}
	}
	
	if err != nil {
		// If client not connected, return saved personal chats from database
		return personalChats, nil
	}
	
	// Client is connected, try to sync latest data
	// Get all contacts first to have proper names
	contacts, err := client.Store.Contacts.GetAllContacts()
	if err != nil {
		fmt.Printf("Failed to get contacts: %v\n", err)
	}
	
	// Get recent conversations from device
	conversations, err := client.Store.ChatSettings.GetAllChatSettings()
	if err != nil {
		fmt.Printf("Failed to get chat settings: %v\n", err)
		return personalChats, nil
	}
	
	// Create a map for quick updates
	chatMap := make(map[string]*repository.WhatsAppChat)
	for i := range personalChats {
		chatMap[personalChats[i].ChatJID] = &personalChats[i]
	}
	
	// Process each conversation - ONLY PERSONAL CHATS
	for jid, settings := range conversations {
		// Skip groups, broadcasts, and status
		if jid.Server == types.GroupServer || jid.Server == types.BroadcastServer || jid.User == "status" {
			continue
		}
		
		// Only process personal chats (s.whatsapp.net)
		if jid.Server != types.DefaultUserServer {
			continue
		}
		
		chatJID := jid.String()
		
		// Get or create chat entry
		chat, exists := chatMap[chatJID]
		if !exists {
			chat = &repository.WhatsAppChat{
				DeviceID: deviceID,
				ChatJID:  chatJID,
				IsGroup:  false, // Always false for personal chats
			}
			chatMap[chatJID] = chat
		}
		
		// Update chat info
		chat.IsMuted = settings.Muted
		
		// Get contact name
		if contact, ok := contacts[jid]; ok {
			if contact.PushName != "" {
				chat.ChatName = contact.PushName
			} else if contact.BusinessName != "" {
				chat.ChatName = contact.BusinessName
			} else if contact.FullName != "" {
				chat.ChatName = contact.FullName
			} else {
				// Format phone number nicely
				chat.ChatName = formatPhoneNumber(jid.User)
			}
		} else {
			// Fallback to formatted phone number
			chat.ChatName = formatPhoneNumber(jid.User)
		}
		
		// Try to get last message from history sync
		if !exists {
			chat.LastMessageTime = time.Now()
			chat.LastMessageText = "Chat synced"
			chat.UnreadCount = 0
		}
		
		// Save to database
		if err := repo.SaveOrUpdateChat(chat); err != nil {
			fmt.Printf("Error saving chat %s: %v\n", chatJID, err)
		}
	}
	
	// Also check the contacts list to ensure we have ALL personal chats
	for jid, contact := range contacts {
		// Skip non-personal chats
		if jid.Server != types.DefaultUserServer || jid.User == "status" {
			continue
		}
		
		chatJID := jid.String()
		
		// Check if we already have this chat
		if _, exists := chatMap[chatJID]; !exists {
			// Create new chat entry for this contact
			chatName := ""
			if contact.PushName != "" {
				chatName = contact.PushName
			} else if contact.BusinessName != "" {
				chatName = contact.BusinessName
			} else if contact.FullName != "" {
				chatName = contact.FullName
			} else {
				chatName = formatPhoneNumber(jid.User)
			}
			
			chat := &repository.WhatsAppChat{
				DeviceID:        deviceID,
				ChatJID:         chatJID,
				ChatName:        chatName,
				IsGroup:         false,
				IsMuted:         false,
				LastMessageText: "",
				LastMessageTime: time.Now().Add(-365 * 24 * time.Hour), // Old date for contacts without messages
				UnreadCount:     0,
			}
			
			// Save to database
			if err := repo.SaveOrUpdateChat(chat); err != nil {
				fmt.Printf("Error saving contact chat %s: %v\n", chatJID, err)
			} else {
				chatMap[chatJID] = chat
			}
		}
	}
	
	// Convert map back to slice
	var updatedChats []repository.WhatsAppChat
	for _, chat := range chatMap {
		// Final filter to ensure no groups
		if !chat.IsGroup && !strings.Contains(chat.ChatJID, "@g.us") {
			updatedChats = append(updatedChats, *chat)
		}
	}
	
	// Sort by last message time (newest first)
	sort.Slice(updatedChats, func(i, j int) bool {
		return updatedChats[i].LastMessageTime.After(updatedChats[j].LastMessageTime)
	})
	
	fmt.Printf("Found %d personal chats for device %s\n", len(updatedChats), deviceID)
	
	return updatedChats, nil
}

// formatPhoneNumber formats a phone number for display
func formatPhoneNumber(phone string) string {
	if len(phone) > 10 {
		return fmt.Sprintf("+%s", phone)
	}
	return phone
}

// GetMessagesForChat fetches and saves messages for a specific chat
func GetMessagesForChat(deviceID, chatJID string, limit int) ([]repository.WhatsAppMessage, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	
	repo := repository.GetWhatsAppRepository()
	
	if err != nil {
		// If client not connected, return saved messages from database
		return repo.GetMessages(deviceID, chatJID, limit)
	}
	
	// Parse JID
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat JID: %v", err)
	}
	
	// Get messages from WhatsApp client
	// Note: This is a simplified version. In reality, you'd need to handle message history sync
	chat, err := client.Store.Chats.GetChat(jid)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %v", err)
	}
	
	var savedMessages []repository.WhatsAppMessage
	messageCount := 0
	
	for _, msg := range chat.Messages {
		if messageCount >= limit {
			break
		}
		
		if msg.Message == nil {
			continue
		}
		
		// Extract message text
		messageText := ""
		messageType := "text"
		
		if msg.Message.Conversation != nil {
			messageText = *msg.Message.Conversation
		} else if msg.Message.ExtendedTextMessage != nil && msg.Message.ExtendedTextMessage.Text != nil {
			messageText = *msg.Message.ExtendedTextMessage.Text
		} else if msg.Message.ImageMessage != nil {
			messageType = "image"
			if msg.Message.ImageMessage.Caption != nil {
				messageText = *msg.Message.ImageMessage.Caption
			}
		} else if msg.Message.VideoMessage != nil {
			messageType = "video"
			if msg.Message.VideoMessage.Caption != nil {
				messageText = *msg.Message.VideoMessage.Caption
			}
		}
		
		// Get sender info
		senderJID := msg.Info.Sender.String()
		senderName := ""
		if msg.Info.IsFromMe {
			senderName = "You"
		} else {
			contact, _ := client.Store.Contacts.GetContact(msg.Info.Sender)
			if contact != nil {
				senderName = contact.PushName
			} else {
				senderName = msg.Info.Sender.User
			}
		}
		
		// Create message record
		whatsappMsg := repository.WhatsAppMessage{
			DeviceID:    deviceID,
			ChatJID:     chatJID,
			MessageID:   msg.Info.ID,
			SenderJID:   senderJID,
			SenderName:  senderName,
			MessageText: messageText,
			MessageType: messageType,
			MediaURL:    "", // TODO: Implement media handling
			IsSent:      msg.Info.IsFromMe,
			IsRead:      msg.Receipt != nil && msg.Receipt.Type == types.ReceiptTypeRead,
			Timestamp:   msg.Info.Timestamp,
		}
		
		// Save to database
		if err := repo.SaveMessage(&whatsappMsg); err != nil {
			fmt.Printf("Error saving message %s: %v\n", msg.Info.ID, err)
			continue
		}
		
		savedMessages = append(savedMessages, whatsappMsg)
		messageCount++
	}
	
	// If we didn't get enough messages from memory, fetch from database
	if len(savedMessages) < limit {
		dbMessages, _ := repo.GetMessages(deviceID, chatJID, limit)
		return dbMessages, nil
	}
	
	return savedMessages, nil
}

// RegisterDeviceClient registers a WhatsApp client when a device connects
func RegisterDeviceClient(deviceID string, client *whatsmeow.Client) {
	cm := GetClientManager()
	cm.AddClient(deviceID, client)
	
	// Start syncing chats in background
	go func() {
		time.Sleep(2 * time.Second) // Wait for connection to stabilize
		_, err := GetChatsForDevice(deviceID)
		if err != nil {
			fmt.Printf("Error syncing chats for device %s: %v\n", deviceID, err)
		}
	}()
}

// UnregisterDeviceClient removes a WhatsApp client when a device disconnects
func UnregisterDeviceClient(deviceID string) {
	cm := GetClientManager()
	cm.RemoveClient(deviceID)
}
// GetAllPersonalChats attempts to get ALL personal chats, including those without recent messages
func GetAllPersonalChats(deviceID string) ([]repository.WhatsAppChat, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not connected")
	}
	
	var allPersonalChats []repository.WhatsAppChat
	chatMap := make(map[string]*repository.WhatsAppChat)
	
	// Method 1: Get all contacts from the contact store
	contacts, err := client.Store.Contacts.GetAllContacts()
	if err == nil {
		fmt.Printf("Found %d contacts in store\n", len(contacts))
		for jid, contact := range contacts {
			// Only process personal contacts
			if jid.Server != types.DefaultUserServer || jid.User == "status" {
				continue
			}
			
			chatJID := jid.String()
			chatName := ""
			
			// Get best available name
			if contact.PushName != "" {
				chatName = contact.PushName
			} else if contact.BusinessName != "" {
				chatName = contact.BusinessName
			} else if contact.FullName != "" {
				chatName = contact.FullName
			} else if contact.FirstName != "" || contact.GivenName != "" {
				firstName := contact.FirstName
				if firstName == "" {
					firstName = contact.GivenName
				}
				familyName := contact.FamilyName
				if familyName == "" {
					familyName = contact.FamilyName
				}
				chatName = strings.TrimSpace(firstName + " " + familyName)
			}
			
			if chatName == "" {
				chatName = formatPhoneNumber(jid.User)
			}
			
			chat := &repository.WhatsAppChat{
				DeviceID:        deviceID,
				ChatJID:         chatJID,
				ChatName:        chatName,
				IsGroup:         false,
				IsMuted:         false,
				LastMessageText: "",
				LastMessageTime: time.Now().Add(-365 * 24 * time.Hour),
				UnreadCount:     0,
			}
			
			chatMap[chatJID] = chat
		}
	}
	
	// Method 2: Get all chat settings (conversations with settings)
	conversations, err := client.Store.ChatSettings.GetAllChatSettings()
	if err == nil {
		fmt.Printf("Found %d conversations with settings\n", len(conversations))
		for jid, settings := range conversations {
			// Only process personal chats
			if jid.Server != types.DefaultUserServer || jid.User == "status" {
				continue
			}
			
			chatJID := jid.String()
			
			// Update existing or create new
			if chat, exists := chatMap[chatJID]; exists {
				chat.IsMuted = settings.Muted
			} else {
				// Create new entry
				contact, _ := client.Store.Contacts.GetContact(jid)
				chatName := ""
				if contact != nil && contact.PushName != "" {
					chatName = contact.PushName
				} else {
					chatName = formatPhoneNumber(jid.User)
				}
				
				chat := &repository.WhatsAppChat{
					DeviceID:        deviceID,
					ChatJID:         chatJID,
					ChatName:        chatName,
					IsGroup:         false,
					IsMuted:         settings.Muted,
					LastMessageText: "",
					LastMessageTime: time.Now().Add(-365 * 24 * time.Hour),
					UnreadCount:     0,
				}
				
				chatMap[chatJID] = chat
			}
		}
	}
	
	// Method 3: Get recent chats (those with recent activity)
	// This would require accessing the chat store differently
	// For now, we'll rely on contacts and chat settings
	
	// Save all to database and build result slice
	repo := repository.GetWhatsAppRepository()
	for _, chat := range chatMap {
		// Try to get existing chat from DB to preserve message info
		existingChat, err := repo.GetChatByJID(deviceID, chat.ChatJID)
		if err == nil && existingChat != nil {
			// Update only necessary fields
			existingChat.ChatName = chat.ChatName
			existingChat.IsMuted = chat.IsMuted
			chat = existingChat
		}
		
		// Save to database
		if err := repo.SaveOrUpdateChat(chat); err != nil {
			fmt.Printf("Error saving chat %s: %v\n", chat.ChatJID, err)
		}
		
		allPersonalChats = append(allPersonalChats, *chat)
	}
	
	// Sort by last message time (newest first)
	sort.Slice(allPersonalChats, func(i, j int) bool {
		return allPersonalChats[i].LastMessageTime.After(allPersonalChats[j].LastMessageTime)
	})
	
	fmt.Printf("Total personal chats found: %d\n", len(allPersonalChats))
	
	return allPersonalChats, nil
}
