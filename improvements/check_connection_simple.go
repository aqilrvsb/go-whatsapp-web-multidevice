package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
)

// CheckConnection is a simple endpoint to verify API is working
func CheckConnection(c *fiber.Ctx) error {
	// Get user session
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}

	// For now, just return success if session exists
	// The actual device checking happens in GetDevices endpoint
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Connection check endpoint is working",
		Results: map[string]interface{}{
			"api_status": "online",
			"timestamp":  time.Now().Unix(),
		},
	})
}