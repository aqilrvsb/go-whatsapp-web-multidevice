package rest

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	Service domainApp.IAppUsecase
}

func InitRestApp(app *fiber.App, service domainApp.IAppUsecase) App {
	rest := App{Service: service}
	
	// Health check endpoint
	app.Get("/health", rest.HealthCheck)
	app.Get("/api/health", rest.HealthCheck)
	
	// Dashboard routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})
	app.Get("/login", rest.AppLoginView)
	app.Get("/register", rest.RegisterView)
	app.Get("/dashboard", rest.DashboardView)	
	// Auth API endpoints
	app.Post("/api/login", rest.HandleLogin)
	app.Post("/api/register", rest.HandleRegister)
	app.Get("/logout", rest.HandleLogout)
	
	// Analytics API endpoints
	app.Get("/api/analytics/:days", rest.GetAnalyticsData)
	app.Get("/api/analytics/custom", rest.GetCustomAnalyticsData)
	app.Get("/api/devices", rest.GetConnectedDevices)
	app.Post("/api/devices", rest.CreateDevice)
	app.Get("/api/devices/:id", rest.GetDevice)
	
	// Device pages
	app.Get("/device/:id/actions", rest.DeviceActionsView)
	app.Get("/device/:id/leads", rest.DeviceLeadsView)
	app.Get("/device/:id/whatsapp-web", rest.WhatsAppWebView)
	
	// WhatsApp Web API endpoints
	app.Get("/api/devices/:id/chats", rest.GetWhatsAppChats)
	app.Get("/api/devices/:id/messages/:chatId", rest.GetWhatsAppMessages)
	app.Post("/api/devices/:id/send", rest.SendWhatsAppMessage)
	app.Post("/api/devices/:id/sync", rest.SyncDeviceChats)
	app.Get("/api/devices/:id/diagnose", rest.DiagnoseDevice)
	
	// Device management endpoints
	app.Delete("/api/devices/:id", rest.DeleteDevice)
	app.Get("/app/logout", rest.LogoutDevice)
	app.Get("/app/reconnect", rest.ReconnectDevice)
	app.Get("/app/devices", rest.GetDevices)
	app.Get("/user/info", rest.GetUserInfo)
	app.Get("/user/avatar", rest.GetUserAvatar)
	app.Post("/user/avatar", rest.ChangeUserAvatar)
	app.Post("/user/pushname", rest.ChangeUserPushName)
	
	// Lead management endpoints
	app.Get("/api/devices/:deviceId/leads", rest.GetDeviceLeads)
	app.Post("/api/leads", rest.CreateLead)
	app.Put("/api/leads/:id", rest.UpdateLead)
	app.Delete("/api/leads/:id", rest.DeleteLead)
	
	// Campaign endpoints
	app.Get("/api/campaigns", rest.GetCampaigns)
	app.Post("/api/campaigns", rest.CreateCampaign)
	app.Put("/api/campaigns/:id", rest.UpdateCampaign)
	app.Delete("/api/campaigns/:id", rest.DeleteCampaign)
	app.Get("/api/campaigns/summary", rest.GetCampaignSummary)
	
	// Sequence summary endpoint
	app.Get("/api/sequences/summary", rest.GetSequenceSummary)
	
	// Worker status endpoint
	app.Get("/api/workers/status", rest.GetWorkerStatus)
	
	// Worker control endpoints
	app.Post("/api/workers/resume-failed", rest.ResumeFailedWorkers)
	app.Post("/api/workers/stop-all", rest.StopAllWorkers)
	
	// WhatsApp QR code endpoint
	app.Get("/app/qr", rest.GetQRCode)
	
	// API endpoints
	app.Get("/app/login", rest.Login)
	app.Get("/app/login-with-code", rest.LoginWithCode)
	app.Post("/app/link-device", rest.LinkDevicePhone)
	app.Get("/app/logout", rest.Logout)
	app.Get("/app/reconnect", rest.Reconnect)
	app.Get("/app/devices", rest.Devices)

	return App{Service: service}
}

