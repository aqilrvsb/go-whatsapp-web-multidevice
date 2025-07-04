package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// SendImageFromWeb handles image sending from WhatsApp Web interface
func SendImageFromWeb(ctx context.Context, client *whatsmeow.Client, recipientJID types.JID, imageB64 string, caption string) (string, error) {
	// Extract base64 data (remove data:image/jpeg;base64, prefix if present)
	b64Data := imageB64
	if strings.Contains(b64Data, ",") {
		parts := strings.Split(b64Data, ",")
		if len(parts) > 1 {
			b64Data = parts[1]
		}
	}
	
	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %v", err)
	}
	
	// Upload the image
	uploadResp, err := client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %v", err)
	}
	
	// Create image message
	imageMsg := &waE2E.ImageMessage{
		Caption:       proto.String(caption),
		URL:           proto.String(uploadResp.URL),
		DirectPath:    proto.String(uploadResp.DirectPath),
		MediaKey:      uploadResp.MediaKey,
		Mimetype:      proto.String(http.DetectContentType(imageData)),
		FileEncSHA256: uploadResp.FileEncSHA256,
		FileSHA256:    uploadResp.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(imageData))),
	}
	
	// Send message
	resp, err := client.SendMessage(ctx, recipientJID, &waE2E.Message{
		ImageMessage: imageMsg,
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to send image: %v", err)
	}
	
	// Store the message with media URL
	go func() {
		// Store with the upload URL so we can display it later
		StoreWhatsAppMessageWithMedia(client.Store.ID.String(), recipientJID.String(), resp.ID, client.Store.ID.String(), caption, "image", uploadResp.URL)
		
		// Notify WebSocket
		NotifyMessageUpdate(client.Store.ID.String(), recipientJID.String(), "Image sent")
	}()
	
	return resp.ID, nil
}

// SendImageFromURL handles image sending from URL
func SendImageFromURL(ctx context.Context, client *whatsmeow.Client, recipientJID types.JID, imageURL string, caption string) (string, error) {
	// Download image from URL
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()
	
	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %v", err)
	}
	
	// Upload the image
	uploadResp, err := client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %v", err)
	}
	
	// Create image message
	imageMsg := &waE2E.ImageMessage{
		Caption:       proto.String(caption),
		URL:           proto.String(uploadResp.URL),
		DirectPath:    proto.String(uploadResp.DirectPath),
		MediaKey:      uploadResp.MediaKey,
		Mimetype:      proto.String(resp.Header.Get("Content-Type")),
		FileEncSHA256: uploadResp.FileEncSHA256,
		FileSHA256:    uploadResp.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(imageData))),
	}
	
	// Send message
	msgResp, err := client.SendMessage(ctx, recipientJID, &waE2E.Message{
		ImageMessage: imageMsg,
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to send image: %v", err)
	}
	
	// Store the message with media URL
	go func() {
		StoreWhatsAppMessageWithMedia(client.Store.ID.String(), recipientJID.String(), msgResp.ID, client.Store.ID.String(), caption, "image", uploadResp.URL)
		
		// Notify WebSocket
		NotifyMessageUpdate(client.Store.ID.String(), recipientJID.String(), "Image sent")
	}()
	
	return msgResp.ID, nil
}


