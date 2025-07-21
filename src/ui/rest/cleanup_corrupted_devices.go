package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
)

// CleanupCorruptedDevices removes devices with invalid IDs from ClientManager
func CleanupCorruptedDevices(c *fiber.Ctx) error {
	cm := whatsapp.GetClientManager()
	allClients := cm.GetAllClients()
	
	removedCount := 0
	for deviceID := range allClients {
		// Check for corrupted device IDs
		if len(deviceID) > 0 && (deviceID[0] == '/' || len(deviceID) != 36) {
			cm.RemoveClient(deviceID)
			removedCount++
			logrus.Infof("Removed corrupted device ID from ClientManager: %s", deviceID)
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cleanup completed",
		Results: map[string]interface{}{
			"removed_count": removedCount,
		},
	})
}
