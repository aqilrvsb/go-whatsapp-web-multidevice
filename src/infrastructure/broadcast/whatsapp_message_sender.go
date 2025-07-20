package broadcast

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/external"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/antipattern"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// WhatsAppMessageSender handles actual WhatsApp message sending
type WhatsAppMessageSender struct {
	clientManager     *whatsapp.ClientManager
	platformSender    *external.PlatformSender
	userRepo         *repository.UserRepository
	messageRandomizer *antipattern.MessageRandomizer
	greetingProcessor *antipattern.GreetingProcessor
}

// NewWhatsAppMessageSender creates a new message sender
func NewWhatsAppMessageSender() *WhatsAppMessageSender {
	return &WhatsAppMessageSender{
		clientManager:     whatsapp.GetClientManager(),
		platformSender:    external.NewPlatformSender(),
		userRepo:         repository.GetUserRepository(),
		messageRandomizer: antipattern.NewMessageRandomizer(),
		greetingProcessor: antipattern.NewGreetingProcessor(),
	}
}

// SendMessage sends a message via WhatsApp or external platform
func (w *WhatsAppMessageSender) SendMessage(deviceID string, msg *broadcast.BroadcastMessage) error {
	// Get device details to check platform
	device, err := w.userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %v", err)
	}
	
	// Check if device has platform
	if device.Platform != "" {
		// Send via external platform
		logrus.Infof("[PLATFORM-SEND] 📤 Sending message via %s platform for device %s (%s)", 
			device.Platform, device.DeviceName, deviceID)
		logrus.Infof("[PLATFORM-SEND] Recipient: %s, Message type: %s", 
			msg.RecipientPhone, msg.Type)
		
		// Get instance/token from device JID (or you can add a separate column)
		instance := device.JID // Using JID as instance/token
		
		logrus.Debugf("[PLATFORM-SEND] Using instance/token: %s", instance)
		
		startTime := time.Now()
		err = w.platformSender.SendMessage(
			device.Platform,
			instance,
			msg.RecipientPhone,
			msg.RecipientName,
			msg.Message,
			msg.ImageURL,
			deviceID,
		)
		duration := time.Since(startTime)
		
		if err != nil {
			logrus.Errorf("[PLATFORM-SEND] ❌ Failed to send via %s platform: %v (took %v)", 
				device.Platform, err, duration)
			return fmt.Errorf("platform send failed: %v", err)
		}
		
		logrus.Infof("[PLATFORM-SEND] ✅ Successfully sent message via %s platform to %s (took %v)", 
			device.Platform, msg.RecipientPhone, duration)
		return nil
	}
	
	// Normal WhatsApp sending
	return w.sendViaWhatsApp(deviceID, msg)
}

// sendViaWhatsApp sends message via normal WhatsApp
func (w *WhatsAppMessageSender) sendViaWhatsApp(deviceID string, msg *broadcast.BroadcastMessage) error {
	// Get WhatsApp client for device
	waClient, err := w.clientManager.GetClient(deviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	if !waClient.IsConnected() {
		return fmt.Errorf("device %s is not connected to WhatsApp", deviceID)
	}
	
	// Check if client is still connected before sending
	if !waClient.IsConnected() {
		logrus.Warnf("Device %s disconnected, attempting to reconnect before sending", deviceID)
		
		// Try to reconnect
		err := waClient.Connect()
		if err != nil {
			return fmt.Errorf("device %s failed to reconnect: %v", deviceID, err)
		}
		
		// Wait a bit for connection to stabilize
		time.Sleep(2 * time.Second)
		
		// Check again
		if !waClient.IsConnected() {
			return fmt.Errorf("device %s still not connected after reconnection attempt", deviceID)
		}
		
		logrus.Infof("Device %s reconnected successfully", deviceID)
	}
	
	if !waClient.IsLoggedIn() {
		return fmt.Errorf("device %s is not logged in", deviceID)
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
	// Get device ID from the message context
	deviceID := msg.DeviceID
	
	// UPDATED: Use RecipientName directly from broadcast_messages
	// This ensures consistency for both campaigns and sequences
	nameToUse := msg.RecipientName
	if nameToUse == "" {
		nameToUse = msg.RecipientPhone // Fallback to phone if name is empty
	}
	
	// Log what name we're using
	logrus.Infof("[WHATSAPP-NAME] Using recipient_name from broadcast_messages: '%s' for phone %s", nameToUse, msg.RecipientPhone)
	
	// Prepare message with greeting using the name from broadcast_messages
	messageWithGreeting := w.greetingProcessor.PrepareMessageWithGreeting(
		msg.Message, 
		nameToUse,  // Use the name from broadcast_messages
		deviceID, 
		msg.RecipientPhone,
	)
	
	// Apply anti-pattern techniques to the message
	randomizedMessage := w.messageRandomizer.RandomizeMessage(messageWithGreeting)
	
	// Add typing delay for human-like behavior (but no presence)
	typingDelay := antipattern.AddTypingDelay(len(msg.Message))
	logrus.Debugf("Simulating typing delay for %v", typingDelay)
	
	// Just wait, don't send presence
	time.Sleep(typingDelay)
	
	// Create message with randomized content
	message := &waE2E.Message{
		Conversation: proto.String(randomizedMessage),
	}
	
	// Send message
	resp, err := waClient.SendMessage(context.Background(), recipient, message)
	if err != nil {
		return fmt.Errorf("failed to send text message: %v", err)
	}
	
	// No presence update after sending
	
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
	
	// Apply anti-pattern to caption if exists
	caption := msg.Message
	if caption != "" {
		caption = w.messageRandomizer.RandomizeMessage(caption)
		
		// Add typing delay for caption (but no presence)
		typingDelay := antipattern.AddTypingDelay(len(caption))
		time.Sleep(typingDelay)
	}
	
	// Create image message
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
	
	// No presence update after sending
	
	logrus.Infof("Image message sent to %s (ID: %s)", recipient.String(), resp.ID)
	
	// Note: Delay should be handled by the broadcast worker based on campaign settings
	// Don't add hardcoded delays here
	
	return nil
}
