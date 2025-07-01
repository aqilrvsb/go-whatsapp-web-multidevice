package rest

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// TransferAILeadsToDevice transfers successful AI campaign leads to device's regular leads table
func (handler *App) TransferAILeadsToDevice(c *fiber.Ctx) error {
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
	
	// Parse request
	var req struct {
		DeviceID   string `json:"device_id"`
		CampaignID string `json:"campaign_id"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	// Verify device belongs to user
	device, err := userRepo.GetDeviceByID(req.DeviceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	if device.UserID != session.UserID {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "Device does not belong to this user",
		})
	}
	
	// Check if device is connected
	if device.Status != "connected" && device.Status != "active" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "DEVICE_NOT_CONNECTED",
			Message: "Device must be connected to transfer leads",
		})
	}
	
	// Get the AI campaign
	var campaign struct {
		ID           string
		Name         string
		CampaignType string
	}
	
	err = userRepo.DB().QueryRow(`
		SELECT id, name, campaign_type 
		FROM campaigns 
		WHERE id = $1 AND device_id = $2
	`, req.CampaignID, req.DeviceID).Scan(&campaign.ID, &campaign.Name, &campaign.CampaignType)
	
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Campaign not found",
		})
	}
	
	// Verify it's an AI campaign
	if campaign.CampaignType != "ai" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_CAMPAIGN_TYPE",
			Message: "Only AI campaigns can be transferred",
		})
	}
	
	// Transfer successful leads from leads_ai to leads
	query := `
		INSERT INTO leads (name, phone, device_id, created_at, status, sent_at)
		SELECT name, phone, device_id, created_at, 'pending', NULL
		FROM leads_ai
		WHERE campaign_id = $1 
		AND device_id = $2 
		AND status = 'delivered'
		AND NOT EXISTS (
			SELECT 1 FROM leads 
			WHERE leads.phone = leads_ai.phone 
			AND leads.device_id = leads_ai.device_id
		)
	`
	
	result, err := userRepo.DB().Exec(query, req.CampaignID, req.DeviceID)
	if err != nil {
		logrus.Errorf("Failed to transfer AI leads: %v", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to transfer leads",
		})
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	logrus.Infof("Transferred %d AI leads from campaign %s to device %s", rowsAffected, req.CampaignID, req.DeviceID)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Successfully transferred %d leads", rowsAffected),
		Results: map[string]interface{}{
			"transferred_count": rowsAffected,
			"device_id":        req.DeviceID,
			"campaign_id":      req.CampaignID,
		},
	})
}
