package whatsapp

import (
	"fmt"
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
	if err != nil {
		// If client not connected, return saved chats from database
		repo := repository.GetWhatsAppRepository()
		return repo.GetChats(deviceID)
	}
	
	// Get chats from WhatsApp client
	chats, err := client.Store.Chats.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %v", err)
	}
	
	// Convert and save chats to database
	repo := repository.GetWhatsAppRepository()
	var savedChats []repository.WhatsAppChat
	
	for jid, chat := range chats {
		// Skip invalid chats
		if chat == nil {
			continue
		}
		
		// Get chat name
		chatName := ""
		if chat.Name != "" {
			chatName = chat.Name
		} else {
			// For individual chats, get contact name
			contact, _ := client.Store.Contacts.GetContact(jid)
			if contact != nil {
				if contact.PushName != "" {
					chatName = contact.PushName
				} else if contact.BusinessName != "" {
					chatName = contact.BusinessName
				} else {
					chatName = jid.User
				}
			} else {
				chatName = jid.User
			}
		}
		
		// Get last message info
		lastMessageText := ""
		lastMessageTime := time.Now()
		if len(chat.Messages) > 0 {
			// Get the most recent message
			for _, msg := range chat.Messages {
				if msg.Message != nil && msg.Info.Timestamp.After(lastMessageTime) {
					lastMessageTime = msg.Info.Timestamp
					if msg.Message.Conversation != nil {
						lastMessageText = *msg.Message.Conversation
					} else if msg.Message.ExtendedTextMessage != nil && msg.Message.ExtendedTextMessage.Text != nil {
						lastMessageText = *msg.Message.ExtendedTextMessage.Text
					}
				}
			}
		}
		
		// Create chat record
		whatsappChat := repository.WhatsAppChat{
			DeviceID:        deviceID,
			ChatJID:         jid.String(),
			ChatName:        chatName,
			IsGroup:         jid.Server == types.GroupServer,
			IsMuted:         chat.Muted,
			LastMessageText: lastMessageText,
			LastMessageTime: lastMessageTime,
			UnreadCount:     int(chat.UnreadCount),
			AvatarURL:       "", // TODO: Implement avatar fetching
		}
		
		// Save to database
		if err := repo.SaveOrUpdateChat(&whatsappChat); err != nil {
			// Log error but continue with other chats
			fmt.Printf("Error saving chat %s: %v\n", jid.String(), err)
			continue
		}
		
		savedChats = append(savedChats, whatsappChat)
	}
	
	return savedChats, nil
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
