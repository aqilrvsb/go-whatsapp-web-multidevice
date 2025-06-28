package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// Fixed DeleteDevice - properly cleans up device and WhatsApp connection
func (handler *App) DeleteDeviceFixed(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
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
	device, err := userRepo.GetDeviceByID(deviceId)
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
	
	logrus.Infof("Deleting device %s (%s) for user %s", device.ID, device.DeviceName, session.UserID)
	
	// Step 1: Disconnect WhatsApp client if connected
	cm := whatsapp.GetClientManager()
	if client := cm.GetClient(deviceId); client != nil {
		logrus.Info("Disconnecting WhatsApp client...")
		
		// Logout from WhatsApp
		if client.IsConnected() {
			client.Logout()
		}
		client.Disconnect()
		
		// Remove from client manager
		cm.RemoveClient(deviceId)
	}
	
	// Step 2: Clean up WhatsApp store data
	// This is done automatically when the device is deleted from database
	
	// Step 3: Clear associated data
	whatsappRepo := repository.GetWhatsAppRepository()
	if whatsappRepo != nil {
		// Clear messages
		err = whatsappRepo.ClearDeviceMessages(deviceId)
		if err != nil {
			logrus.Errorf("Failed to clear messages: %v", err)
		}
		
		// Clear chats
		err = whatsappRepo.ClearDeviceChats(deviceId)
		if err != nil {
			logrus.Errorf("Failed to clear chats: %v", err)
		}
	}
	
	// Step 4: Delete device from database
	err = userRepo.DeleteDevice(deviceId)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to delete device",
		})
	}
	
	logrus.Infof("Successfully deleted device %s", device.DeviceName)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device deleted successfully",
		Results: fiber.Map{
			"device_id": deviceId,
			"device_name": device.DeviceName,
		},
	})
}

// Fixed LogoutDevice - properly disconnects device from WhatsApp
func (handler *App) LogoutDeviceFixed(c *fiber.Ctx) error {
	deviceId := c.Query("deviceId")
	if deviceId == "" {
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
	device, err := userRepo.GetDeviceByID(deviceId)
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
	
	logrus.Infof("Logging out device %s (%s)", device.ID, device.DeviceName)
	
	// Disconnect WhatsApp client
	cm := whatsapp.GetClientManager()
	if client := cm.GetClient(deviceId); client != nil {
		// Logout from WhatsApp
		if client.IsConnected() {
			err = client.Logout()
			if err != nil {
				logrus.Errorf("Error logging out: %v", err)
			}
		}
		
		// Disconnect client
		client.Disconnect()
		
		// Remove from client manager
		cm.RemoveClient(deviceId)
		
		logrus.Info("WhatsApp client disconnected and removed from manager")
	}
	
	// Update device status in database
	err = userRepo.UpdateDeviceStatus(deviceId, "disconnected", "", "")
	if err != nil {
		logrus.Errorf("Error updating device status: %v", err)
	}
	
	// Clean up any session data
	whatsapp.ClearConnectionSession(session.UserID)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device logged out successfully",
		Results: map[string]interface{}{
			"device_id": deviceId,
			"device_name": device.DeviceName,
			"status": "disconnected",
		},
	})
}