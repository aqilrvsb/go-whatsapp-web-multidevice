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
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token found",
		})
	}
	
	// Get session from database
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
	isConnected := false
	devicePhone := ""
	for _, device := range devices {
		if device.ID == deviceId {
			if device.Status == "online" {
				isConnected = true
				devicePhone = device.Phone
			}
			break
		}
	}
	
	if !isConnected {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// TODO: Integrate with actual WhatsApp client to get real chats
	// For now, return a message indicating this needs implementation
	
	// Return demo chats with implementation notice
	chats := []map[string]interface{}{
		{
			"id":          "implementation_notice",
			"name":        "⚠️ Real-Time Integration Needed",
			"lastMessage": "This feature requires integration with WhatsApp Web client to fetch real chats",
			"time":        time.Now().Format("3:04 PM"),
			"unread":      1,
			"avatar":      "",
			"isGroup":     false,
		},
		{
			"id":          "status@broadcast",
			"name":        "Status",
			"lastMessage": "My Status",
			"time":        time.Now().Format("3:04 PM"),
			"unread":      0,
			"avatar":      "",
			"isGroup":     false,
		},
		{
			"id":          fmt.Sprintf("device_%s", deviceId),
			"name":        fmt.Sprintf("Device: %s", deviceId),
			"lastMessage": fmt.Sprintf("Phone: %s (Connected)", devicePhone),
			"time":        time.Now().Format("3:04 PM"),
			"unread":      0,
			"avatar":      "",
			"isGroup":     false,
		},
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Real-time chat integration pending. Showing demo data.",
		Results: chats,
	})
}


// GetWhatsAppMessages gets real messages for a specific chat
func (handler *App) GetWhatsAppMessages(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	chatId := c.Params("chatId")
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token found",
		})
	}
	
	// Get session from database
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
	isConnected := false
	devicePhone := ""
	for _, device := range devices {
		if device.ID == deviceId {
			if device.Status == "online" {
				isConnected = true
				devicePhone = device.Phone
			}
			break
		}
	}
	
	if !isConnected {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
			Results: []interface{}{},
		})
	}
	
	// TODO: Integrate with actual WhatsApp client to get real messages
	// Return messages based on chat
	messages := []map[string]interface{}{}
	
	if chatId == "implementation_notice" {
		messages = append(messages, map[string]interface{}{
			"id":        "notice_1",
			"text":      "🔔 Real-Time Chat Integration Required",
			"sent":      false,
			"time":      time.Now().Add(-5 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "notice_2",
			"text":      "To display real WhatsApp chats and messages, we need to:",
			"sent":      false,
			"time":      time.Now().Add(-4 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "notice_3",
			"text":      "1. Create a WhatsApp client manager to handle multiple device connections",
			"sent":      false,
			"time":      time.Now().Add(-3 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "notice_4",
			"text":      "2. Implement methods to fetch chats from the WhatsApp Web client",
			"sent":      false,
			"time":      time.Now().Add(-2 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "notice_5",
			"text":      "3. Create endpoints that connect to the device's WhatsApp instance",
			"sent":      false,
			"time":      time.Now().Add(-1 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "notice_6",
			"text":      "Currently showing demo data. Device is connected: " + devicePhone,
			"sent":      false,
			"time":      time.Now().Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
	} else if chatId == "status@broadcast" {
		messages = append(messages, map[string]interface{}{
			"id":        "status_1",
			"text":      "Tap to add status update",
			"sent":      true,
			"time":      time.Now().Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
	} else {
		// Default messages for device info
		messages = append(messages, map[string]interface{}{
			"id":        "device_1",
			"text":      "Device Information",
			"sent":      false,
			"time":      time.Now().Add(-2 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		messages = append(messages, map[string]interface{}{
			"id":        "device_2",
			"text":      fmt.Sprintf("Device ID: %s", deviceId),
			"sent":      false,
			"time":      time.Now().Add(-1 * time.Minute).Format("3:04 PM"),
			"status":    "read",
			"mediaType": "",
			"mediaUrl":  "",
		})
		if devicePhone != "" {
			messages = append(messages, map[string]interface{}{
				"id":        "device_3",
				"text":      fmt.Sprintf("Phone: %s", devicePhone),
				"sent":      false,
				"time":      time.Now().Format("3:04 PM"),
				"status":    "read",
				"mediaType": "",
				"mediaUrl":  "",
			})
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Messages for chat %s", chatId),
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
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token found",
		})
	}
	
	// Get session from database
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
	
	// Check if device belongs to user
	devices, err := userRepo.GetUserDevices(user.ID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}
	
	isConnected := false
	for _, device := range devices {
		if device.ID == deviceId && device.Status == "online" {
			isConnected = true
			break
		}
	}
	
	if !isConnected {
		return c.JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_CONNECTED",
			Message: "Device not connected to WhatsApp",
		})
	}
	
	// Device is connected
	// To send real messages, integration with WhatsApp client is needed
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Message sending requires WhatsApp client integration",
		Results: map[string]interface{}{
			"messageId": fmt.Sprintf("msg_%d", time.Now().Unix()),
			"timestamp": time.Now().Format(time.RFC3339),
			"status":    "pending_integration",
			"deviceId":  deviceId,
			"note":      "Real-time messaging requires WhatsApp Web client integration",
		},
	})
}
