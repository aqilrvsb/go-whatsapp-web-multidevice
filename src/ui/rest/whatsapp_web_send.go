package rest

import (
	"fmt"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"strings"
)

// SendWhatsAppWebMessage handles sending messages from WhatsApp Web view
func (handler *App) SendWhatsAppWebMessage(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Message request structure
	type MessageRequest struct {
		ChatID   string `json:"chatId"`
		Message  string `json:"message"`
		ImageURL string `json:"imageUrl,omitempty"`
		ImageB64 string `json:"imageB64,omitempty"`
	}
	
	var request MessageRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request format",
		})
	}
	
	// Validate request
	if request.ChatID == "" || (request.Message == "" && request.ImageURL == "" && request.ImageB64 == "") {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "ChatID and either message or image is required",
		})
	}
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token found",
		})
	}
	
	// Verify session and device ownership
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil || session == nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	// Get user from database
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Get user devices to check if this device belongs to user
	devices, err := userRepo.GetUserDevices(user.ID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}
	
	// Check if device belongs to user and is online
	deviceBelongsToUser := false
	isOnline := false
	for _, device := range devices {
		if device.ID == deviceId {
			deviceBelongsToUser = true
			isOnline = device.Status == "online"
			break
		}
	}
	
	if !deviceBelongsToUser {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "Device does not belong to user",
		})
	}
	
	if !isOnline {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "DEVICE_OFFLINE",
			Message: "Device is offline",
		})
	}
	
	// Get the send service
	if handler.Send == nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Send service not available",
		})
	}
	
	// Parse JID for recipient
	recipientJID, err := types.ParseJID(request.ChatID)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_JID",
			Message: fmt.Sprintf("Invalid chat ID format: %v", err),
		})
	}
	
	// Prepare phone number from JID
	phone := recipientJID.User
	if !strings.Contains(phone, "@") {
		phone = fmt.Sprintf("%s@%s", recipientJID.User, recipientJID.Server)
	}
	
	// Send message based on type
	if request.ImageURL != "" || request.ImageB64 != "" {
		// Get WhatsApp client directly
		cm := whatsapp.GetClientManager()
		client, err := cm.GetClient(deviceId)
		if err != nil || client == nil || !client.IsConnected() {
			return c.Status(400).JSON(utils.ResponseData{
				Status:  400,
				Code:    "DEVICE_OFFLINE",
				Message: "Device is not connected",
			})
		}
		
		var messageID string
		
		// Send image based on source
		if request.ImageB64 != "" {
			messageID, err = whatsapp.SendImageFromWeb(c.UserContext(), client, recipientJID, request.ImageB64, request.Message)
			if err != nil {
				return c.Status(500).JSON(utils.ResponseData{
					Status:  500,
					Code:    "SEND_FAILED",
					Message: fmt.Sprintf("Failed to send image: %v", err),
				})
			}
		} else if request.ImageURL != "" {
			messageID, err = whatsapp.SendImageFromURL(c.UserContext(), client, recipientJID, request.ImageURL, request.Message)
			if err != nil {
				return c.Status(500).JSON(utils.ResponseData{
					Status:  500,
					Code:    "SEND_FAILED",
					Message: fmt.Sprintf("Failed to send image: %v", err),
				})
			}
		}
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Image sent successfully",
			Results: map[string]interface{}{
				"messageId": messageID,
				"status":    "sent",
			},
		})
		
	} else {
		// Send text message directly using WhatsApp client
		cm := whatsapp.GetClientManager()
		client, err := cm.GetClient(deviceId)
		if err != nil || client == nil || !client.IsConnected() {
			return c.Status(400).JSON(utils.ResponseData{
				Status:  400,
				Code:    "DEVICE_OFFLINE",
				Message: "Device is not connected",
			})
		}
		
		// Send text message
		resp, err := client.SendMessage(c.UserContext(), recipientJID, &waE2E.Message{
			Conversation: proto.String(request.Message),
		})
		
		if err != nil {
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "SEND_FAILED",
				Message: fmt.Sprintf("Failed to send message: %v", err),
			})
		}
		
		// Store in messages table
		go whatsapp.StoreWhatsAppMessage(deviceId, request.ChatID, resp.ID, client.Store.ID.String(), request.Message, "text")
		
		// Notify WebSocket
		go whatsapp.NotifyMessageUpdate(deviceId, request.ChatID, map[string]interface{}{
			"id":        resp.ID,
			"text":      request.Message,
			"type":      "text",
			"sent":      true,
			"time":      time.Now().Format("15:04"),
			"timestamp": time.Now().Unix(),
		})
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Message sent successfully",
			Results: map[string]interface{}{
				"messageId": resp.ID,
				"status":    "sent",
			},
		})
	}
}
