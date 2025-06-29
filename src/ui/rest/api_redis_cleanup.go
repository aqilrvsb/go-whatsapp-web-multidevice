package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// InitRedisCleanupAPI initializes Redis cleanup endpoints
func InitRedisCleanupAPI(app *fiber.App) {
	app.Post("/api/redis/cleanup-device/:deviceId", CleanupDeviceFromRedis)
	app.Post("/api/redis/cleanup-old-devices", CleanupAllOldDevices)
}

// CleanupDeviceFromRedis removes all Redis data for a specific device
func CleanupDeviceFromRedis(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Get broadcast manager
	bm := broadcast.GetBroadcastManager()
	
	// If it's an UltraScaleRedisManager, use cleanup method
	if ultraBM, ok := bm.(*broadcast.UltraScaleRedisManager); ok {
		ultraBM.CleanupNonExistentDevice(deviceID)
		logrus.Infof("Cleaned up device %s from Redis", deviceID)
		
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Device cleaned up from Redis",
			"deviceId": deviceID,
		})
	}
	
	return c.Status(400).JSON(fiber.Map{
		"error": "Redis cleanup not available",
	})
}

// CleanupAllOldDevices cleans up all non-existent devices from Redis
func CleanupAllOldDevices(c *fiber.Ctx) error {
	// List of known old devices to clean up
	oldDevices := []string{
		"3472b8c5-974b-4deb-bab9-792cc5a09c57", // Your old device
		// Add more as needed
	}
	
	bm := broadcast.GetBroadcastManager()
	if ultraBM, ok := bm.(*broadcast.UltraScaleRedisManager); ok {
		for _, deviceID := range oldDevices {
			ultraBM.CleanupNonExistentDevice(deviceID)
		}
		
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Old devices cleaned up from Redis",
			"devices": oldDevices,
		})
	}
	
	return c.Status(400).JSON(fiber.Map{
		"error": "Redis cleanup not available",
	})
}
