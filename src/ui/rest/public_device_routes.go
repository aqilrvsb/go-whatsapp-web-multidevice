package rest

import (
	"database/sql"
	"fmt"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// InitPublicDeviceRoutes initializes public device view routes
func InitPublicDeviceRoutes(app *fiber.App, db *sql.DB) {
	// Public device view - no auth required
	app.Get("/device/:deviceId", func(c *fiber.Ctx) error {
		deviceID := c.Params("deviceId")
		if deviceID == "" {
			return c.Status(404).SendString("Device not found")
		}
		
		// Verify device exists
		userRepo := repository.GetUserRepository()
		device, err := userRepo.GetDeviceByID(deviceID)
		if err != nil {
			return c.Status(404).SendString("Device not found")
		}
		
		logrus.Infof("Public view accessed for device: %s (%s)", device.DeviceName, device.ID)
		
		// Render EXACT dashboard - public_device.html is now exact copy of dashboard.html
		return c.Render("views/public_device", fiber.Map{
			"AppHost":        fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname()),
			"AppVersion":     config.AppVersion,
			"DeviceID":       device.ID,
			"DeviceName":     device.DeviceName,
			"DevicePhone":    device.Phone,
			"MaxFileSize":    humanize.Bytes(uint64(config.WhatsappSettingMaxFileSize)),
			"MaxVideoSize":   humanize.Bytes(uint64(config.WhatsappSettingMaxVideoSize)),
			"IsPublicView":   true,
			"BasicAuthToken": nil, // No auth for public view
		})
	})
	
	// Public leads view - no auth required
	app.Get("/public/device/:deviceId/leads", func(c *fiber.Ctx) error {
		deviceID := c.Params("deviceId")
		if deviceID == "" {
			return c.Status(404).SendString("Device not found")
		}
		
		// Verify device exists
		userRepo := repository.GetUserRepository()
		device, err := userRepo.GetDeviceByID(deviceID)
		if err != nil {
			return c.Status(404).SendString("Device not found")
		}
		
		logrus.Infof("Public leads view accessed for device: %s (%s)", device.DeviceName, device.ID)
		
		return c.Render("views/public_device_leads", fiber.Map{
			"AppHost":        fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname()),
			"AppVersion":     config.AppVersion,
			"DeviceID":       device.ID,
			"DeviceName":     device.DeviceName,
			"DevicePhone":    device.Phone,
			"IsPublicView":   true,
		})
	})
}

// PublicDeviceAPI provides API endpoints for public device view
type PublicDeviceAPI struct {
	db *sql.DB
}

// InitPublicDeviceAPI initializes public device API endpoints
func InitPublicDeviceAPI(app *fiber.App) {
	api := &PublicDeviceAPI{
		db: database.GetDB(),
	}
	
	// Public API endpoints - no auth required
	publicAPI := app.Group("/api/public/device/:deviceId")
	
	// Device info endpoint
	publicAPI.Get("/devices", api.GetDevices)
	
	// Campaign summary endpoint
	publicAPI.Get("/campaign-summary", api.GetCampaignSummary)
	
	// Sequence summary endpoint  
	publicAPI.Get("/sequence-summary", api.GetSequenceSummary)
	
	// Leads endpoint
	publicAPI.Get("/leads", api.GetLeads)
	
	// Lead CRUD endpoints
	publicAPI.Post("/lead", api.CreateLead)
	publicAPI.Put("/lead/:leadId", api.UpdateLead)
	publicAPI.Delete("/lead/:leadId", api.DeleteLead)
	publicAPI.Post("/leads/import", api.ImportLeads)
	
	// Get device statistics
	publicAPI.Get("/info", api.GetDeviceStats)
	
	// Get campaigns for device
	publicAPI.Get("/campaigns", api.GetDeviceCampaigns)
	
	// Get sequences for device
	publicAPI.Get("/sequences", api.GetDeviceSequences)
	
	// Get messages for device
	publicAPI.Get("/messages", api.GetDeviceMessages)
}

func (api *PublicDeviceAPI) GetDeviceStats(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Return simple device info
	return c.JSON(fiber.Map{
		"device": fiber.Map{
			"id":         device.ID,
			"name":       device.DeviceName,
			"phone":      device.Phone,
			"status":     device.Status,
			"created_at": device.CreatedAt,
		},
	})
}

