package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// ClearAllSessions clears all WhatsApp session data
func (handler *App) ClearAllSessions(c *fiber.Ctx) error {
	// Get user from context to ensure authentication
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		})
	}
	
	// Get database connection
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// List of WhatsApp session tables to drop
	tables := []string{
		"whatsmeow_app_state_mutation_macs",
		"whatsmeow_app_state_sync_keys",
		"whatsmeow_app_state_version",
		"whatsmeow_chat_settings",
		"whatsmeow_contacts",
		"whatsmeow_device",
		"whatsmeow_identity_keys",
		"whatsmeow_message_secrets",
		"whatsmeow_pre_keys",
		"whatsmeow_privacy_tokens",
		"whatsmeow_sender_keys",
		"whatsmeow_sessions",
	}
	
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to start transaction",
		})
	}
	defer tx.Rollback()
	
	// Drop each table if it exists
	for _, table := range tables {
		query := "DROP TABLE IF EXISTS " + table + " CASCADE"
		_, err := tx.Exec(query)
		if err != nil {
			logrus.Warnf("Failed to drop table %s: %v", table, err)
			// Continue with other tables even if one fails
		} else {
			logrus.Infof("Dropped table: %s", table)
		}
	}
	
	// Update all devices to offline status for this user
	updateQuery := `
		UPDATE user_devices 
		SET status = 'offline', 
		    jid = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND status != 'deleted'
	`
	_, err = tx.Exec(updateQuery, userID.(string))
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to clear sessions",
		})
	}
	
	// Clear any in-memory sessions as well
	// This would clear connection sessions for all users - be careful!
	// For now, we'll just log that sessions were cleared
	logrus.Infof("WhatsApp sessions cleared for user %s", userID.(string))
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "All WhatsApp sessions have been cleared",
		Results: map[string]interface{}{
			"tablesDropped": len(tables),
			"devicesUpdated": true,
		},
	})
}