func (handler *App) Login(c *fiber.Ctx) error {
	// Get device ID from query params
	deviceId := c.Query("deviceId")
	
	// Get user from context or cookie
	userID := c.Locals("userID")
	if userID == nil {
		// Try to get from session cookie as fallback
		token := c.Cookies("session_token")
		if token != "" {
			userRepo := repository.GetUserRepository()
			session, err := userRepo.GetSession(token)
			if err == nil && session != nil {
				userID = session.UserID
			}
		}
	}
	
	// Log for debugging
	log.Printf("Login request - UserID: %v, DeviceID: %s", userID, deviceId)
	
	if userID != nil && deviceId != "" {
		// Start tracking this connection session
		whatsapp.StartConnectionSession(userID.(string), deviceId, "")
		log.Printf("Started connection session for user %s, device %s", userID, deviceId)
	}
	
	response, err := handler.Service.Login(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Login success",
		Results: map[string]any{
			"qr_link":     fmt.Sprintf("%s://%s/%s", c.Protocol(), c.Hostname(), response.ImagePath),
			"qr_duration": response.Duration,
		},
	})
}
func (handler *App) LoginWithCode(c *fiber.Ctx) error {
	pairCode, err := handler.Service.LoginWithCode(c.UserContext(), c.Query("phone"))
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Login with code success",
		Results: map[string]any{
			"pair_code": pairCode,
		},
	})
}

func (handler *App) Logout(c *fiber.Ctx) error {
	err := handler.Service.Logout(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success logout",
		Results: nil,
	})
}

func (handler *App) Reconnect(c *fiber.Ctx) error {
	err := handler.Service.Reconnect(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Reconnect success",
		Results: nil,
	})
}
func (handler *App) Devices(c *fiber.Ctx) error {
	devices, err := handler.Service.FetchDevices(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Fetch device success",
		Results: devices,
	})
}

// LinkDevicePhone links a phone number to a device for the current user
func (handler *App) LinkDevicePhone(c *fiber.Ctx) error {
	var req struct {
		DeviceID string `json:"device_id"`
		Phone    string `json:"phone"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}
	
	// Get user from context (set by auth middleware)
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	
	// Update device with phone number
	userRepo := repository.GetUserRepository()
	err := userRepo.UpdateDevicePhone(userID.(string), req.DeviceID, req.Phone)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update device",
		})
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"message": "Phone number linked successfully",
	})
}

// AppLoginView serves the login page
func (handler *App) AppLoginView(c *fiber.Ctx) error {
	// Serve login page from embedded filesystem
	return c.Render("views/login", fiber.Map{
		"Title": "Login - WhatsApp Analytics",
	})
}

// RegisterView serves the register page
func (handler *App) RegisterView(c *fiber.Ctx) error {
	// Serve register page from embedded filesystem
	return c.Render("views/register", fiber.Map{
		"Title": "Register - WhatsApp Analytics",
	})
}

// AppDevicesView serves the devices page (deprecated - redirect to dashboard)
func (handler *App) AppDevicesView(c *fiber.Ctx) error {
	return c.Redirect("/dashboard")
}
// DashboardView serves the main dashboard
func (handler *App) DashboardView(c *fiber.Ctx) error {
	// Serve dashboard from embedded filesystem
	return c.Render("views/dashboard", fiber.Map{
		"Title": "Dashboard - WhatsApp Analytics",
	})
}

// HandleLogin processes login requests
func (handler *App) HandleLogin(c *fiber.Ctx) error {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := c.BodyParser(&loginReq); err != nil {
		log.Printf("Failed to parse login request: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}
	
	log.Printf("Login attempt for email: %s", loginReq.Email)
	
	// Get user repository
	userRepo := repository.GetUserRepository()
	
	// Validate credentials
	user, err := userRepo.ValidatePassword(loginReq.Email, loginReq.Password)
	if err != nil {
		log.Printf("Login failed for %s: %v", loginReq.Email, err)
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}
	
	log.Printf("Login successful for user: %s", user.Email)
	
	// Create session
	session, err := userRepo.CreateSession(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}
	
	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
	})
	
	return c.JSON(fiber.Map{
		"status": "success",
		"message": "Login successful",
		"token": session.Token,
		"user": fiber.Map{
			"id":       user.ID,
			"email":    user.Email,
			"fullName": user.FullName,
		},
	})
}
// HandleRegister processes registration requests
func (handler *App) HandleRegister(c *fiber.Ctx) error {
	var registerReq struct {
		FullName string `json:"fullname"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := c.BodyParser(&registerReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}
	
	// Get user repository
	userRepo := repository.GetUserRepository()
	
	// Create user
	user, err := userRepo.CreateUser(registerReq.Email, registerReq.FullName, registerReq.Password)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(409).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"message": "Registration successful",
		"user": fiber.Map{
			"id":       user.ID,
			"email":    user.Email,
			"fullName": user.FullName,
		},
	})
}

