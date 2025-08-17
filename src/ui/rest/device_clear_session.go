package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
)

// ClearDeviceSession clears WhatsApp session for a specific device
func (handler *App) ClearDeviceSession(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	if deviceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Device ID is required",
		})
	}
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	// Get device details
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Verify device belongs to user
	if device.UserID != session.UserID {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "Device does not belong to this user",
		})
	}
	
	logrus.Infof("Clearing WhatsApp session for device %s (%s)", device.ID, device.DeviceName)
	
	// Use enhanced logout for complete cleanup
	err = whatsapp.EnhancedLogout(deviceID)
	if err != nil {
		logrus.Errorf("Error during enhanced logout: %v", err)
	}
	
	// Verify logout was successful
	if !whatsapp.VerifyDeviceLoggedOut(deviceID) {
		logrus.Warn("Device may not be fully logged out - forcing additional cleanup")
		
		// Force remove from all managers
		cm := whatsapp.GetClientManager()
		cm.RemoveClient(deviceID)
		
		dcm := whatsapp.GetDeviceConnectionManager()
		dcm.RemoveConnection(deviceID)
	}
	
	// Clear any connection session
	whatsapp.ClearConnectionSession(session.UserID)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device session cleared successfully",
		Results: map[string]interface{}{
			"device_id": deviceID,
			"device_name": device.DeviceName,
			"status": "offline",
		},
	})
}