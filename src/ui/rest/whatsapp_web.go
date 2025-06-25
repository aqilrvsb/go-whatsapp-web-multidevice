package rest

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
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
	
	// For now, return mock data until WhatsApp integration is complete
	// TODO: Get actual chats from WhatsApp connection for this device
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
		Message: fmt.Sprintf("Chats for device %s retrieved", deviceId),
		Results: chats,
	})
}

// GetWhatsAppMessages gets messages for a specific chat
func (handler *App) GetWhatsAppMessages(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	chatId := c.Params("chatId")
	
	// For now, return mock data until WhatsApp integration is complete
	// TODO: Get actual messages from WhatsApp connection for this device
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
		Message: fmt.Sprintf("Messages for device %s, chat %s retrieved", deviceId, chatId),
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
	
	// For now, return success until WhatsApp integration is complete
	// TODO: Send actual message via WhatsApp connection for this device
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Message sent to chat %s on device %s", request.ChatID, deviceId),
		Results: map[string]interface{}{
			"messageId": "msg_123",
			"timestamp": "2025-06-25T10:00:00Z",
		},
	})
}
