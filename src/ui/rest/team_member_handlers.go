package rest

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
)

// TeamMemberHandlers contains all team member related handlers
type TeamMemberHandlers struct {
	repo *repository.TeamMemberRepository
}

func NewTeamMemberHandlers(repo *repository.TeamMemberRepository) *TeamMemberHandlers {
	return &TeamMemberHandlers{repo: repo}
}

// GetAllTeamMembers returns all team members with device counts
func (h *TeamMemberHandlers) GetAllTeamMembers(c *fiber.Ctx) error {
	// The CustomAuth middleware should have already authenticated the user
	// No need for additional checks here
	
	ctx := context.Background()
	
	members, err := h.repo.GetAllWithDeviceCount(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get team members",
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"data":    members,
	})
}

// CreateTeamMember creates a new team member
func (h *TeamMemberHandlers) CreateTeamMember(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get current user ID (admin) - try different context keys
	var userID uuid.UUID
	var err error
	
	// Try to get user ID from different possible context keys
	if userIDStr, ok := c.Locals("UserID").(string); ok {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			// If not a valid UUID string, generate a default one
			userID = uuid.New()
		}
	} else if userIDUUID, ok := c.Locals("UserID").(uuid.UUID); ok {
		userID = userIDUUID
	} else {
		// If no user ID found, use a default UUID
		// This is okay since we're already authenticated by middleware
		userID = uuid.New()
	}
	
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Validate inputs
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}
	
	// Check if username already exists
	existing, err := h.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing username",
		})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Username already exists",
		})
	}
	
	// Create team member
	member := &models.TeamMember{
		Username:  req.Username,
		Password:  req.Password,
		CreatedBy: userID,
		IsActive:  true,
	}
	
	if err := h.repo.Create(ctx, member); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create team member",
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"data":    member,
	})
}

