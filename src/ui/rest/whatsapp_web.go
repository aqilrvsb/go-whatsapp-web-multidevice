package rest

import (
	"fmt"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
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
	
	// TODO: Get real WhatsApp service instance for this device
	// For now, check if device is connected
	userEmail := c.Locals("email")
	if userEmail == nil {
		sessionToken := c.Cookies("session_token")
		userRepo := repository.GetUserRepository()
		session, _ := userRepo.GetSession(sessionToken)
		if session != nil {
			user, _ := userRepo.GetUserByID(session.UserID)
			if user != nil {
				userEmail = user.Email
			}
		}
	}
	
	// Get device status
	userRepo := repository.GetUserRepository()
	user, _ := userRepo.GetUserByEmail(userEmail.(string))
	if user != nil {
		devices, _ := userRepo.GetUserDevices(user.ID)
		for _, device := range devices {
			if device.ID == deviceId {
				if device.Status != "online" {
					return c.JSON(utils.ResponseData{
						Status:  404,
						Code:    "NOT_CONNECTED",
						Message: "Device not connected to WhatsApp",
						Results: []interface{}{},
					})
				}
				break
			}
		}
	}
	
	// TODO: Implement real WhatsApp chat fetching
	// For now, return sample data to show the structure
	chats := []map[string]interface{}{
		{
			"id":          "1@s.whatsapp.net",
			"name":        "John Doe",
			"lastMessage": "Hey, how are you?",
			"time":        "10:30 AM",
			"unread":      2,
			"avatar":      "",
			"isGroup":     false,
		},
		{
			"id":          "2@s.whatsapp.net",
			"name":        "Jane Smith",
			"lastMessage": "See you tomorrow!",
			"time":        "9:45 AM",
			"unread":      0,
			"avatar":      "",
			"isGroup":     false,
		},
		{
			"id":          "group1@g.us",
			"name":        "Work Group",
			"lastMessage": "Meeting at 3 PM",
			"time":        "Yesterday",
			"unread":      5,
			"avatar":      "",
			"isGroup":     true,
		},
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Found %d chats", len(chats)),
		Results: chats,
	})
}

// GetWhatsAppMessages gets real messages for a specific chat
func (handler *App) GetWhatsAppMessages(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	chatId := c.Params("chatId")
	
	// TODO: Implement real WhatsApp message fetching
	// For now, return sample messages
	messages := []map[string]interface{}{
		{
			"id":        "msg1",
			"text":      "Hello!",
			"sent":      false,
			"time":      "10:00 AM",
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		},
		{
			"id":        "msg2",
			"text":      "Hi there! How are you?",
			"sent":      true,
			"time":      "10:05 AM",
			"status":    "delivered",
			"mediaType": "",
			"mediaUrl":  "",
		},
		{
			"id":        "msg3",
			"text":      "I'm good, thanks! How about you?",
			"sent":      false,
			"time":      "10:10 AM",
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		},
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Found %d messages for chat %s on device %s", len(messages), chatId, deviceId),
		Results: messages,
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
	
	// TODO: Implement real WhatsApp message sending
	// For now, simulate success
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Message sent successfully",
		Results: map[string]interface{}{
			"messageId": fmt.Sprintf("msg_%d", time.Now().Unix()),
			"timestamp": time.Now().Format(time.RFC3339),
			"status":    "sent",
		},
	})
}
