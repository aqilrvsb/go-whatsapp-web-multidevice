package broadcast

import (
	"context"
	"fmt"
	"time"
	
	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// SendWhatsAppMessage sends a message using the WhatsApp client
func SendWhatsAppMessage(client *whatsmeow.Client, msg domainBroadcast.BroadcastMessage) error {
	// Sanitize phone number
	phone := msg.RecipientPhone
	whatsapp.SanitizePhone(&phone)
	
	recipient, err := types.ParseJID(phone + "@s.whatsapp.net")
	if err != nil {
		logrus.Errorf("Invalid phone number %s: %v", phone, err)
		return err
	}
	
	logrus.Infof("Sending %s message to %s", msg.Type, recipient.String())
	
	// Handle different message types
	switch msg.Type {
	case "text":
		if msg.MediaURL != "" {
			// Image with caption
			logrus.Infof("Sending image with caption to %s", recipient.String())
			
			imageBytes, err := whatsapp.DownloadMedia(msg.MediaURL)
			if err != nil {
				logrus.Errorf("Failed to download image: %v", err)
				return err
			}
			
			uploadResp, err := client.Upload(context.Background(), imageBytes, whatsmeow.MediaImage)
			if err != nil {
				logrus.Errorf("Failed to upload image: %v", err)
				return err
			}
			
			imageMsg := &waProto.ImageMessage{
				Caption:       proto.String(msg.Content),
				Url:           proto.String(uploadResp.URL),
				DirectPath:    proto.String(uploadResp.DirectPath),
				MediaKey:      uploadResp.MediaKey,
				Mimetype:      proto.String("image/jpeg"),
				FileEncSha256: uploadResp.FileEncSHA256,
				FileSha256:    uploadResp.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(imageBytes))),
			}
			
			_, err = client.SendMessage(context.Background(), recipient, &waProto.Message{
				ImageMessage: imageMsg,
			})
			
			if err != nil {
				logrus.Errorf("Failed to send image message: %v", err)
				return err
			}
			
			logrus.Infof("Successfully sent image with caption to %s", recipient.String())
		} else {
			// Text only
			logrus.Infof("Sending text message to %s", recipient.String())
			
			_, err = client.SendMessage(context.Background(), recipient, &waProto.Message{
				Conversation: proto.String(msg.Content),
			})
			
			if err != nil {
				logrus.Errorf("Failed to send text message: %v", err)
				return err
			}
			
			logrus.Infof("Successfully sent text message to %s", recipient.String())
		}
		
	case "image":
		// Image only (no caption)
		logrus.Infof("Sending image to %s", recipient.String())
		
		imageBytes, err := whatsapp.DownloadMedia(msg.MediaURL)
		if err != nil {
			logrus.Errorf("Failed to download image: %v", err)
			return err
		}
		
		uploadResp, err := client.Upload(context.Background(), imageBytes, whatsmeow.MediaImage)
		if err != nil {
			logrus.Errorf("Failed to upload image: %v", err)
			return err
		}
		
		imageMsg := &waProto.ImageMessage{
			Url:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String("image/jpeg"),
			FileEncSha256: uploadResp.FileEncSHA256,
			FileSha256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageBytes))),
		}
		
		_, err = client.SendMessage(context.Background(), recipient, &waProto.Message{
			ImageMessage: imageMsg,
		})
		
		if err != nil {
			logrus.Errorf("Failed to send image: %v", err)
			return err
		}
		
		logrus.Infof("Successfully sent image to %s", recipient.String())
		
	default:
		logrus.Warnf("Unknown message type: %s", msg.Type)
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
	
	// Wait for the delay
	delay := whatsapp.GetRandomDelay(msg.MinDelay, msg.MaxDelay)
	logrus.Infof("Waiting %v before next message", delay)
	time.Sleep(delay)
	
	return nil
}
