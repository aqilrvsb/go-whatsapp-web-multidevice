package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/proxy"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Proxy struct{}

// GetProxyStats returns proxy statistics
func (controller *Proxy) GetProxyStats(c *fiber.Ctx) error {
	pm := proxy.GetProxyManager()
	stats := pm.GetProxyStats()
	
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "Proxy statistics",
		Results: stats,
	})
}

// GetDeviceProxy returns proxy assigned to a device
func (controller *Proxy) GetDeviceProxy(c *fiber.Ctx) error {
	deviceID := c.Params("device_id")
	
	pm := proxy.GetProxyManager()
	assignedProxy := pm.GetProxyForDevice(deviceID)
	
	if assignedProxy == nil {
		return c.JSON(utils.ResponseData{
			Code:    "NOT_FOUND",
			Message: "No proxy assigned to this device",
			Results: nil,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "Device proxy info",
		Results: assignedProxy,
	})
}

// RefreshProxies manually triggers proxy refresh
func (controller *Proxy) RefreshProxies(c *fiber.Ctx) error {
	pm := proxy.GetProxyManager()
	
	// Trigger proxy refresh in background
	go pm.FetchMalaysianProxies()
	
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS", 
		Message: "Proxy refresh initiated",
		Results: nil,
	})
}

// AssignProxy manually assigns a proxy to device
func (controller *Proxy) AssignProxy(c *fiber.Ctx) error {
	type request struct {
		DeviceID string `json:"device_id"`
	}
	
	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}
	
	pm := proxy.GetProxyManager()
	assignedProxy, err := pm.AssignProxyToDevice(req.DeviceID)
	if err != nil {
		return c.JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "Proxy assigned successfully",
		Results: assignedProxy,
	})
}