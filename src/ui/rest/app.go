package rest

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
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