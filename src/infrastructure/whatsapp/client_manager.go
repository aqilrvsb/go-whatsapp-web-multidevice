package whatsapp

import (
	"fmt"
	"sort"
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

// GetChatsForDevice fetches and saves chats for a specific device
func GetChatsForDevice(deviceID string) ([]repository.WhatsAppChat, error) {
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	
	// Always try to get from database first
	repo := repository.GetWhatsAppRepository()
	savedChats, _ := repo.GetChats(deviceID)
	
	if err != nil {
		// If client not connected, return saved chats from database
		return savedChats, nil
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
		return savedChats, nil
	}
	
	// Create a map for quick updates
	chatMap := make(map[string]*repository.WhatsAppChat)
	for i := range savedChats {
		chatMap[savedChats[i].ChatJID] = &savedChats[i]
	}
	
	// Process each conversation
	for jid, settings := range conversations {
		chatJID := jid.String()
		
		// Get or create chat entry
		chat, exists := chatMap[chatJID]
		if !exists {
			chat = &repository.WhatsAppChat{
				DeviceID: deviceID,
				ChatJID:  chatJID,
			}
			chatMap[chatJID] = chat
		}
		
		// Update chat info
		chat.IsGroup = jid.Server == types.GroupServer
		chat.IsMuted = settings.Muted
		
		// Get chat name
		if chat.IsGroup {
			// For groups, get group info
			groupInfo, err := client.GetGroupInfo(jid)
			if err == nil && groupInfo != nil {
				chat.ChatName = groupInfo.Name
			}
		} else {
			// For individual chats, use contact name or push name
			if contact, ok := contacts[jid]; ok && contact.PushName != "" {
				chat.ChatName = contact.PushName
			} else if contact, ok := contacts[jid]; ok && contact.BusinessName != "" {
				chat.ChatName = contact.BusinessName
			} else {
				// Fallback to phone number
				chat.ChatName = jid.User
			}
		}
		
		// Try to get last message from history sync
		// Note: This is a simplified approach. In production, you'd handle history sync events
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
	
	// Convert map back to slice
	var updatedChats []repository.WhatsAppChat
	for _, chat := range chatMap {
		updatedChats = append(updatedChats, *chat)
	}
	
	// Sort by last message time (newest first)
	sort.Slice(updatedChats, func(i, j int) bool {
		return updatedChats[i].LastMessageTime.After(updatedChats[j].LastMessageTime)
	})
	
	return updatedChats, nil
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
