package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
)

// validateDeviceOwnership checks if the authenticated user owns the specified device
func validateDeviceOwnership(c *fiber.Ctx, userID, deviceID string) error {
	// Check device ownership
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
			Status:  404,
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Verify the device belongs to the authenticated user
	if device.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "You don't have permission to use this device",
		})
	}
	
	// Check if device is online
	if device.Status != "online" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "DEVICE_OFFLINE",
			Message: "Device is not connected. Please connect the device first.",
		})
	}
	
	return nil
}
