package rest

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	"go.mau.fi/whatsmeow/types"
	"encoding/base64"
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
		// Send image message
		imageReq := send.ImageRequest{
			Phone:   phone,
			Caption: request.Message,
		}
		
		// Handle base64 image
		if request.ImageB64 != "" {
			// Extract base64 data (remove data:image/jpeg;base64, prefix if present)
			b64Data := request.ImageB64
			if strings.Contains(b64Data, ",") {
				parts := strings.Split(b64Data, ",")
				if len(parts) > 1 {
					b64Data = parts[1]
				}
			}
			
			// Decode base64
			imageData, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				return c.Status(400).JSON(utils.ResponseData{
					Status:  400,
					Code:    "INVALID_IMAGE",
					Message: "Failed to decode base64 image",
				})
			}
			
			// Store as temporary variable accessible to send service
			imageReq.ImageB64 = b64Data
			imageReq.ImageBytes = imageData
		} else if request.ImageURL != "" {
			imageReq.ImageURL = request.ImageURL
		}
		
		// Use context to pass device ID
		ctx := c.UserContext()
		ctx = whatsapp.WithDeviceID(ctx, deviceId)
		
		response, err := handler.Send.Service.SendImage(ctx, imageReq)
		if err != nil {
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "SEND_FAILED",
				Message: fmt.Sprintf("Failed to send image: %v", err),
			})
		}
		
		// Store in messages table
		go whatsapp.StoreWhatsAppMessage(deviceId, request.ChatID, response.MessageID, "", request.Message, "image")
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Image sent successfully",
			Results: map[string]interface{}{
				"messageId": response.MessageID,
				"status":    response.Status,
			},
		})
		
	} else {
		// Send text message
		textReq := send.MessageRequest{
			Phone:   phone,
			Message: request.Message,
		}
		
		// Use context to pass device ID
		ctx := c.UserContext()
		ctx = whatsapp.WithDeviceID(ctx, deviceId)
		
		response, err := handler.Send.Service.SendText(ctx, textReq)
		if err != nil {
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "SEND_FAILED",
				Message: fmt.Sprintf("Failed to send message: %v", err),
			})
		}
		
		// Store in messages table
		go whatsapp.StoreWhatsAppMessage(deviceId, request.ChatID, response.MessageID, "", request.Message, "text")
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Message sent successfully",
			Results: map[string]interface{}{
				"messageId": response.MessageID,
				"status":    response.Status,
			},
		})
	}
}
