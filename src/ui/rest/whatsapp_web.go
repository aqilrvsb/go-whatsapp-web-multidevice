package rest

import (
	"fmt"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	whatsapp2 "github.com/aldinokemal/go-whatsapp-web-multidevice/services/whatsapp"
)

// WhatsAppWebView renders the WhatsApp Web interface for a device
func (handler *App) WhatsAppWebView(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Check if user has valid session cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Redirect("/login")
	}
	
	// Verify session is valid
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil || session == nil {
		return c.Redirect("/login")
	}
	
	// Session is valid, render WhatsApp Web
	return c.Render("views/whatsapp_web", fiber.Map{
		"DeviceID": deviceId,
	})
}

// GetWhatsAppChats gets real chats for a specific device
func (handler *App) GetWhatsAppChats(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get the WhatsApp service for this device
	service := whatsapp2.GetWhatsAppService(deviceId)
	if service == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// Get real chats from WhatsApp
	chats, err := service.GetChats(c.UserContext())
	if err != nil {
		return c.JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to get chats: %v", err),
			Results: []interface{}{},
		})
	}
	
	// Format chats for frontend
	formattedChats := []map[string]interface{}{}
	for _, chat := range chats {
		formattedChats = append(formattedChats, map[string]interface{}{
			"id":          chat.ID,
			"name":        chat.Name,
			"lastMessage": chat.LastMessage,
			"time":        chat.LastMessageTime,
			"unread":      chat.UnreadCount,
			"avatar":      chat.Avatar,
			"isGroup":     chat.IsGroup,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Found %d chats", len(formattedChats)),
		Results: formattedChats,
	})
}

// GetWhatsAppMessages gets real messages for a specific chat
func (handler *App) GetWhatsAppMessages(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	chatId := c.Params("chatId")
	
	// Get the WhatsApp service for this device
	service := whatsapp2.GetWhatsAppService(deviceId)
	if service == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// Get real messages from WhatsApp
	messages, err := service.GetMessages(c.UserContext(), chatId)
	if err != nil {
		return c.JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to get messages: %v", err),
			Results: []interface{}{},
		})
	}
	
	// Format messages for frontend
	formattedMessages := []map[string]interface{}{}
	for _, msg := range messages {
		formattedMessages = append(formattedMessages, map[string]interface{}{
			"id":        msg.ID,
			"text":      msg.Text,
			"sent":      msg.FromMe,
			"time":      msg.Timestamp.Format("3:04 PM"),
			"status":    msg.Status,
			"mediaType": msg.MediaType,
			"mediaUrl":  msg.MediaURL,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Found %d messages", len(formattedMessages)),
		Results: formattedMessages,
	})
}

// SendWhatsAppMessage sends a real message via WhatsApp
func (handler *App) SendWhatsAppMessage(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	var request struct {
		ChatID  string `json:"chatId"`
		Message string `json:"message"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request",
		})
	}
	
	// Get the WhatsApp service for this device
	service := whatsapp2.GetWhatsAppService(deviceId)
	if service == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
		})
	}
	
	// Send real message via WhatsApp
	messageId, err := service.SendTextMessage(c.UserContext(), request.ChatID, request.Message)
	if err != nil {
		return c.JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to send message: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Message sent successfully",
		Results: map[string]interface{}{
			"messageId": messageId,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}