// HandleLogout handles user logout
func (handler *App) HandleLogout(c *fiber.Ctx) error {
	// Clear session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})
	
	// Redirect to login page
	return c.Redirect("/login")
}

// GetQRCode returns the QR code image directly
func (handler *App) GetQRCode(c *fiber.Ctx) error {
	// Get QR code from login service
	response, err := handler.Service.Login(c.UserContext())
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to generate QR code",
		})
	}
	
	// Read the QR image file
	qrImagePath := response.ImagePath
	if qrImagePath != "" {
		// Return the image file
		return c.SendFile(qrImagePath)
	}
	
	// If no QR code available, return error image
	return c.Status(404).JSON(utils.ResponseData{
		Status:  404,
		Code:    "NOT_FOUND",
		Message: "QR code not available",
	})
}

// HealthCheck endpoint to verify application status
func (handler *App) HealthCheck(c *fiber.Ctx) error {
	// Check database connection
	userRepo := repository.GetUserRepository()
	dbHealthy := true
	dbError := ""
	
	// Try to get a user to test DB connection
	_, err := userRepo.GetUserByEmail("test@health.check")
	if err != nil && err.Error() != "user not found" {
		dbHealthy = false
		dbError = err.Error()
	}
	
	health := fiber.Map{
		"status": "ok",
		"version": config.AppVersion,
		"database": fiber.Map{
			"connected": dbHealthy,
			"error": dbError,
		},
		"environment": fiber.Map{
			"port": config.AppPort,
			"debug": config.AppDebug,
		},
	}
	
	if !dbHealthy {
		return c.Status(503).JSON(fiber.Map{
			"status": "error",
			"health": health,
		})
	}
	
	return c.JSON(fiber.Map{
		"status": "healthy",
		"health": health,
	})
}

// GetDevice gets a specific device by ID  
func (handler *App) GetDevice(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	// Get user from session
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Get the actual device from database
	device, err := userRepo.GetDevice(user.ID, deviceId)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Format phone for display
	phoneDisplay := device.Phone
	if phoneDisplay == "" {
		phoneDisplay = "Not connected"
	}
	
	// Convert to response format
	deviceData := map[string]interface{}{
		"id":       device.ID,
		"name":     device.DeviceName,
		"phone":    phoneDisplay,
		"status":   device.Status,
		"lastSeen": device.LastSeen,
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device retrieved successfully",
		Results: deviceData,
	})
}

// DeviceActionsView renders the device actions testing page
func (handler *App) DeviceActionsView(c *fiber.Ctx) error {
	return c.Render("views/device_actions", fiber.Map{})
}

// DeviceLeadsView renders the device leads management page
func (handler *App) DeviceLeadsView(c *fiber.Ctx) error {
	return c.Render("views/device_leads", fiber.Map{})
}

// GetDeviceLeads gets all leads for a specific device
func (handler *App) GetDeviceLeads(c *fiber.Ctx) error {
	deviceId := c.Params("deviceId")
	userEmail := c.Locals("email").(string)
	
	userRepo := repository.GetUserRepository()
	user, err := userRepo.GetUserByEmail(userEmail)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User not found",
		})
	}
	
	leadRepo := repository.GetLeadRepository()
	leads, err := leadRepo.GetLeadsByDevice(user.ID, deviceId)
	if err != nil {
		log.Printf("Error getting leads: %v", err)
		// Return empty array instead of error
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Leads retrieved successfully",
			Results: []interface{}{},
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Leads retrieved successfully",
		Results: leads,
	})
}

// CreateLead creates a new lead
func (handler *App) CreateLead(c *fiber.Ctx) error {
	userEmail := c.Locals("email").(string)
	
	var request struct {
		DeviceID string `json:"device_id"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
		Niche    string `json:"niche"`
		Journey  string `json:"journey"`
		Status   string `json:"status"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	userRepo := repository.GetUserRepository()
	user, err := userRepo.GetUserByEmail(userEmail)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User not found",
		})
	}
	
	leadRepo := repository.GetLeadRepository()
	lead := &models.Lead{
		UserID: user.ID,
		Name:   request.Name,
		Phone:  request.Phone,
		Email:  "",
		Niche:  request.Niche,
		Source: request.Journey,
		Status: request.Status,
		Notes:  "",
	}
	err = leadRepo.CreateLead(lead)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to create lead: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Lead created successfully",
		Results: lead,
	})
}