// UpdateTeamMember updates an existing team member
func (h *TeamMemberHandlers) UpdateTeamMember(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member ID from params
	memberID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid team member ID",
		})
	}
	
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsActive bool   `json:"is_active"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Get existing member
	member, err := h.repo.GetByID(ctx, memberID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get team member",
		})
	}
	if member == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Team member not found",
		})
	}
	
	// Update fields
	if req.Username != "" {
		member.Username = strings.TrimSpace(req.Username)
	}
	if req.Password != "" {
		member.Password = strings.TrimSpace(req.Password)
	}
	member.IsActive = req.IsActive
	
	// Save updates
	if err := h.repo.Update(ctx, member); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update team member",
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"data":    member,
	})
}

// DeleteTeamMember deletes a team member
func (h *TeamMemberHandlers) DeleteTeamMember(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member ID from params
	memberID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid team member ID",
		})
	}
	
	// Delete team member
	if err := h.repo.Delete(ctx, memberID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete team member",
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Team member deleted successfully",
	})
}

// LoginTeamMember handles team member login
func (h *TeamMemberHandlers) LoginTeamMember(c *fiber.Ctx) error {
	ctx := context.Background()
	
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Find team member
	member, err := h.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check credentials",
		})
	}
	
	if member == nil || member.Password != req.Password || !member.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials or account inactive",
		})
	}
	
	// Create session
	session, err := h.repo.CreateSession(ctx, member.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}
	
	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "team_session",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
	})
	
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"member": member,
			"token":  session.Token,
		},
	})
}

// LogoutTeamMember handles team member logout
func (h *TeamMemberHandlers) LogoutTeamMember(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get token from cookie
	token := c.Cookies("team_session")
	if token != "" {
		// Delete session
		h.repo.DeleteSession(ctx, token)
	}
	
	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "team_session",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}

// TeamMemberAuthMiddleware checks if the request is from a valid team member
func (h *TeamMemberHandlers) TeamMemberAuthMiddleware(c *fiber.Ctx) error {
	// Get token from cookie
	token := c.Cookies("team_session")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	ctx := context.Background()
	
	// Get session
	session, err := h.repo.GetSessionByToken(ctx, token)
	if err != nil || session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired session",
		})
	}
	
	// Get team member
	member, err := h.repo.GetByID(ctx, session.TeamMemberID)
	if err != nil || member == nil || !member.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid team member",
		})
	}
	
	// Store in context
	c.Locals("teamMember", member)
	c.Locals("isTeamMember", true)
	
	return c.Next()
}
// GetTeamMemberInfo returns the current team member info
func (h *TeamMemberHandlers) GetTeamMemberInfo(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member from context (set by middleware)
	member, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// Get device IDs for this team member
	deviceIDs, err := h.repo.GetDeviceIDsForMember(ctx, member.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get device IDs",
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"member": fiber.Map{
			"id":       member.ID,
			"username": member.Username,
		},
		"device_ids": deviceIDs,
	})
}
// isAdminUser checks if the current user is an admin (not a team member)
func isAdminUser(c *fiber.Ctx) bool {
	// Check if user is authenticated as team member
	if c.Locals("isTeamMember") == true {
		return false
	}
	
	// Check if user has a valid user session (admin)
	// Note: The middleware sets "UserID" with capital U
	userID := c.Locals("UserID")
	return userID != nil
}

// adminOnly middleware ensures only admin users can access
func adminOnly(c *fiber.Ctx) error {
	if !isAdminUser(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	return c.Next()
}

// GetTeamDevices returns devices accessible to the team member
func (h *TeamMemberHandlers) GetTeamDevices(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member from context
	member, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// Get devices that match team member's username from repository
	devices, err := h.repo.GetTeamMemberDevices(ctx, member.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch devices",
		})
	}
	
	return c.JSON(devices)
}

// GetTeamCampaignsSummary returns campaign summary for team member's devices
func (h *TeamMemberHandlers) GetTeamCampaignsSummary(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member from context
	member, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// Get team member's devices
	devices, err := h.repo.GetTeamMemberDevices(ctx, member.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch devices",
		})
	}
	
	// Extract device IDs
	deviceIDs := []string{}
	for _, device := range devices {
		if deviceID, ok := device["id"].(uuid.UUID); ok {
			deviceIDs = append(deviceIDs, deviceID.String())
		}
	}
	
	// Get date filter from query parameters
	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")
	
	// Get campaigns that use these devices
	campaignRepo := repository.GetCampaignRepository()
	campaigns := []models.Campaign{}
	
	// Get all campaigns for the user (team members see all campaigns but filtered data)
	allCampaigns, err := campaignRepo.GetCampaignsByUser("")
	if err == nil {
		for _, campaign := range allCampaigns {
			// Check if campaign uses any of team member's devices
			hasDevice := false
			for _, deviceID := range deviceIDs {
				if campaignUsesDevice(int64(campaign.ID), deviceID) {
					hasDevice = true
					break
				}
			}
			if hasDevice {
				// Apply date filter if provided
				if startDate != "" || endDate != "" {
					campaignDate, _ := time.Parse("2006-01-02", campaign.CampaignDate)
					if startDate != "" {
						startDateTime, _ := time.Parse("2006-01-02", startDate)
						if campaignDate.Before(startDateTime) {
							continue
						}
					}
					if endDate != "" {
						endDateTime, _ := time.Parse("2006-01-02", endDate)
						if campaignDate.After(endDateTime) {
							continue
						}
					}
				}
				campaigns = append(campaigns, campaign)
			}
		}
	}
	
	// Calculate statistics (same as admin)
	totalCampaigns := len(campaigns)
	pendingCampaigns := 0
	triggeredCampaigns := 0
	processingCampaigns := 0
	sentCampaigns := 0
	failedCampaigns := 0
	
	for _, campaign := range campaigns {
		switch campaign.Status {
		case "scheduled", "pending":
			pendingCampaigns++
		case "triggered":
			triggeredCampaigns++
		case "processing":
			processingCampaigns++
		case "sent", "finished":
			sentCampaigns++
		case "failed":
			failedCampaigns++
		}
	}
	
	// Get broadcast statistics
	totalShouldSend := 0
	totalDoneSend := 0
	totalFailedSend := 0
	
	for _, campaign := range campaigns {
		// Get stats only for team member's devices
		for _, deviceID := range deviceIDs {
			shouldSend, doneSend, failedSend := getCampaignDeviceStats(int64(campaign.ID), deviceID)
			totalShouldSend += shouldSend
			totalDoneSend += doneSend
			totalFailedSend += failedSend
		}
	}
	
	totalRemainingSend := totalShouldSend - totalDoneSend - totalFailedSend
	if totalRemainingSend < 0 {
		totalRemainingSend = 0
	}
	
	// Get recent campaigns with their broadcast stats
	recentCampaigns := []map[string]interface{}{}
	if len(campaigns) > 0 {
		limit := min(5, len(campaigns))
		for i := 0; i < limit; i++ {
			campaign := campaigns[i]
			
			// Get broadcast stats for this campaign (only team devices)
			shouldSend := 0
			doneSend := 0
			failedSend := 0
			
			for _, deviceID := range deviceIDs {
				s, d, f := getCampaignDeviceStats(int64(campaign.ID), deviceID)
				shouldSend += s
				doneSend += d
				failedSend += f
			}
			
			remainingSend := shouldSend - doneSend - failedSend
			if remainingSend < 0 {
				remainingSend = 0
			}
			
			campaignData := map[string]interface{}{
				"id":               campaign.ID,
				"title":            campaign.Title,
				"campaign_date":    campaign.CampaignDate,
				"time_schedule":    campaign.TimeSchedule,
				"niche":            campaign.Niche,
				"target_status":    campaign.TargetStatus,
				"status":           campaign.Status,
				"message":          campaign.Message,
				"image_url":        campaign.ImageURL,
				"should_send":      shouldSend,
				"done_send":        doneSend,
				"failed_send":      failedSend,
				"remaining_send":   remainingSend,
			}
			
			recentCampaigns = append(recentCampaigns, campaignData)
		}
	}
	
	summary := map[string]interface{}{
		"campaigns": map[string]interface{}{
			"total": totalCampaigns,
			"pending": pendingCampaigns,
			"triggered": triggeredCampaigns,
			"processing": processingCampaigns,
			"sent": sentCampaigns,
			"failed": failedCampaigns,
		},
		"broadcast_stats": map[string]interface{}{
			"total_should_send":    totalShouldSend,
			"total_done_send":      totalDoneSend,
			"total_failed_send":    totalFailedSend,
			"total_remaining_send": totalRemainingSend,
		},
		"recent_campaigns": recentCampaigns,
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Campaign summary",
		Results: summary,
	})
}

// GetTeamCampaignsAnalytics returns campaign analytics for team member's devices  
func (h *TeamMemberHandlers) GetTeamCampaignsAnalytics(c *fiber.Ctx) error {
	// Get team member from context
	_, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// For now, return simplified analytics
	// In production, you'd calculate real metrics based on team member's devices
	return c.JSON(fiber.Map{
		"total_campaigns": 0,
		"total_messages": 0,
		"success_rate": 0,
		"devices_used": 0,
		"time_series": []fiber.Map{},
		"device_performance": []fiber.Map{},
	})
}

// GetTeamSequencesSummary returns sequence summary for team member's devices
func (h *TeamMemberHandlers) GetTeamSequencesSummary(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Get team member from context
	member, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// Get team member's devices
	devices, err := h.repo.GetTeamMemberDevices(ctx, member.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch devices",
		})
	}
	
	// Extract device IDs
	deviceIDs := []string{}
	for _, device := range devices {
		if deviceID, ok := device["id"].(uuid.UUID); ok {
			deviceIDs = append(deviceIDs, deviceID.String())
		}
	}
	
	// Get sequences that use these devices
	db := database.GetDB()
	
	// Count sequences that have been used with team member's devices
	var totalSequences int
	query := `
		SELECT COUNT(DISTINCT s.id) 
		FROM sequences s
		JOIN sequence_contacts sc ON s.id = sc.sequence_id
		WHERE sc.processing_device_id = ANY($1)
	`
	db.QueryRow(query, pq.Array(deviceIDs)).Scan(&totalSequences)
	
	// Get flow and contact statistics
	var totalFlows, totalShouldSend, totalDoneSend, totalFailedSend int
	
	// Count total flows
	query = `
		SELECT COUNT(DISTINCT sequence_stepid) 
		FROM sequence_contacts 
		WHERE processing_device_id = ANY($1)
	`
	db.QueryRow(query, pq.Array(deviceIDs)).Scan(&totalFlows)
	
	// Count contacts
	query = `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as done,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM sequence_contacts
		WHERE processing_device_id = ANY($1)
	`
	db.QueryRow(query, pq.Array(deviceIDs)).Scan(&totalShouldSend, &totalDoneSend, &totalFailedSend)
	
	totalRemainingSend := totalShouldSend - totalDoneSend - totalFailedSend
	if totalRemainingSend < 0 {
		totalRemainingSend = 0
	}
	
	// Get recent sequences
	recentSequences := []map[string]interface{}{}
	query = `
		SELECT DISTINCT s.id, s.name, s.trigger, s.niche, s.status
		FROM sequences s
		JOIN sequence_contacts sc ON s.id = sc.sequence_id
		WHERE sc.processing_device_id = ANY($1)
		ORDER BY s.created_at DESC
		LIMIT 5
	`
	rows, err := db.Query(query, pq.Array(deviceIDs))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var seq struct {
				ID      int64
				Name    string
				Trigger string
				Niche   string
				Status  string
			}
			if err := rows.Scan(&seq.ID, &seq.Name, &seq.Trigger, &seq.Niche, &seq.Status); err == nil {
				// Get stats for this sequence
				var seqShould, seqDone, seqFailed int
				statsQuery := `
					SELECT 
						COUNT(*) as total,
						COUNT(CASE WHEN status = 'sent' THEN 1 END) as done,
						COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
					FROM sequence_contacts
					WHERE sequence_id = $1 AND processing_device_id = ANY($2)
				`
				db.QueryRow(statsQuery, seq.ID, pq.Array(deviceIDs)).Scan(&seqShould, &seqDone, &seqFailed)
				
				recentSequences = append(recentSequences, map[string]interface{}{
					"id":              seq.ID,
					"name":            seq.Name,
					"trigger":         seq.Trigger,
					"niche":           seq.Niche,
					"status":          seq.Status,
					"should_send":     seqShould,
					"done_send":       seqDone,
					"failed_send":     seqFailed,
					"remaining_send":  seqShould - seqDone - seqFailed,
				})
			}
		}
	}
	
	return c.JSON(fiber.Map{
		"sequences": fiber.Map{
			"total": totalSequences,
			"active": 0, // You can calculate these based on status
			"inactive": 0,
		},
		"total_flows": totalFlows,
		"total_should_send": totalShouldSend,
		"total_done_send": totalDoneSend,
		"total_failed_send": totalFailedSend,
		"total_remaining_send": totalRemainingSend,
		"recent_sequences": recentSequences,
	})
}

// GetTeamSequencesAnalytics returns sequence analytics for team member's devices
func (h *TeamMemberHandlers) GetTeamSequencesAnalytics(c *fiber.Ctx) error {
	// Get team member from context
	_, ok := c.Locals("teamMember").(*models.TeamMember)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// Return simplified analytics for now
	return c.JSON(fiber.Map{
		"total_sequences": 0,
		"total_contacts": 0,
		"completion_rate": 0,
		"active_flows": 0,
		"time_series": []fiber.Map{},
		"sequence_performance": []fiber.Map{},
	})
}

// Helper function to check if campaign uses a specific device
func campaignUsesDevice(campaignID int64, deviceID string) bool {
	db := database.GetDB()
	var count int
	query := `SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = $1 AND device_id = $2`
	db.QueryRow(query, campaignID, deviceID).Scan(&count)
	return count > 0
}

// Helper function to get campaign stats for a specific device
func getCampaignDeviceStats(campaignID int64, deviceID string) (shouldSend, doneSend, failedSend int) {
	db := database.GetDB()
	
	// Get total messages for this device
	var total int
	query := `SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = $1 AND device_id = $2`
	db.QueryRow(query, campaignID, deviceID).Scan(&total)
	shouldSend = total
	
	// Get sent messages
	query = `SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = $1 AND device_id = $2 AND status = 'sent'`
	db.QueryRow(query, campaignID, deviceID).Scan(&doneSend)
	
	// Get failed messages
	query = `SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = $1 AND device_id = $2 AND status = 'failed'`
	db.QueryRow(query, campaignID, deviceID).Scan(&failedSend)
	
	return
}


