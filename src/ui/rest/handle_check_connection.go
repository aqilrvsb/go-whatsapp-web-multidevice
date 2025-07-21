package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
)

// HandleCheckConnection checks the connection status of all devices
func HandleCheckConnection(c *fiber.Ctx) error {
	// Get session from cookie
	cookie := c.Cookies("session_token")
	if cookie == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}

	// Validate session
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(cookie)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}

	// Get user devices
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}

	// Check connection status for each device
	cm := whatsapp.GetClientManager()
	deviceStatuses := make(map[string]bool)
	
	for _, device := range devices {
		// Skip platform devices
		if device.Platform != "" {
			deviceStatuses[device.ID] = true // Always consider platform devices as connected
			continue
		}
		
		// Check if client exists and is connected
		client, err := cm.GetClient(device.ID)
		if err == nil && client != nil && client.IsConnected() {
			deviceStatuses[device.ID] = true
		} else {
			deviceStatuses[device.ID] = false
		}
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Connection status checked",
		Results: deviceStatuses,
	})
}
