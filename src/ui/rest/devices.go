package rest

import (
	"fmt"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// GetConnectedDevices returns real connected devices
func (handler *App) GetConnectedDevices(c *fiber.Ctx) error {
	// Get user from context (set by auth middleware)
	userID := c.Locals("userID")
	if userID == nil {
		// Fallback: try to get from session cookie
		token := c.Cookies("session_token")
		if token == "" {
			return c.Status(401).JSON(utils.ResponseData{
				Status:  401,
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			})
		}
		
		userRepo := repository.GetUserRepository()
		session, err := userRepo.GetSession(token)
		if err != nil {
			return c.Status(401).JSON(utils.ResponseData{
				Status:  401,
				Code:    "UNAUTHORIZED",
				Message: "Invalid session",
			})
		}
		userID = session.UserID
	}
	
	// Get user devices from database
	userRepo := repository.GetUserRepository()
	devices, err := userRepo.GetUserDevices(userID.(string))
	if err != nil {
		// If no devices found, return empty array instead of error
		if err.Error() == "no devices found" {
			return c.JSON(utils.ResponseData{
				Status:  200,
				Code:    "SUCCESS",
				Message: "No devices found",
				Results: []fiber.Map{},
			})
		}
		
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}
	
	// Convert to response format
	deviceList := make([]fiber.Map, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, fiber.Map{
			"id":       device.ID,
			"name":     device.DeviceName,
			"phone":    device.Phone,
			"status":   device.Status,
			"lastSeen": device.LastSeen.Format("2006-01-02 15:04:05"),
			"jid":      device.JID,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Devices retrieved",
		Results: deviceList,
	})
}

// CreateDevice creates a new device for the user
func (handler *App) CreateDevice(c *fiber.Ctx) error {
	// Get user from context
	userID := c.Locals("userID")
	if userID == nil {
		// Fallback: try to get from session cookie
		token := c.Cookies("session_token")
		if token == "" {
			return c.Status(401).JSON(utils.ResponseData{
				Status:  401,
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			})
		}
		
		userRepo := repository.GetUserRepository()
		session, err := userRepo.GetSession(token)
		if err != nil {
			return c.Status(401).JSON(utils.ResponseData{
				Status:  401,
				Code:    "UNAUTHORIZED",
				Message: "Invalid session",
			})
		}
		userID = session.UserID
	}
	
	// Parse request body
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	if req.Name == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Device name is required",
		})
	}
	
	// Create device in database
	userRepo := repository.GetUserRepository()
	
	device, err := userRepo.AddUserDevice(userID.(string), req.Name)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to create device",
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device created successfully",
		Results: fiber.Map{
			"success": true,
			"device": fiber.Map{
				"id":       device.ID,
				"name":     device.DeviceName,
				"phone":    device.Phone,
				"status":   device.Status,
				"lastSeen": device.LastSeen.Format("2006-01-02 15:04:05"),
			},
		},
	})
}