// GetCampaignSummary returns campaign data for public view
func (api *PublicDeviceAPI) GetCampaignSummary(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// First, verify device exists and get actual device info
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		logrus.Errorf("Device not found: %s - %v", deviceID, err)
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get date filters - handle both formats
	startDate := c.Query("start_date", c.Query("start", ""))
	endDate := c.Query("end_date", c.Query("end", ""))
	// TODO: Add date filtering to sequence summary query if needed
	_ = startDate // Currently unused
	_ = endDate   // Currently unused
	
	// Build query - use the actual device ID
	query := `
		SELECT 
			c.id,
			c.campaign_name as title,
			c.niche,
			c.target_status,
			c.campaign_date,
			c.time_schedule,
			c.campaign_status as status,
			c.message_template,
			c.image_url,
			c.ai_generated,
			COALESCE(stats.total_sent, 0) as total_sent,
			COALESCE(stats.total_failed, 0) as total_failed,
			COALESCE(stats.total_pending, 0) as total_pending
		FROM campaigns c
		LEFT JOIN (
			SELECT 
				campaign_id,
				SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as total_sent,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as total_failed,
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as total_pending
			FROM broadcast_messages
			WHERE device_id = ?
			GROUP BY campaign_id
		) stats ON c.id = stats.campaign_id
		WHERE c.device_id = ?
	`
	
	args := []interface{}{device.ID, device.ID}
	
	// Add date filters if provided
	if startDate != "" && endDate != "" {
		query += " AND c.campaign_date BETWEEN ? AND ?"
		args = append(args, startDate, endDate)
	}
	
	query += " ORDER BY c.campaign_date DESC, c.time_schedule DESC"
	
	// Execute query
	rows, err := api.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("Failed to get campaign summary for device %s: %v", device.ID, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch campaign summary"})
	}
	defer rows.Close()
	
	campaigns := []fiber.Map{}
	for rows.Next() {
		var campaign struct {
			ID             string
			Title          string
			Niche          sql.NullString
			TargetStatus   sql.NullString
			CampaignDate   string
			TimeSchedule   sql.NullString
			Status         string
			MessageTemplate sql.NullString
			ImageURL       sql.NullString
			AIGenerated    bool
			TotalSent      int
			TotalFailed    int
			TotalPending   int
		}
		
		err := rows.Scan(
			&campaign.ID,
			&campaign.Title,
			&campaign.Niche,
			&campaign.TargetStatus,
			&campaign.CampaignDate,
			&campaign.TimeSchedule,
			&campaign.Status,
			&campaign.MessageTemplate,
			&campaign.ImageURL,
			&campaign.AIGenerated,
			&campaign.TotalSent,
			&campaign.TotalFailed,
			&campaign.TotalPending,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan campaign: %v", err)
			continue
		}
		
		// Map status
		status := "pending"
		if campaign.Status == "completed" || campaign.TotalSent > 0 {
			status = "completed"
		} else if campaign.Status == "failed" || campaign.TotalFailed > 0 {
			status = "failed"
		} else if campaign.Status == "ongoing" {
			status = "ongoing"
		}
		
		campaigns = append(campaigns, fiber.Map{
			"id":               campaign.ID,
			"title":            campaign.Title,
			"niche":            campaign.Niche.String,
			"target_status":    campaign.TargetStatus.String,
			"campaign_date":    campaign.CampaignDate,
			"time_schedule":    campaign.TimeSchedule.String,
			"status":           status,
			"message_template": campaign.MessageTemplate.String,
			"image_url":        campaign.ImageURL.String,
			"ai":               map[bool]string{true: "ai", false: "manual"}[campaign.AIGenerated],
			"total_sent":       campaign.TotalSent,
			"total_failed":     campaign.TotalFailed,
			"total_pending":    campaign.TotalPending,
			"total_contacts":   campaign.TotalSent + campaign.TotalFailed + campaign.TotalPending,
		})
	}
	
	return c.JSON(fiber.Map{
		"campaigns": campaigns,
		"total":     len(campaigns),
	})
}

