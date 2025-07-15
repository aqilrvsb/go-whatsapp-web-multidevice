package rest

import (
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// WebhookLeadRequest represents the incoming webhook payload
type WebhookLeadRequest struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	TargetStatus string `json:"target_status"`
	DeviceID     string `json:"device_id"`
	UserID       string `json:"user_id"`
	Niche        string `json:"niche"`
	Trigger      string `json:"trigger"`
	DeviceName   string `json:"device_name"`  // New field
	Platform     string `json:"platform"`     // New field
}

// InitWebhookLead initializes the webhook endpoint for creating leads
func InitWebhookLead(app *fiber.App) {
	// Public webhook endpoint (no auth middleware)
	app.Post("/webhook/lead/create", CreateLeadWebhook)
}

// CreateLeadWebhook handles the webhook request to create a lead
func CreateLeadWebhook(c *fiber.Ctx) error {
	// Parse request body
	var request WebhookLeadRequest
	if err := c.BodyParser(&request); err != nil {
		logrus.Error("Webhook Lead: Failed to parse request body - ", err)
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	// Log the incoming request for debugging
	logrus.Info("Webhook Lead: Received request - ", request)

	// Basic validation - only check required fields
	if request.Name == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "Name is required",
		})
	}
	
	if request.Phone == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "Phone is required",
		})
	}
	
	if request.UserID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "User ID is required",
		})
	}
	
	if request.DeviceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "VALIDATION_ERROR",
			Message: "Device ID is required",
		})
	}

	// Check if device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(request.DeviceID)
	
	// If device doesn't exist, create it
	if err != nil || device == nil {
		logrus.Info("Webhook Lead: Device not found, creating new device - ", request.DeviceID)
		
		// Create new device
		newDevice := &models.UserDevice{
			ID:               request.DeviceID, // Use the provided device_id as the ID
			UserID:           request.UserID,
			DeviceName:       request.DeviceName,
			Phone:            "", // null/empty
			JID:              request.DeviceID, // Also save device_id in JID column
			Status:           "online",
			LastSeen:         time.Now(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			MinDelaySeconds:  5,
			MaxDelaySeconds:  15,
			Platform:         request.Platform,
		}
		
		// If device_name not provided, use a default
		if newDevice.DeviceName == "" {
			newDevice.DeviceName = "Device-" + time.Now().Format("20060102150405")
		}
		
		// Create device in database
		err = userRepo.CreateDevice(newDevice)
		if err != nil {
			logrus.Error("Webhook Lead: Failed to create device - ", err)
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "DEVICE_CREATE_FAILED",
				Message: "Failed to create device",
				Results: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		
		logrus.Info("Webhook Lead: Device created successfully - ", newDevice.ID)
	}

	// Create lead object with direct field mapping
	lead := &models.Lead{
		ID:           uuid.New().String(),
		Name:         request.Name,
		Phone:        request.Phone,
		Niche:        request.Niche,
		Trigger:      request.Trigger,
		TargetStatus: request.TargetStatus,
		DeviceID:     request.DeviceID,
		UserID:       request.UserID,
		Platform:     request.Platform, // Add platform field
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create lead in database
	leadRepo := repository.GetLeadRepository()
	err = leadRepo.CreateLead(lead)
	if err != nil {
		logrus.Error("Webhook Lead: Failed to create lead - ", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CREATE_FAILED",
			Message: "Failed to create lead",
			Results: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	logrus.Info("Webhook Lead: Successfully created lead - ", lead.ID)

	// Return success response with all the data that was saved
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Lead created successfully",
		Results: map[string]interface{}{
			"lead_id":       lead.ID,
			"name":          lead.Name,
			"phone":         lead.Phone,
			"niche":         lead.Niche,
			"trigger":       lead.Trigger,
			"target_status": lead.TargetStatus,
			"device_id":     lead.DeviceID,
			"user_id":       lead.UserID,
			"platform":      lead.Platform,
			"device_created": device == nil, // Indicates if a new device was created
			"created_at":    lead.CreatedAt,
		},
	})
}
