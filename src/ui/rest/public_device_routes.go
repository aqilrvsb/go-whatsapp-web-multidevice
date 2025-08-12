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
	publicAPI.Get("/devices", api.GetDeviceInfo)
	
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
	
	// Get date filters - handle both formats
	startDate := c.Query("start_date", c.Query("start", ""))
	endDate := c.Query("end_date", c.Query("end", ""))
	
	// Build query
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
	
	args := []interface{}{deviceID, deviceID}
	
	// Add date filters if provided
	if startDate != "" && endDate != "" {
		query += " AND c.campaign_date BETWEEN ? AND ?"
		args = append(args, startDate, endDate)
	}
	
	query += " ORDER BY c.campaign_date DESC, c.time_schedule DESC"
	
	// Execute query
	rows, err := api.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("Failed to get campaign summary: %v", err)
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
	
	// Get date filters - handle both formats
	startDate := c.Query("start_date", c.Query("start", ""))
	endDate := c.Query("end_date", c.Query("end", ""))
	
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
			WHERE device_id = ?
			GROUP BY sequence_id
		) messages ON s.id = messages.sequence_id
		WHERE s.device_id = ?
	`
	
	args := []interface{}{deviceID, deviceID}
	
	// Add date filters if provided
	if startDate != "" && endDate != "" {
		query += " AND DATE(s.created_at) BETWEEN ? AND ?"
		args = append(args, startDate, endDate)
	}
	
	query += " GROUP BY s.id ORDER BY s.created_at DESC"
	
	// Execute query
	rows, err := api.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("Failed to get sequence summary: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sequence summary"})
	}
	defer rows.Close()
	
	sequences := []fiber.Map{}
	for rows.Next() {
		var seq struct {
			ID             string
			Name           string
			Description    sql.NullString
			Niche          sql.NullString
			TargetStatus   sql.NullString
			StartTrigger   sql.NullString
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
		
		sequences = append(sequences, fiber.Map{
			"id":              seq.ID,
			"name":            seq.Name,
			"description":     seq.Description.String,
			"niche":           seq.Niche.String,
			"target_status":   seq.TargetStatus.String,
			"trigger":         seq.StartTrigger.String,
			"schedule_time":   seq.TimeSchedule.String,
			"status":          map[bool]string{true: "active", false: "inactive"}[seq.IsActive],
			"created_at":      seq.CreatedAt,
			"contacts_count":  seq.TotalContacts,
			"completed_count": seq.CompletedCount,
			"total_sent":      seq.TotalSent,
			"total_failed":    seq.TotalFailed,
			"total_pending":   seq.TotalPending,
		})
	}
	
	return c.JSON(fiber.Map{
		"sequences": sequences,
		"total":     len(sequences),
	})
}

// GetLeads returns leads for the device
func (api *PublicDeviceAPI) GetLeads(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Query leads
	query := `
		SELECT id, name, phone, niche, target_status, trigger, created_at
		FROM leads
		WHERE device_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := api.db.Query(query, deviceID)
	if err != nil {
		logrus.Errorf("Failed to get leads: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch leads"})
	}
	defer rows.Close()
	
	leads := []fiber.Map{}
	for rows.Next() {
		var lead struct {
			ID           string
			Name         string
			Phone        string
			Niche        string
			TargetStatus string
			Trigger      sql.NullString
			CreatedAt    time.Time
		}
		
		err := rows.Scan(
			&lead.ID,
			&lead.Name,
			&lead.Phone,
			&lead.Niche,
			&lead.TargetStatus,
			&lead.Trigger,
			&lead.CreatedAt,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan lead: %v", err)
			continue
		}
		
		leads = append(leads, fiber.Map{
			"id":            lead.ID,
			"name":          lead.Name,
			"phone":         lead.Phone,
			"niche":         lead.Niche,
			"target_status": lead.TargetStatus,
			"trigger":       lead.Trigger.String,
			"created_at":    lead.CreatedAt,
		})
	}
	
	return c.JSON(fiber.Map{
		"leads": leads,
		"total": len(leads),
	})
}

// GetDeviceCampaigns returns campaigns for the device
func (api *PublicDeviceAPI) GetDeviceCampaigns(c *fiber.Ctx) error {
	return api.GetCampaignSummary(c)
}