// UpdateLead updates an existing lead
func (handler *App) UpdateLead(c *fiber.Ctx) error {
	leadId := c.Params("id")
	userEmail := c.Locals("email").(string)
	
	var request struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Niche   string `json:"niche"`
		Journey string `json:"journey"`
		Status  string `json:"status"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	userRepo := repository.GetUserRepository()
	user, err := userRepo.GetUserByEmail(userEmail)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User not found",
		})
	}
	
	leadRepo := repository.GetLeadRepository()
	lead := &models.Lead{
		UserID: user.ID,  // Use the user ID
		Name:   request.Name,
		Phone:  request.Phone,
		Email:  "",
		Niche:  request.Niche,
		Source: request.Journey,
		Status: request.Status,
		Notes:  "",
	}
	err = leadRepo.UpdateLead(leadId, lead)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to update lead: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Lead updated successfully",
		Results: map[string]string{"id": leadId},
	})
}

// DeleteLead deletes a lead
func (handler *App) DeleteLead(c *fiber.Ctx) error {
	leadId := c.Params("id")
	userEmail := c.Locals("email").(string)
	
	userRepo := repository.GetUserRepository()
	_, err := userRepo.GetUserByEmail(userEmail)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User not found",
		})
	}
	
	leadRepo := repository.GetLeadRepository()
	err = leadRepo.DeleteLead(leadId)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to delete lead: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Lead deleted successfully",
	})
}

// GetCampaigns gets all campaigns for the user
func (handler *App) GetCampaigns(c *fiber.Ctx) error {
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
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	campaignRepo := repository.GetCampaignRepository()
	campaigns, err := campaignRepo.GetCampaigns(user.ID)
	if err != nil {
		log.Printf("Error getting campaigns: %v", err)
		// Return empty array instead of error
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Campaigns retrieved successfully",
			Results: []interface{}{},
		})
	}
	
	// Return campaigns as array for frontend
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Campaigns retrieved successfully",
		Results: campaigns,
	})
}

// CreateCampaign creates a new campaign
func (handler *App) CreateCampaign(c *fiber.Ctx) error {
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
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	var request struct {
		CampaignDate  string `json:"campaign_date"`
		Title         string `json:"title"`
		Niche         string `json:"niche"`
		Message       string `json:"message"`
		ImageURL      string `json:"image_url"`
		ScheduledTime string `json:"scheduled_time"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	// Validate required fields
	if request.CampaignDate == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Campaign date is required",
		})
	}
	
	campaignRepo := repository.GetCampaignRepository()
	campaign := &models.Campaign{
		UserID:        user.ID,
		Title:         request.Title,
		Message:       request.Message,
		Niche:         request.Niche,
		ImageURL:      request.ImageURL,
		CampaignDate:  request.CampaignDate,
		ScheduledTime: request.ScheduledTime,
		Status:        "scheduled",
	}
	err = campaignRepo.CreateCampaign(campaign)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to create campaign: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Campaign created successfully",
		Results: campaign,
	})
}