// GetSequenceSummary returns sequence data for public view
func (api *PublicDeviceAPI) GetSequenceSummary(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// First, verify device exists and get actual device info
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		logrus.Errorf("Device not found: %s - %v", deviceID, err)
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get date filters - handle both formats
	startDate := c.Query("start_date", c.Query("start", ""))
	endDate := c.Query("end_date", c.Query("end", ""))
	// TODO: Add date filtering to sequence summary query if needed
	_ = startDate // Currently unused
	_ = endDate   // Currently unused
	
	// Build query - get sequence summary with message counts
	query := `
		SELECT 
			s.id,
			s.sequence_name,
			s.description,
			s.niche,
			s.target_status,
			s.start_trigger,
			s.time_schedule,
			s.is_active,
			s.created_at,
			COUNT(DISTINCT sc.id) as total_contacts,
			COUNT(DISTINCT CASE WHEN sc.status = 'completed' THEN sc.id END) as completed_count,
			COALESCE(messages.total_sent, 0) as total_sent,
			COALESCE(messages.total_failed, 0) as total_failed,
			COALESCE(messages.total_pending, 0) as total_pending
		FROM sequences s
		LEFT JOIN sequence_contacts sc ON s.id = sc.sequence_id
		LEFT JOIN (
			SELECT 
				sequence_id,
				SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as total_sent,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as total_failed,
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as total_pending
			FROM broadcast_messages
			WHERE device_id = ? AND sequence_id IS NOT NULL
			GROUP BY sequence_id
		) messages ON s.id = messages.sequence_id
		WHERE s.device_id = ?
		GROUP BY s.id, s.sequence_name, s.description, s.niche, s.target_status, 
				 s.start_trigger, s.time_schedule, s.is_active, s.created_at,
				 messages.total_sent, messages.total_failed, messages.total_pending
		ORDER BY s.created_at DESC
	`
	
	args := []interface{}{device.ID, device.ID}
	
	// Execute query
	rows, err := api.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("Failed to get sequence summary for device %s: %v", device.ID, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sequence summary"})
	}
	defer rows.Close()
	
	sequences := []fiber.Map{}
	totalSent := 0
	totalFailed := 0
	totalPending := 0
	totalContacts := 0
	
	for rows.Next() {
		var seq struct {
			ID             string
			Name           string
			Description    sql.NullString
			Niche          sql.NullString
			TargetStatus   sql.NullString
			StartTrigger   string
			TimeSchedule   sql.NullString
			IsActive       bool
			CreatedAt      time.Time
			TotalContacts  int
			CompletedCount int
			TotalSent      int
			TotalFailed    int
			TotalPending   int
		}
		
		err := rows.Scan(
			&seq.ID,
			&seq.Name,
			&seq.Description,
			&seq.Niche,
			&seq.TargetStatus,
			&seq.StartTrigger,
			&seq.TimeSchedule,
			&seq.IsActive,
			&seq.CreatedAt,
			&seq.TotalContacts,
			&seq.CompletedCount,
			&seq.TotalSent,
			&seq.TotalFailed,
			&seq.TotalPending,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan sequence: %v", err)
			continue
		}
		
		// Update totals
		totalSent += seq.TotalSent
		totalFailed += seq.TotalFailed
		totalPending += seq.TotalPending
		totalContacts += seq.TotalContacts
		
		// Calculate completion rate
		completionRate := 0.0
		if seq.TotalContacts > 0 {
			completionRate = float64(seq.CompletedCount) / float64(seq.TotalContacts) * 100
		}
		
		sequences = append(sequences, fiber.Map{
			"id":               seq.ID,
			"name":             seq.Name,
			"description":      seq.Description.String,
			"niche":            seq.Niche.String,
			"target_status":    seq.TargetStatus.String,
			"trigger":          seq.StartTrigger,
			"time_schedule":    seq.TimeSchedule.String,
			"is_active":        seq.IsActive,
			"created_at":       seq.CreatedAt.Format("2006-01-02"),
			"total_contacts":   seq.TotalContacts,
			"completed_count":  seq.CompletedCount,
			"completion_rate":  fmt.Sprintf("%.1f", completionRate),
			"total_sent":       seq.TotalSent,
			"total_failed":     seq.TotalFailed,
			"total_pending":    seq.TotalPending,
		})
	}
	
	// Return both individual sequences and summary
	return c.JSON(fiber.Map{
		"sequences": sequences,
		"summary": fiber.Map{
			"total_sequences": len(sequences),
			"total_contacts":  totalContacts,
			"total_sent":      totalSent,
			"total_failed":    totalFailed,
			"total_pending":   totalPending,
		},
	})
}

