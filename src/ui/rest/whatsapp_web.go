package rest

import (
	"fmt"
	"strings"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
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
	
	// Check if device belongs to user
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
	
	// Get query parameter for fetching all contacts
	fetchAll := c.Query("all") == "true"
	
	// Get chats (from WhatsApp if online, from database if offline)
	var chats []repository.WhatsAppChat
	var err error
	
	if fetchAll && isOnline {
		// Try to get ALL personal chats including contacts without messages
		chats, err = whatsapp.GetAllPersonalChats(deviceId)
		if err != nil {
			// Fallback to regular method
			chats, err = whatsapp.GetChatsForDevice(deviceId)
		}
	} else {
		// Regular method - get chats with recent activity
		chats, err = whatsapp.GetChatsForDevice(deviceId)
	}
	if err != nil && isOnline {
		// If online but failed to get chats, return error
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to get chats: %v", err),
		})
	}
	
	// Convert to response format
	var responseChats []map[string]interface{}
	for _, chat := range chats {
		// Format time
		timeStr := ""
		if !chat.LastMessageTime.IsZero() {
			now := time.Now()
			if chat.LastMessageTime.Day() == now.Day() {
				timeStr = chat.LastMessageTime.Format("3:04 PM")
			} else if chat.LastMessageTime.Year() == now.Year() {
				timeStr = chat.LastMessageTime.Format("Jan 2")
			} else {
				timeStr = chat.LastMessageTime.Format("01/02/06")
			}
		}
		
		// Determine avatar based on type
		avatar := ""
		if chat.IsGroup {
			avatar = "👥"
		} else {
			avatar = "👤"
		}
		
		responseChats = append(responseChats, map[string]interface{}{
			"id":          chat.ChatJID,
			"name":        chat.ChatName,
			"lastMessage": chat.LastMessageText,
			"time":        timeStr,
			"unread":      chat.UnreadCount,
			"avatar":      avatar,
			"isGroup":     chat.IsGroup,
			"isMuted":     chat.IsMuted,
		})
	}
	
	// If no chats found, return empty array with appropriate message
	if len(responseChats) == 0 {
		message := "No chats found"
		if !isOnline {
			message = "Device offline. No saved chats available."
		}
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: message,
			Results: []interface{}{},
		})
	}
	
	// Return chats with status info
	statusMessage := fmt.Sprintf("Found %d chats", len(responseChats))
	if !isOnline {
		statusMessage = fmt.Sprintf("Device offline. Showing %d saved chats", len(responseChats))
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: statusMessage,
		Results: responseChats,
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
	
	// Check if device belongs to user
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
	
	// Get messages (from WhatsApp if online, from database if offline)
	messages, err := whatsapp.GetMessagesForChat(deviceId, chatId, 50) // Get last 50 messages
	if err != nil && isOnline {
		// If online but failed to get messages, return error
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to get messages: %v", err),
		})
	}
	
	// Convert to response format
	var responseMessages []map[string]interface{}
	for _, msg := range messages {
		// Format time
		timeStr := msg.Timestamp.Format("3:04 PM")
		
		// Determine if message contains current user's phone
		myPhone := ""
		for _, device := range devices {
			if device.ID == deviceId && device.Phone != "" {
				myPhone = device.Phone
				break
			}
		}
		
		// Check if message is sent by me
		isSent := msg.IsSent
		if myPhone != "" && strings.Contains(msg.SenderJID, myPhone) {
			isSent = true
		}
		
		responseMessages = append(responseMessages, map[string]interface{}{
			"id":        msg.MessageID,
			"text":      msg.MessageText,
			"sent":      isSent,
			"time":      timeStr,
			"status":    "read", // Simplified status
			"mediaType": msg.MessageType,
			"mediaUrl":  msg.MediaURL,
			"sender":    msg.SenderName,
		})
	}
	
	// If no messages found, return empty array
	if len(responseMessages) == 0 {
		message := "No messages in this chat"
		if !isOnline {
			message = "Device offline. No saved messages available."
		}
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: message,
			Results: []interface{}{},
		})
	}
	
	// Return messages with status info
	statusMessage := fmt.Sprintf("Found %d messages", len(responseMessages))
	if !isOnline {
		statusMessage = fmt.Sprintf("Device offline. Showing %d saved messages", len(responseMessages))
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: statusMessage,
		Results: responseMessages,
	})
}

// SendWhatsAppMessage sends a real message via WhatsApp (Read-only mode, not implemented)
func (handler *App) SendWhatsAppMessage(c *fiber.Ctx) error {
	return c.Status(403).JSON(utils.ResponseData{
		Status:  403,
		Code:    "FORBIDDEN",
		Message: "This is a read-only view. Message sending is disabled.",
	})
}
