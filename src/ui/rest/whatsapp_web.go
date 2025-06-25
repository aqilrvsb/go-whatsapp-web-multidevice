package rest

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	whatsapp2 "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/whatsapp"
)

// WhatsAppWebView renders the WhatsApp Web interface for a device
func (handler *App) WhatsAppWebView(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Check if user is authenticated
	userEmail := c.Locals("email")
	if userEmail == nil {
		return c.Redirect("/login")
	}
	
	// Render the WhatsApp Web view
	return c.Render("views/whatsapp_web", fiber.Map{
		"DeviceID": deviceId,
	})
}

// GetWhatsAppChats gets chats for a specific device
func (handler *App) GetWhatsAppChats(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get WhatsApp instance for this device
	whatsappService := whatsapp2.WhatsAppServiceMap[deviceId]
	if whatsappService == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// Get chats from WhatsApp
	// This is a placeholder - implement actual chat retrieval
	chats := []map[string]interface{}{
		{
			"id":          "1",
			"name":        "Contact 1",
			"lastMessage": "Hello",
			"time":        "10:30 AM",
			"unread":      0,
			"avatar":      "",
		},
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Chats retrieved",
		Results: chats,
	})
}

// GetWhatsAppMessages gets messages for a specific chat
func (handler *App) GetWhatsAppMessages(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	chatId := c.Params("chatId")
	
	// Get WhatsApp instance for this device
	whatsappService := whatsapp2.WhatsAppServiceMap[deviceId]
	if whatsappService == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// Get messages from WhatsApp
	// This is a placeholder - implement actual message retrieval
	messages := []map[string]interface{}{
		{
			"id":   "1",
			"text": "Hello!",
			"sent": false,
			"time": "10:00 AM",
		},
		{
			"id":   "2",
			"text": "Hi there!",
			"sent": true,
			"time": "10:05 AM",
		},
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Messages for chat %s retrieved", chatId),
		Results: messages,
	})
}

// SendWhatsAppMessage sends a message via WhatsApp Web
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
	
	// Get WhatsApp instance for this device
	whatsappService := whatsapp2.WhatsAppServiceMap[deviceId]
	if whatsappService == nil {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not connected to WhatsApp",
		})
	}
	
	// Send message via WhatsApp
	// This is a placeholder - implement actual message sending
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Message sent",
		Results: map[string]interface{}{
			"messageId": "msg_123",
			"timestamp": "2025-06-25T10:00:00Z",
		},
	})
}
