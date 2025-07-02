package whatsapp

import (
	"context"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"time"
)

// Context key for device ID
type contextKey string

const deviceIDKey contextKey = "deviceID"

// WithDeviceID adds device ID to context
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, deviceIDKey, deviceID)
}

// GetDeviceIDFromContext retrieves device ID from context
func GetDeviceIDFromContext(ctx context.Context) string {
	if deviceID, ok := ctx.Value(deviceIDKey).(string); ok {
		return deviceID
	}
	return ""
}

// StoreWhatsAppMessage stores a message in the whatsapp_messages table
func StoreWhatsAppMessage(deviceID, chatJID, messageID, senderJID, messageText, messageType string) {
	StoreWhatsAppMessageWithTimestamp(deviceID, chatJID, messageID, senderJID, messageText, messageType, time.Now().Unix())
}

// StoreWhatsAppMessageWithTimestamp stores a message with specific timestamp
func StoreWhatsAppMessageWithTimestamp(deviceID, chatJID, messageID, senderJID, messageText, messageType string, timestamp int64) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// If sender is empty, it's a sent message
	if senderJID == "" {
		// Get our JID from client
		cm := GetClientManager()
		client, err := cm.GetClient(deviceID)
		if err != nil {
			logrus.Warnf("Failed to get client for storing message: %v", err)
			return
		}
		if client.Store.ID != nil {
			senderJID = client.Store.ID.String()
		}
	}
	
	// Insert message (trigger will handle cleanup if exists)
	query := `
		INSERT INTO whatsapp_messages (device_id, chat_jid, message_id, sender_jid, message_text, message_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, message_id) DO UPDATE SET
			message_text = EXCLUDED.message_text,
			message_type = EXCLUDED.message_type,
			timestamp = EXCLUDED.timestamp
	`
	
	_, err := db.Exec(query, deviceID, chatJID, messageID, senderJID, messageText, messageType, timestamp)
	if err != nil {
		logrus.Warnf("Failed to store message: %v", err)
	}
}

// ImageRequest extension for WhatsApp Web
type ImageRequestExt struct {
	ImageB64   string
	ImageBytes []byte
}