// GetDeviceInfo returns device information
func (api *PublicDeviceAPI) GetDeviceInfo(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get device statistics
	var stats struct {
		TotalLeads     int
		TotalCampaigns int
		TotalSequences int
		MessagesSent   int
	}
	
	// Count leads
	api.db.QueryRow("SELECT COUNT(*) FROM leads WHERE device_id = ?", device.ID).Scan(&stats.TotalLeads)
	
	// Count campaigns
	api.db.QueryRow("SELECT COUNT(*) FROM campaigns WHERE device_id = ?", device.ID).Scan(&stats.TotalCampaigns)
	
	// Count sequences
	api.db.QueryRow("SELECT COUNT(*) FROM sequences WHERE device_id = ?", device.ID).Scan(&stats.TotalSequences)
	
	// Count messages sent
	api.db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = ? AND status = 'sent'", device.ID).Scan(&stats.MessagesSent)
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": fiber.Map{
			"id":              device.ID,
			"device_name":     device.DeviceName,
			"jid":             device.Phone,
			"status":          device.Status,
			"created_at":      device.CreatedAt,
			"total_leads":     stats.TotalLeads,
			"total_campaigns": stats.TotalCampaigns,
			"total_sequences": stats.TotalSequences,
			"messages_sent":   stats.MessagesSent,
		},
	})
}

// GetDevices returns all devices (for public view, returns only the current device)
func (api *PublicDeviceAPI) GetDevices(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// For public view, only return the current device
	devices := []fiber.Map{
		{
			"id":           device.ID,
			"device_name":  device.DeviceName,
			"phone":        device.Phone,
			"jid":          device.Phone,
			"status":       device.Status,
			"created_at":   device.CreatedAt,
		},
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": devices,
	})
}


// GetLeads returns leads for the device
func (api *PublicDeviceAPI) GetLeads(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get pagination
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)
	offset := (page - 1) * perPage
	
	// Get filters
	niche := c.Query("niche", "")
	status := c.Query("status", "")
	search := c.Query("search", "")
	
	// Build query
	query := "SELECT id, phone, name, niche, target_status, created_at FROM leads WHERE device_id = ?"
	countQuery := "SELECT COUNT(*) FROM leads WHERE device_id = ?"
	args := []interface{}{device.ID}
	
	// Add filters
	if niche != "" && niche != "all" {
		query += " AND niche = ?"
		countQuery += " AND niche = ?"
		args = append(args, niche)
	}
	
	if status != "" && status != "all" {
		query += " AND target_status = ?"
		countQuery += " AND target_status = ?"
		args = append(args, status)
	}
	
	if search != "" {
		query += " AND (phone LIKE ? OR name LIKE ?)"
		countQuery += " AND (phone LIKE ? OR name LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}
	
	// Get total count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	api.db.QueryRow(countQuery, countArgs...).Scan(&total)
	
	// Add pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)
	
	// Execute query
	rows, err := api.db.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch leads"})
	}
	defer rows.Close()
	
	leads := []fiber.Map{}
	for rows.Next() {
		var lead struct {
			ID           string
			Phone        string
			Name         sql.NullString
			Niche        sql.NullString
			TargetStatus sql.NullString
			CreatedAt    time.Time
		}
		
		err := rows.Scan(&lead.ID, &lead.Phone, &lead.Name, &lead.Niche, &lead.TargetStatus, &lead.CreatedAt)
		if err != nil {
			continue
		}
		
		leads = append(leads, fiber.Map{
			"id":            lead.ID,
			"phone":         lead.Phone,
			"name":          lead.Name.String,
			"niche":         lead.Niche.String,
			"target_status": lead.TargetStatus.String,
			"created_at":    lead.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": fiber.Map{
			"leads": leads,
			"pagination": fiber.Map{
				"page":     page,
				"per_page": perPage,
				"total":    total,
				"pages":    (total + perPage - 1) / perPage,
			},
		},
	})
}

