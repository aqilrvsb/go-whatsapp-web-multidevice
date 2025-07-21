package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// validateDeviceOwnership checks if the authenticated user owns the specified device
func validateDeviceOwnership(c *fiber.Ctx, userID, deviceID string) error {
	// Convert userID string to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_USER_ID",
			Message: "Invalid user ID format",
		})
	}
	
	// Convert deviceID string to UUID
	deviceUUID, err := uuid.Parse(deviceID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_DEVICE_ID",
			Message: "Invalid device ID format",
		})
	}
	
	// Check device ownership
	deviceRepo := repository.GetDeviceRepository()
	device, err := deviceRepo.GetDeviceByID(deviceUUID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
			Status:  404,
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Verify the device belongs to the authenticated user
	if device.UserID != userUUID {
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
