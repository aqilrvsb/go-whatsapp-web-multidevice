package rest

import (
	"fmt"
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
	
	// Disconnect WhatsApp client
	cm := whatsapp.GetClientManager()
	var jid string
	if client, err := cm.GetClient(deviceID); err == nil && client != nil {
		// Get JID before disconnecting
		if client.Store != nil && client.Store.ID != nil {
			jid = client.Store.ID.String()
		}
		
		// Logout from WhatsApp
		if client.IsConnected() {
			err = client.Logout(c.UserContext())
			if err != nil {
				logrus.Errorf("Error logging out: %v", err)
			}
		}
		
		// Disconnect client
		client.Disconnect()
		
		// Remove from client manager
		cm.RemoveClient(deviceID)
		
		logrus.Info("WhatsApp client disconnected and removed from manager")
	}
	
	// Clear WhatsApp session tables for this device
	if jid != "" {
		db := userRepo.DB()
		tables := []string{
			"whatsmeow_device",
			"whatsmeow_identity_keys",
			"whatsmeow_pre_keys",
			"whatsmeow_sessions",
			"whatsmeow_sender_keys",
			"whatsmeow_app_state_sync_keys",
			"whatsmeow_app_state_version",
			"whatsmeow_message_secrets",
			"whatsmeow_privacy_tokens",
			"whatsmeow_chat_settings",
			"whatsmeow_contacts",
			"whatsmeow_app_state_mutation_macs",
		}
		
		for _, table := range tables {
			query := fmt.Sprintf("DELETE FROM %s WHERE jid = $1", table)
			result, err := db.Exec(query, jid)
			if err != nil {
				logrus.Warnf("Failed to clear table %s for JID %s: %v", table, jid, err)
			} else {
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					logrus.Infof("Cleared %d rows from table %s for JID %s", rowsAffected, table, jid)
				}
			}
		}
	}
	
	// Update device status to offline (not disconnected)
	err = userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
	if err != nil {
		logrus.Errorf("Error updating device status: %v", err)
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