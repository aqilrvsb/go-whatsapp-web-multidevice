package broadcast

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
	var imageData []byte
	var err error
	
	// Check if it's a data URL or regular URL
	if strings.HasPrefix(msg.ImageURL, "data:") {
		// Handle data URL (base64 encoded image)
		parts := strings.SplitN(msg.ImageURL, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid data URL format")
		}
		
		// Decode base64 data
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("failed to decode base64 image: %v", err)
		}
	} else {
		// Handle regular URL
		resp, err := http.Get(msg.ImageURL)
		if err != nil {
			return fmt.Errorf("failed to download image: %v", err)
		}
		defer resp.Body.Close()
		
		imageData, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read image data: %v", err)
		}
	}
	
	// Upload to WhatsApp servers
	uploaded, err := waClient.Upload(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %v", err)
	}
	
	// Create image message
	caption := msg.Message
	message := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Mimetype:      proto.String("image/jpeg"),
		},
	}
	
	// Send message
	resp, err := waClient.SendMessage(context.Background(), recipient, message)
	if err != nil {
		return fmt.Errorf("failed to send image message: %v", err)
	}
	
	logrus.Infof("Image message sent to %s (ID: %s)", recipient.String(), resp.ID)
	return nil
}
