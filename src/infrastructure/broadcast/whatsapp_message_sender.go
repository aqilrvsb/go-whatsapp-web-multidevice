package broadcast

import (
	"context"
	"fmt"
	"strings"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// WhatsAppMessageSender handles actual WhatsApp message sending
type WhatsAppMessageSender struct {
	clientManager *whatsapp.ClientManager
}

// NewWhatsAppMessageSender creates a new message sender
func NewWhatsAppMessageSender() *WhatsAppMessageSender {
	return &WhatsAppMessageSender{
		clientManager: whatsapp.GetClientManager(),
	}
}

// SendMessage sends a message via WhatsApp
func (w *WhatsAppMessageSender) SendMessage(deviceID string, msg *broadcast.BroadcastMessage) error {
	// Get WhatsApp client for device
	waClient, err := w.clientManager.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	if !waClient.IsConnected() {
		return fmt.Errorf("device %s is not connected to WhatsApp", deviceID)
	}
	
	// Parse recipient JID
	recipientJID, err := types.ParseJID(msg.RecipientPhone + "@s.whatsapp.net")
	if err != nil {
		// Try without suffix
		recipientJID, err = types.ParseJID(msg.RecipientPhone)
		if err != nil {
			return fmt.Errorf("invalid recipient phone: %v", err)
		}
	}
	
	// Validate recipient
	if !strings.ContainsRune(recipientJID.User, '@') {
		info, err := waClient.IsOnWhatsApp([]string{recipientJID.User})
		if err != nil {
			return fmt.Errorf("failed to check WhatsApp: %v", err)
		}
		if len(info) == 0 || !info[0].IsIn {
			return fmt.Errorf("recipient %s is not on WhatsApp", msg.RecipientPhone)
		}
	}
	
	// Send based on message type
	if msg.Type == "image" && msg.ImageURL != "" {
		return w.sendImageMessage(waClient, recipientJID, msg)
	} else {
		return w.sendTextMessage(waClient, recipientJID, msg)
	}
}

// sendTextMessage sends a text message
func (w *WhatsAppMessageSender) sendTextMessage(waClient *whatsmeow.Client, recipient types.JID, msg *broadcast.BroadcastMessage) error {
	// Create message
	message := &waE2E.Message{
		Conversation: proto.String(msg.Message),
	}
	
	// Send message
	resp, err := waClient.SendMessage(context.Background(), recipient, message)
	if err != nil {
		return fmt.Errorf("failed to send text message: %v", err)
	}
	
	logrus.Infof("Text message sent to %s (ID: %s)", recipient.String(), resp.ID)
	return nil
}

// sendImageMessage sends an image message
func (w *WhatsAppMessageSender) sendImageMessage(waClient *whatsmeow.Client, recipient types.JID, msg *broadcast.BroadcastMessage) error {
	// For now, send as text with image URL
	// TODO: Implement actual image upload and sending
	
	messageText := msg.Message
	if messageText == "" {
		messageText = "📷 Image"
	}
	messageText += fmt.Sprintf("\n\n🔗 %s", msg.ImageURL)
	
	message := &waE2E.Message{
		Conversation: proto.String(messageText),
	}
	
	resp, err := waClient.SendMessage(context.Background(), recipient, message)
	if err != nil {
		return fmt.Errorf("failed to send image message: %v", err)
	}
	
	logrus.Infof("Image message sent to %s (ID: %s)", recipient.String(), resp.ID)
	return nil
}