// CreateLead creates a new lead
func (api *PublicDeviceAPI) CreateLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var lead struct {
		Phone        string `json:"phone"`
		Name         string `json:"name"`
		Niche        string `json:"niche"`
		TargetStatus string `json:"target_status"`
	}
	
	if err := c.BodyParser(&lead); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	// Insert lead
	result, err := api.db.Exec(
		"INSERT INTO leads (device_id, phone, name, niche, target_status) VALUES (?, ?, ?, ?, ?)",
		device.ID, lead.Phone, lead.Name, lead.Niche, lead.TargetStatus,
	)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create lead"})
	}
	
	leadID, _ := result.LastInsertId()
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"message": "Lead created successfully",
		"lead_id": leadID,
	})
}

// UpdateLead updates an existing lead
func (api *PublicDeviceAPI) UpdateLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	leadID := c.Params("leadId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var lead struct {
		Phone        string `json:"phone"`
		Name         string `json:"name"`
		Niche        string `json:"niche"`
		TargetStatus string `json:"target_status"`
	}
	
	if err := c.BodyParser(&lead); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	// Update lead
	_, err = api.db.Exec(
		"UPDATE leads SET phone = ?, name = ?, niche = ?, target_status = ? WHERE id = ? AND device_id = ?",
		lead.Phone, lead.Name, lead.Niche, lead.TargetStatus, leadID, device.ID,
	)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update lead"})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"message": "Lead updated successfully",
	})
}

// DeleteLead deletes a lead
func (api *PublicDeviceAPI) DeleteLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	leadID := c.Params("leadId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Delete lead
	_, err = api.db.Exec("DELETE FROM leads WHERE id = ? AND device_id = ?", leadID, device.ID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete lead"})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"message": "Lead deleted successfully",
	})
}

// ImportLeads imports multiple leads
func (api *PublicDeviceAPI) ImportLeads(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var leads []struct {
		Phone  string `json:"phone"`
		Name   string `json:"name"`
		Niche  string `json:"niche"`
		Status string `json:"status"`
	}
	
	if err := c.BodyParser(&leads); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	// Import leads
	imported := 0
	for _, lead := range leads {
		_, err := api.db.Exec(
			"INSERT INTO leads (device_id, phone, name, niche, status) VALUES (?, ?, ?, ?, ?)",
			device.ID, lead.Phone, lead.Name, lead.Niche, lead.Status,
		)
		if err == nil {
			imported++
		}
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"message": fmt.Sprintf("Imported %d leads successfully", imported),
		"imported": imported,
		"total": len(leads),
	})
}

// GetDeviceCampaigns returns campaigns for a device
func (api *PublicDeviceAPI) GetDeviceCampaigns(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get campaigns filtered by user_id
	rows, err := api.db.Query(`
		SELECT id, campaign_name, niche, target_status, campaign_date, 
		       time_schedule, campaign_status, message_template, image_url
		FROM campaigns 
		WHERE user_id = ?
		ORDER BY campaign_date DESC, time_schedule DESC
	`, device.UserID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch campaigns"})
	}
	defer rows.Close()
	
	campaigns := []fiber.Map{}
	for rows.Next() {
		var campaign struct {
			ID             string
			Name           string
			Niche          sql.NullString
			TargetStatus   sql.NullString
			CampaignDate   string
			TimeSchedule   sql.NullString
			Status         string
			MessageTemplate sql.NullString
			ImageURL       sql.NullString
		}
		
		err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.Niche,
			&campaign.TargetStatus,
			&campaign.CampaignDate,
			&campaign.TimeSchedule,
			&campaign.Status,
			&campaign.MessageTemplate,
			&campaign.ImageURL,
		)
		
		if err != nil {
			continue
		}
		
		campaigns = append(campaigns, fiber.Map{
			"id":               campaign.ID,
			"campaign_name":    campaign.Name,
			"niche":            campaign.Niche.String,
			"target_status":    campaign.TargetStatus.String,
			"campaign_date":    campaign.CampaignDate,
			"time_schedule":    campaign.TimeSchedule.String,
			"campaign_status":  campaign.Status,
			"message_template": campaign.MessageTemplate.String,
			"image_url":        campaign.ImageURL.String,
		})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": campaigns,
	})
}