// GetDeviceSequences returns sequences for the device
func (api *PublicDeviceAPI) GetDeviceSequences(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Get sequences for this device
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
			COUNT(DISTINCT sl.id) as total_leads,
			COUNT(DISTINCT CASE WHEN sl.status = 'completed' THEN sl.id END) as completed_leads,
			COUNT(DISTINCT ss.id) as total_steps
		FROM sequences s
		LEFT JOIN sequence_leads sl ON s.id = sl.sequence_id
		LEFT JOIN sequence_steps ss ON s.id = ss.sequence_id
		WHERE s.device_id = ?
		GROUP BY s.id
		ORDER BY s.created_at DESC
	`
	
	rows, err := api.db.Query(query, deviceID)
	if err != nil {
		logrus.Errorf("Failed to get sequences: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sequences"})
	}
	defer rows.Close()
	
	sequences := []fiber.Map{}
	for rows.Next() {
		var seq struct {
			ID             string
			Name           string
			Description    sql.NullString
			Niche          sql.NullString
			TargetStatus   sql.NullString
			StartTrigger   sql.NullString
			TimeSchedule   sql.NullString
			IsActive       bool
			CreatedAt      time.Time
			TotalLeads     int
			CompletedLeads int
			TotalSteps     int
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
			&seq.TotalLeads,
			&seq.CompletedLeads,
			&seq.TotalSteps,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan sequence: %v", err)
			continue
		}
		
		sequences = append(sequences, fiber.Map{
			"id":              seq.ID,
			"name":            seq.Name,
			"description":     seq.Description.String,
			"niche":           seq.Niche.String,
			"target_status":   seq.TargetStatus.String,
			"trigger":         seq.StartTrigger.String,
			"schedule_time":   seq.TimeSchedule.String,
			"status":          map[bool]string{true: "active", false: "inactive"}[seq.IsActive],
			"created_at":      seq.CreatedAt,
			"contacts_count":  seq.TotalLeads,
			"completed_count": seq.CompletedLeads,
			"days":            seq.TotalSteps,
		})
	}
	
	return c.JSON(fiber.Map{
		"sequences": sequences,
		"total":     len(sequences),
	})
}

// GetDeviceMessages returns messages for the device
func (api *PublicDeviceAPI) GetDeviceMessages(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Get pagination params
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	
	// Query messages
	query := `
		SELECT 
			id,
			campaign_id,
			sequence_id,
			lead_id,
			message_content,
			status,
			created_at,
			sent_at
		FROM broadcast_messages
		WHERE device_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := api.db.Query(query, deviceID, limit, offset)
	if err != nil {
		logrus.Errorf("Failed to get messages: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch messages"})
	}
	defer rows.Close()
	
	messages := []fiber.Map{}
	for rows.Next() {
		var msg struct {
			ID             string
			CampaignID     sql.NullString
			SequenceID     sql.NullString
			LeadID         sql.NullString
			MessageContent string
			Status         string
			CreatedAt      time.Time
			SentAt         sql.NullTime
		}
		
		err := rows.Scan(
			&msg.ID,
			&msg.CampaignID,
			&msg.SequenceID,
			&msg.LeadID,
			&msg.MessageContent,
			&msg.Status,
			&msg.CreatedAt,
			&msg.SentAt,
		)
		
		if err != nil {
			logrus.Errorf("Failed to scan message: %v", err)
			continue
		}
		
		messages = append(messages, fiber.Map{
			"id":               msg.ID,
			"campaign_id":      msg.CampaignID.String,
			"sequence_id":      msg.SequenceID.String,
			"lead_id":          msg.LeadID.String,
			"message_content":  msg.MessageContent,
			"status":           msg.Status,
			"created_at":       msg.CreatedAt,
			"sent_at":          msg.SentAt.Time,
		})
	}
	
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM broadcast_messages WHERE device_id = ?`
	api.db.QueryRow(countQuery, deviceID).Scan(&total)
	
	return c.JSON(fiber.Map{
		"messages": messages,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func calculateSuccessRate(sent, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(sent) / float64(total) * 100
}

// GetDeviceInfo returns device data for the devices tab
func (api *PublicDeviceAPI) GetDeviceInfo(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Return device data in the format expected by the devices tab
	deviceData := fiber.Map{
		"id":           device.ID,
		"name":         device.DeviceName,
		"phone":        device.Phone,
		"status":       device.Status,
		"created_at":   device.CreatedAt,
		"updated_at":   device.UpdatedAt,
		"is_connected": device.Status == "online",
	}
	
	// Return as array since the frontend expects an array of devices
	return c.JSON(fiber.Map{
		"devices": []fiber.Map{deviceData},
		"total":   1,
	})
}

// CreateLead creates a new lead for the device
func (api *PublicDeviceAPI) CreateLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var lead struct {
		Name         string `json:"name"`
		Phone        string `json:"phone"`
		Niche        string `json:"niche"`
		TargetStatus string `json:"target_status"`
		Trigger      string `json:"trigger"`
	}
	
	if err := c.BodyParser(&lead); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	// Create lead in database
	query := `
		INSERT INTO leads (device_id, name, phone, niche, target_status, trigger, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := api.db.Exec(query, 
		deviceID, 
		lead.Name, 
		lead.Phone, 
		lead.Niche, 
		lead.TargetStatus,
		lead.Trigger,
		time.Now(),
	)
	
	if err != nil {
		logrus.Errorf("Failed to create lead: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create lead"})
	}
	
	leadID, _ := result.LastInsertId()
	
	return c.JSON(fiber.Map{
		"success": true,
		"lead_id": leadID,
		"message": "Lead created successfully",
	})
}

