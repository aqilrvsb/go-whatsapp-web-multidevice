package whatsapp

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// HandleMessageForWebView stores messages for WhatsApp Web view
// Only stores personal chat messages (no groups)
func HandleMessageForWebView(deviceID string, evt *events.Message) {
	// Skip group messages - we only want personal chats
	if evt.Info.IsGroup || evt.Info.Chat.Server != types.DefaultUserServer {
		return
	}
	
	// Skip status updates
	if evt.Info.Chat.User == "status" {
		return
	}
	
	// Extract message text
	messageText := extractMessageText(evt)
	messageType := "text"
	
	// Check for different message types
	if evt.Message.GetImageMessage() != nil {
		messageType = "image"
		if caption := evt.Message.GetImageMessage().GetCaption(); caption != "" {
			messageText = caption
		}
	} else if evt.Message.GetVideoMessage() != nil {
		messageType = "video"
		if caption := evt.Message.GetVideoMessage().GetCaption(); caption != "" {
			messageText = caption
		}
	} else if evt.Message.GetAudioMessage() != nil {
		messageType = "audio"
	} else if evt.Message.GetDocumentMessage() != nil {
		messageType = "document"
		if fileName := evt.Message.GetDocumentMessage().GetFileName(); fileName != "" {
			messageText = "📄 " + fileName
		}
	}
	
	// Store message using the new function
	StoreWhatsAppMessage(
		deviceID, 
		evt.Info.Chat.String(), 
		evt.Info.ID, 
		evt.Info.Sender.String(), 
		messageText, 
		messageType,
	)
	
	logrus.Debugf("Stored %s message from %s in chat %s", messageType, evt.Info.Sender.String(), evt.Info.Chat.String())
}

// HandleHistorySyncForWebView processes history sync to get recent messages
func HandleHistorySyncForWebView(deviceID string, evt *events.HistorySync) {
	logrus.Infof("Processing history sync for WhatsApp Web view - device %s", deviceID)
	
	count := 0
	for _, conv := range evt.Data.Conversations {
		// Skip groups
		if conv.ID != nil && (strings.Contains(*conv.ID, "@g.us") || strings.Contains(*conv.ID, "@broadcast")) {
			continue
		}
		
		// Process only recent 20 messages per chat
		msgCount := 0
		for i := len(conv.Messages) - 1; i >= 0 && msgCount < 20; i-- {
			msg := conv.Messages[i]
			if msg.Message == nil || msg.Message.Key == nil {
				continue
			}
			
			// Extract message info
			messageID := msg.Message.Key.GetId()
			chatJID := msg.Message.Key.GetRemoteJid()
			senderJID := msg.Message.Key.GetParticipant()
			if senderJID == "" && msg.Message.Key.GetFromMe() {
				senderJID = msg.Message.Key.GetRemoteJid()
			}
			
			messageText := extractMessageFromProto(msg.Message.Message)
			timestamp := time.Unix(int64(msg.Message.GetMessageTimestamp()), 0)
			isFromMe := msg.Message.Key.GetFromMe()
			
			storeMessage(deviceID, chatJID, messageID, senderJID, messageText, timestamp, isFromMe)
			msgCount++
			count++
		}
	}
	
	logrus.Infof("Stored %d personal chat messages for device %s", count, deviceID)
}

// GetMessagesForChatWeb retrieves recent messages for a chat (WhatsApp Web view)
func GetMessagesForChatWeb(deviceID string, chatJID string) ([]map[string]interface{}, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Get last 20 messages for this chat
	query := `
		SELECT message_id, sender_jid, message_text, timestamp, is_from_me
		FROM whatsapp_messages
		WHERE device_id = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT 20
	`
	
	rows, err := db.Query(query, deviceID, chatJID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %v", err)
	}
	defer rows.Close()
	
	var messages []map[string]interface{}
	
	for rows.Next() {
		var messageID, senderJID, messageText string
		var timestamp int64
		var isFromMe bool
		
		err := rows.Scan(&messageID, &senderJID, &messageText, &timestamp, &isFromMe)
		if err != nil {
			continue
		}
		
		msgTime := time.Unix(timestamp, 0)
		
		messages = append(messages, map[string]interface{}{
			"id":        messageID,
			"text":      messageText,
			"fromMe":    isFromMe,
			"time":      msgTime.Format("3:04 PM"),
			"timestamp": timestamp,
			"status":    "sent",
		})
	}
	
	// Reverse to show oldest first (like WhatsApp Web)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	
	return messages, nil
}

// Helper functions

func storeMessage(deviceID, chatJID, messageID, senderJID, messageText string, 
	timestamp time.Time, isFromMe bool) {
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	query := `
		INSERT INTO whatsapp_messages 
		(device_id, chat_jid, message_id, sender_jid, message_text, timestamp, is_from_me)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, chat_jid, message_id) DO NOTHING
	`
	
	_, err := db.Exec(query, deviceID, chatJID, messageID, senderJID, 
		messageText, timestamp.Unix(), isFromMe)
	
	if err != nil {
		logrus.Debugf("Failed to store message: %v", err)
	}
}

func extractMessageText(evt *events.Message) string {
	if evt.Message.GetConversation() != "" {
		return evt.Message.GetConversation()
	}
	
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	
	// Media messages
	if img := evt.Message.GetImageMessage(); img != nil {
		if img.GetCaption() != "" {
			return "📷 " + img.GetCaption()
		}
		return "📷 Photo"
	}
	
	if vid := evt.Message.GetVideoMessage(); vid != nil {
		return "📹 Video"
	}
	
	if aud := evt.Message.GetAudioMessage(); aud != nil {
		return "🎵 Voice message"
	}
	
	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		return "📄 Document"
	}
	
	return ""
}

func extractMessageFromProto(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	
	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}
	
	// Media messages
	if msg.ImageMessage != nil {
		if msg.ImageMessage.Caption != nil {
			return "📷 " + *msg.ImageMessage.Caption
		}
		return "📷 Photo"
	}
	
	if msg.VideoMessage != nil {
		return "📹 Video"
	}
	
	if msg.AudioMessage != nil {
		return "🎵 Voice message"
	}
	
	if msg.DocumentMessage != nil {
		return "📄 Document"
	}
	
	return ""
}