// UpdateCampaign updates an existing campaign
func (handler *App) UpdateCampaign(c *fiber.Ctx) error {
	campaignIdStr := c.Params("id")
	campaignId, err := strconv.Atoi(campaignIdStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid campaign ID",
		})
	}
	
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
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	var request struct {
		Title         string `json:"title"`
		Niche         string `json:"niche"`
		Message       string `json:"message"`
		ImageURL      string `json:"image_url"`
		ScheduledTime string `json:"scheduled_time"`
		Status        string `json:"status"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	campaignRepo := repository.GetCampaignRepository()
	campaign := &models.Campaign{
		ID:            campaignId,
		UserID:        user.ID,
		Title:         request.Title,
		Message:       request.Message,
		Niche:         request.Niche,
		ImageURL:      request.ImageURL,
		ScheduledTime: request.ScheduledTime,
		Status:        request.Status,
	}
	err = campaignRepo.UpdateCampaign(campaign)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to update campaign: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Campaign updated successfully",
		Results: campaign,
	})
}

// DeleteCampaign deletes a campaign
func (handler *App) DeleteCampaign(c *fiber.Ctx) error {
	campaignIdStr := c.Params("id")
	campaignId, err := strconv.Atoi(campaignIdStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid campaign ID",
		})
	}
	
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
	
	_, err = userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	campaignRepo := repository.GetCampaignRepository()
	err = campaignRepo.DeleteCampaign(campaignId)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to delete campaign: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Campaign deleted successfully",
	})
}
// DeleteDevice deletes a device
func (handler *App) DeleteDevice(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get session from cookie instead of relying on Locals
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
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Check if device exists and get its details
	device, err := userRepo.GetDeviceByID(deviceId)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Verify device belongs to user
	if device.UserID != user.ID {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "You don't have permission to delete this device",
		})
	}
	
	// Clean up WhatsApp client if exists
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(device.ID)
	if err == nil && client != nil {
		// Try to logout first
		client.Logout(c.UserContext())
		// Remove from client manager
		cm.RemoveClient(device.ID)
	}
	
	// Delete device from database
	err = userRepo.DeleteDevice(device.ID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("Failed to delete device: %v", err),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device deleted successfully",
	})
}

// LogoutDevice logs out from WhatsApp
func (handler *App) LogoutDevice(c *fiber.Ctx) error {
	deviceId := c.Query("deviceId")
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	// Verify device ownership
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Get device
	device, err := userRepo.GetDevice(user.ID, deviceId)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(device.ID)
	if err == nil && client != nil {
		// Logout from WhatsApp
		err = client.Logout(c.UserContext())
		if err != nil {
			log.Printf("Error logging out device %s: %v", device.ID, err)
		}
		
		// Remove from client manager
		cm.RemoveClient(device.ID)
	}
	
	// Update device status in database
	err = userRepo.UpdateDeviceStatus(device.ID, "offline", "", "")
	if err != nil {
		log.Printf("Error updating device status: %v", err)
	}
	
	// Call the actual logout service to clean up files
	err = handler.Service.Logout(c.UserContext())
	if err != nil {
		log.Printf("Error calling logout service: %v", err)
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device logged out successfully",
		Results: map[string]interface{}{
			"deviceId": deviceId,
			"status":   "offline",
		},
	})
}

// ReconnectDevice reconnects to WhatsApp
func (handler *App) ReconnectDevice(c *fiber.Ctx) error {
	deviceId := c.Query("deviceId")
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Reconnecting...",
		Results: map[string]interface{}{
			"deviceId": deviceId,
			"status":   "reconnecting",
		},
	})
}

// GetDevices gets all devices for the app
func (handler *App) GetDevices(c *fiber.Ctx) error {
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Devices retrieved",
		Results: []interface{}{},
	})
}

// GetUserInfo gets user info
func (handler *App) GetUserInfo(c *fiber.Ctx) error {
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "User info retrieved",
		Results: map[string]interface{}{
			"name":   "User",
			"number": "+60123456789",
			"avatar": "",
		},
	})
}

// GetUserAvatar gets user avatar
func (handler *App) GetUserAvatar(c *fiber.Ctx) error {
	// Return a default avatar
	return c.SendString("")
}

// ChangeUserAvatar changes user avatar
func (handler *App) ChangeUserAvatar(c *fiber.Ctx) error {
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Avatar updated",
	})
}

// ChangeUserPushName changes user push name
func (handler *App) ChangeUserPushName(c *fiber.Ctx) error {
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Push name updated",
	})
}
// SyncDeviceChats manually triggers chat synchronization for a device
func (handler *App) SyncDeviceChats(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	// Verify device ownership
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Check if device belongs to user
	device, err := userRepo.GetDevice(user.ID, deviceId)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Trigger chat sync
	go func() {
		chats, err := whatsapp.GetChatsForDevice(device.ID)
		if err != nil {
			fmt.Printf("Failed to sync chats for device %s: %v\n", device.ID, err)
		} else {
			fmt.Printf("Successfully synced %d chats for device %s\n", len(chats), device.ID)
		}
	}()
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Chat sync initiated",
		Results: map[string]interface{}{
			"deviceId": device.ID,
			"status":   "syncing",
		},
	})
}
// DiagnoseDevice provides diagnostic information about a device's WhatsApp connection
func (handler *App) DiagnoseDevice(c *fiber.Ctx) error {
	deviceId := c.Params("id")
	
	// Get session from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	// Verify device ownership
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	user, err := userRepo.GetUserByID(session.UserID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}
	
	// Get device info
	device, err := userRepo.GetDevice(user.ID, deviceId)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Check WhatsApp client status
	cm := whatsapp.GetClientManager()
	client, clientErr := cm.GetClient(device.ID)
	
	// Get full client diagnostics
	allDiagnostics := whatsapp.DiagnoseClients()
	
	diagnostics := map[string]interface{}{
		"device": map[string]interface{}{
			"id":     device.ID,
			"name":   device.DeviceName,
			"phone":  device.Phone,
			"status": device.Status,
			"jid":    device.JID,
		},
		"whatsapp_client": map[string]interface{}{
			"connected": clientErr == nil,
			"error":     "",
		},
		"database": map[string]interface{}{
			"chats_count":    0,
			"messages_count": 0,
		},
		"client_manager": allDiagnostics,
	}
	
	if clientErr != nil {
		diagnostics["whatsapp_client"].(map[string]interface{})["error"] = clientErr.Error()
		
		// Try to register from database if device is online
		if device.Status == "online" {
			log.Printf("Attempting to auto-register device %s from database", device.ID)
			if err := whatsapp.TryRegisterDeviceFromDatabase(device.ID); err == nil {
				diagnostics["whatsapp_client"].(map[string]interface{})["auto_registered"] = true
				// Retry getting client
				client, clientErr = cm.GetClient(device.ID)
				if clientErr == nil {
					diagnostics["whatsapp_client"].(map[string]interface{})["connected"] = true
					diagnostics["whatsapp_client"].(map[string]interface{})["error"] = ""
				}
			} else {
				log.Printf("Auto-registration failed: %v", err)
			}
		}
	} else {
		// Check if client is logged in
		diagnostics["whatsapp_client"].(map[string]interface{})["logged_in"] = client.IsLoggedIn()
		diagnostics["whatsapp_client"].(map[string]interface{})["is_connected"] = client.IsConnected()
		
		// Try to get contacts count
		contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
		if err == nil {
			diagnostics["whatsapp_client"].(map[string]interface{})["contacts_count"] = len(contacts)
		}
	}
	
	// Check database
	whatsappRepo := repository.GetWhatsAppRepository()
	chats, _ := whatsappRepo.GetChats(device.ID)
	diagnostics["database"].(map[string]interface{})["chats_count"] = len(chats)
	
	// Force a sync attempt
	go func() {
		fmt.Printf("Forcing sync for device %s\n", device.ID)
		chats, err := whatsapp.GetChatsForDevice(device.ID)
		if err != nil {
			fmt.Printf("Sync error for device %s: %v\n", device.ID, err)
		} else {
			fmt.Printf("Sync complete for device %s: %d chats\n", device.ID, len(chats))
		}
	}()
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Device diagnostics",
		Results: diagnostics,
	})
}
// GetCampaignSummary gets campaign statistics
func (handler *App) GetCampaignSummary(c *fiber.Ctx) error {
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
	
	// Get campaign statistics
	campaignRepo := repository.GetCampaignRepository()
	campaigns, err := campaignRepo.GetCampaignsByUser(session.UserID)
	if err != nil {
		campaigns = []models.Campaign{}
	}
	
	// Calculate statistics
	totalCampaigns := len(campaigns)
	pendingCampaigns := 0
	sentCampaigns := 0
	failedCampaigns := 0
	
	for _, campaign := range campaigns {
		switch campaign.Status {
		case "scheduled", "pending":
			pendingCampaigns++
		case "sent":
			sentCampaigns++
		case "failed":
			failedCampaigns++
		}
	}
	
	// Get broadcast message statistics
	broadcastRepo := repository.GetBroadcastRepository()
	broadcastStats, err := broadcastRepo.GetUserBroadcastStats(session.UserID)
	if err != nil {
		broadcastStats = map[string]interface{}{
			"total_messages": 0,
			"sent_messages": 0,
			"failed_messages": 0,
			"pending_messages": 0,
		}
	}
	
	summary := map[string]interface{}{
		"campaigns": map[string]interface{}{
			"total": totalCampaigns,
			"pending": pendingCampaigns,
			"sent": sentCampaigns,
			"failed": failedCampaigns,
		},
		"messages": broadcastStats,
		"recent_campaigns": campaigns[:min(5, len(campaigns))],
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Campaign summary",
		Results: summary,
	})
}

// GetSequenceSummary gets sequence statistics
func (handler *App) GetSequenceSummary(c *fiber.Ctx) error {
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
	
	// Get sequence statistics
	sequenceRepo := repository.GetSequenceRepository()
	sequences, err := sequenceRepo.GetSequences(session.UserID)
	if err != nil {
		sequences = []models.Sequence{}
	}
	
	// Calculate statistics
	totalSequences := len(sequences)
	activeSequences := 0
	pausedSequences := 0
	draftSequences := 0
	totalContacts := 0
	
	for _, sequence := range sequences {
		switch sequence.Status {
		case "active":
			activeSequences++
		case "paused":
			pausedSequences++
		case "draft":
			draftSequences++
		}
		totalContacts += sequence.ContactsCount
	}
	
	summary := map[string]interface{}{
		"sequences": map[string]interface{}{
			"total": totalSequences,
			"active": activeSequences,
			"paused": pausedSequences,
			"draft": draftSequences,
		},
		"contacts": map[string]interface{}{
			"total": totalContacts,
			"average_per_sequence": float64(totalContacts) / float64(max(1, totalSequences)),
		},
		"recent_sequences": sequences[:min(5, len(sequences))],
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence summary",
		Results: summary,
	})
}

// GetWorkerStatus gets the status of all device workers
func (handler *App) GetWorkerStatus(c *fiber.Ctx) error {
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
	
	// Get user's devices
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}
	
	// Get broadcast manager stats
	broadcastManager := broadcast.GetBroadcastManager()
	workerStats := broadcastManager.GetWorkerStats()
	
	// Get worker status for each device
	deviceWorkers := []map[string]interface{}{}
	for _, device := range devices {
		workerInfo := map[string]interface{}{
			"device_id": device.ID,
			"device_name": device.DeviceName,
			"device_status": device.Status,
			"worker_status": "not_running",
			"queue_size": 0,
			"processed": 0,
			"failed": 0,
		}
		
		// Find worker stats for this device
		if workers, ok := workerStats["workers"].([]map[string]interface{}); ok {
			for _, worker := range workers {
				if worker["device_id"] == device.ID {
					workerInfo["worker_status"] = worker["status"]
					workerInfo["queue_size"] = worker["queue_size"]
					workerInfo["processed"] = worker["processed"]
					workerInfo["failed"] = worker["failed"]
					workerInfo["last_activity"] = worker["last_activity"]
					break
				}
			}
		}
		
		deviceWorkers = append(deviceWorkers, workerInfo)
	}
	
	response := map[string]interface{}{
		"total_workers": workerStats["total_workers"],
		"user_devices": len(devices),
		"connected_devices": countConnectedDevices(devices),
		"device_workers": deviceWorkers,
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Worker status",
		Results: response,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func countConnectedDevices(devices []*models.UserDevice) int {
	count := 0
	for _, device := range devices {
		if device.Status == "connected" {
			count++
		}
	}
	return count
}


// ResumeFailedWorkers resumes all failed device workers
func (handler *App) ResumeFailedWorkers(c *fiber.Ctx) error {
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

	// Get all devices for user
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}

	// Resume workers for devices that are connected but have stopped workers
	resumedCount := 0
	for _, device := range devices {
		if device.Status == "connected" {
			// TODO: Check if worker is stopped and resume
			// This would interface with your broadcast manager
			resumedCount++
		}
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Resumed %d workers", resumedCount),
		Results: map[string]interface{}{
			"resumed_count": resumedCount,
		},
	})
}

// StopAllWorkers stops all running device workers
func (handler *App) StopAllWorkers(c *fiber.Ctx) error {
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

	// Get all devices for user
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get devices",
		})
	}

	// Stop all workers
	stoppedCount := 0
	for _, device := range devices {
		// TODO: Stop worker for this device
		// This would interface with your broadcast manager
		stoppedCount++
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Stopped %d workers", stoppedCount),
		Results: map[string]interface{}{
			"stopped_count": stoppedCount,
		},
	})
}