// GetDeviceSequences returns sequences for a device
func (api *PublicDeviceAPI) GetDeviceSequences(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get sequences with statistics filtered by user_id
	rows, err := api.db.Query(`
		SELECT 
			s.id,
			s.sequence_name as name,
			s.start_trigger as trigger,
			s.is_active,
			COUNT(DISTINCT ss.id) as total_flows,
			COUNT(DISTINCT sc.id) as total_contacts,
			COUNT(DISTINCT CASE WHEN sc.status = 'completed' THEN sc.id END) as contacts_done,
			COUNT(DISTINCT CASE WHEN sc.status = 'failed' THEN sc.id END) as contacts_failed
		FROM sequences s
		LEFT JOIN sequence_steps ss ON s.id = ss.sequence_id
		LEFT JOIN sequence_contacts sc ON s.id = sc.sequence_id
		WHERE s.user_id = ?
		GROUP BY s.id, s.sequence_name, s.start_trigger, s.is_active
		ORDER BY s.created_at DESC
	`, device.UserID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sequences"})
	}
	defer rows.Close()
	
	sequences := []fiber.Map{}
	for rows.Next() {
		var seq struct {
			ID             string
			Name           string
			Trigger        sql.NullString
			IsActive       bool
			TotalFlows     int
			TotalContacts  int
			ContactsDone   int
			ContactsFailed int
		}
		
		err := rows.Scan(
			&seq.ID,
			&seq.Name,
			&seq.Trigger,
			&seq.IsActive,
			&seq.TotalFlows,
			&seq.TotalContacts,
			&seq.ContactsDone,
			&seq.ContactsFailed,
		)
		
		if err != nil {
			continue
		}
		
		// Calculate success rate
		successRate := 0.0
		if seq.TotalContacts > 0 {
			successRate = float64(seq.ContactsDone) / float64(seq.TotalContacts) * 100
		}
		
		sequences = append(sequences, fiber.Map{
			"id":               seq.ID,
			"name":             seq.Name,
			"trigger":          seq.Trigger.String,
			"is_active":        seq.IsActive,
			"total_flows":      seq.TotalFlows,
			"total_contacts":   seq.TotalContacts,
			"contacts_done":    seq.ContactsDone,
			"contacts_failed":  seq.ContactsFailed,
			"success_rate":     fmt.Sprintf("%.1f", successRate),
		})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": sequences,
	})
}

// GetDeviceMessages returns recent messages for a device
func (api *PublicDeviceAPI) GetDeviceMessages(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Get recent messages
	rows, err := api.db.Query(`
		SELECT 
			id,
			recipient_phone,
			recipient_name,
			message_content,
			status,
			created_at,
			sent_at,
			error_message
		FROM broadcast_messages
		WHERE device_id = ?
		ORDER BY created_at DESC
		LIMIT 100
	`, device.ID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch messages"})
	}
	defer rows.Close()
	
	messages := []fiber.Map{}
	for rows.Next() {
		var msg struct {
			ID             string
			RecipientPhone string
			RecipientName  sql.NullString
			MessageContent string
			Status         string
			CreatedAt      time.Time
			SentAt         sql.NullTime
			ErrorMessage   sql.NullString
		}
		
		err := rows.Scan(
			&msg.ID,
			&msg.RecipientPhone,
			&msg.RecipientName,
			&msg.MessageContent,
			&msg.Status,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.ErrorMessage,
		)
		
		if err != nil {
			continue
		}
		
		messages = append(messages, fiber.Map{
			"id":               msg.ID,
			"recipient_phone":  msg.RecipientPhone,
			"recipient_name":   msg.RecipientName.String,
			"message_content":  msg.MessageContent,
			"status":           msg.Status,
			"created_at":       msg.CreatedAt.Format("2006-01-02 15:04:05"),
			"sent_at":          msg.SentAt.Time.Format("2006-01-02 15:04:05"),
			"error_message":    msg.ErrorMessage.String,
		})
	}
	
	return c.JSON(fiber.Map{
		"code": "SUCCESS",
		"results": messages,
	})
}