// UpdateLead updates an existing lead
func (api *PublicDeviceAPI) UpdateLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	leadID := c.Params("leadId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var lead struct {
		Name         string `json:"name"`
		Phone        string `json:"phone"`
		Niche        string `json:"niche"`
		TargetStatus string `json:"target_status"`
		Trigger      string `json:"trigger"`
	}
	
	if err := c.BodyParser(&lead); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	// Update lead in database
	query := `
		UPDATE leads 
		SET name = ?, phone = ?, niche = ?, target_status = ?, trigger = ?, updated_at = ?
		WHERE id = ? AND device_id = ?
	`
	
	result, err := api.db.Exec(query, 
		lead.Name, 
		lead.Phone, 
		lead.Niche, 
		lead.TargetStatus,
		lead.Trigger,
		time.Now(),
		leadID,
		deviceID,
	)
	
	if err != nil {
		logrus.Errorf("Failed to update lead: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update lead"})
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Lead not found"})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Lead updated successfully",
	})
}

// DeleteLead deletes a lead
func (api *PublicDeviceAPI) DeleteLead(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	leadID := c.Params("leadId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	// Delete lead from database
	query := `DELETE FROM leads WHERE id = ? AND device_id = ?`
	
	result, err := api.db.Exec(query, leadID, deviceID)
	
	if err != nil {
		logrus.Errorf("Failed to delete lead: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete lead"})
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Lead not found"})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Lead deleted successfully",
	})
}

// ImportLeads imports multiple leads from CSV
func (api *PublicDeviceAPI) ImportLeads(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Verify device exists
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetDeviceByID(deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}
	
	var request struct {
		Leads []struct {
			Name         string `json:"name"`
			Phone        string `json:"phone"`
			Niche        string `json:"niche"`
			TargetStatus string `json:"target_status"`
			Trigger      string `json:"trigger"`
		} `json:"leads"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	if len(request.Leads) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "No leads to import"})
	}
	
	// Start transaction
	tx, err := api.db.Begin()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to start transaction"})
	}
	defer tx.Rollback()
	
	// Prepare insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO leads (device_id, name, phone, niche, target_status, trigger, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to prepare statement"})
	}
	defer stmt.Close()
	
	// Insert each lead
	successCount := 0
	for _, lead := range request.Leads {
		_, err := stmt.Exec(
			deviceID,
			lead.Name,
			lead.Phone,
			lead.Niche,
			lead.TargetStatus,
			lead.Trigger,
			time.Now(),
		)
		if err != nil {
			logrus.Errorf("Failed to insert lead %s: %v", lead.Name, err)
			continue
		}
		successCount++
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Successfully imported %d out of %d leads", successCount, len(request.Leads)),
		"imported": successCount,
		"total": len(request.Leads),
	})
}
