package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// CheckDeviceRegistration checks if a device's WhatsApp client is properly registered
func CheckDeviceRegistration(c *fiber.Ctx) error {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "device_id is required",
		})
	}
	
	// Check database status
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	
	dbStatus := map[string]interface{}{
		"found": err == nil,
		"error": "",
	}
	
	if err != nil {
		dbStatus["error"] = err.Error()
	} else {
		dbStatus["id"] = device.ID
		dbStatus["name"] = device.DeviceName
		dbStatus["phone"] = device.Phone
		dbStatus["jid"] = device.JID
		dbStatus["status"] = device.Status
	}
	
	// Check ClientManager registration
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	
	clientStatus := map[string]interface{}{
		"registered": err == nil,
		"error": "",
		"connected": false,
		"logged_in": false,
	}
	
	if err != nil {
		clientStatus["error"] = err.Error()
	} else if client != nil {
		clientStatus["connected"] = client.IsConnected()
		clientStatus["logged_in"] = client.IsLoggedIn()
		if client.Store != nil && client.Store.ID != nil {
			clientStatus["phone"] = client.Store.ID.User
			clientStatus["jid"] = client.Store.ID.String()
		}
	}
	
	// Check all registered clients
	allClients := cm.GetAllClients()
	registeredDevices := make([]string, 0, len(allClients))
	for id := range allClients {
		registeredDevices = append(registeredDevices, id)
	}
	
	return c.JSON(fiber.Map{
		"device_id": deviceID,
		"database": dbStatus,
		"client_manager": clientStatus,
		"all_registered_devices": registeredDevices,
		"total_registered": len(registeredDevices),
	})
}
