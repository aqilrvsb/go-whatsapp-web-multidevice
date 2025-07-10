package rest

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
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
	// Check if user is admin
	if !isAdminUser(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	
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
	// Check if user is admin
	if !isAdminUser(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	
	ctx := context.Background()
	
	// Get current user ID (admin)
	userIDInterface := c.Locals("UserID")
	if userIDInterface == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}
	
	// Convert to UUID
	userID, err := uuid.Parse(userIDInterface.(string))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid user ID format",
		})
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
	// Check if user is admin
	if !isAdminUser(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	
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
	// Check if user is admin
	if !isAdminUser(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	
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
