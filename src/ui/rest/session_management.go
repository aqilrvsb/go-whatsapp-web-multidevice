package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// ClearDeviceSession clears all WhatsApp session data for a specific device
func (handler *App) ClearDeviceSession(c *fiber.Ctx) error {
	deviceID := c.Query("deviceId")
	
	if deviceID == "" {
		return c.JSON(fiber.Map{
			"code":    "BAD_REQUEST",
			"message": "Device ID is required",
		})
	}
	
	logrus.Infof("Clearing WhatsApp session for device: %s", deviceID)
	
	// Clear the session
	err := whatsapp.ClearWhatsAppSessionData(deviceID)
	if err != nil {
		logrus.Errorf("Failed to clear session: %v", err)
		return c.JSON(fiber.Map{
			"code":    "ERROR",
			"message": "Failed to clear session",
			"error":   err.Error(),
		})
	}
	
	// Update device status
	userRepo := repository.GetUserRepository()
	err = userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	return c.JSON(fiber.Map{
		"code":    "SUCCESS",
		"message": "Session cleared successfully",
	})
}

// ClearAllSessions clears all WhatsApp sessions (admin function)
func (handler *App) ClearAllSessions(c *fiber.Ctx) error {
	logrus.Warn("Clearing ALL WhatsApp sessions")
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Get session info before clearing
	info, _ := whatsapp.GetWhatsAppSessionInfo(db)
	
	// Clear all sessions
	err := whatsapp.ClearAllWhatsAppSessions(db)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    "ERROR",
			"message": "Failed to clear sessions",
			"error":   err.Error(),
		})
	}
	
	// Update all devices to offline
	_, err = db.Exec(`UPDATE user_devices SET status = 'offline', phone = NULL, jid = NULL`)
	if err != nil {
		logrus.Errorf("Failed to update device statuses: %v", err)
	}
	
	return c.JSON(fiber.Map{
		"code":    "SUCCESS",
		"message": "All sessions cleared",
		"info":    info,
	})
}
