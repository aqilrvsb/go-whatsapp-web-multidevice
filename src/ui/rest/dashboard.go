package rest

import (
	"embed"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

// Dashboard routes
func InitDashboardRoutes(app *fiber.App, embedViews embed.FS) {
	// Create template engine
	engine := html.NewFileSystem(embedViews, ".html")
	app = fiber.New(fiber.Config{
		Views: engine,
		ViewsLayout: "",
	})

	// Public routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Render("views/login", fiber.Map{
			"Title": "Login - WhatsApp Analytics",
		})
	})

	// Protected routes (require authentication)
	protected := app.Group("/", BasicAuth())
	
	protected.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.Render("views/dashboard", fiber.Map{
			"Title": "Dashboard - WhatsApp Analytics",
			"User": c.Locals("username"),
		})
	})

	// API Routes for dashboard
	api := app.Group("/api", BasicAuth())
	
	// Analytics endpoints
	api.Get("/analytics/:days", getAnalytics)
	api.Get("/devices", getDevices)
	api.Post("/devices", addDevice)
	api.Delete("/devices/:id", deleteDevice)
	api.Get("/devices/:id/qr", getDeviceQR)
	api.Post("/devices/:id/logout", logoutDevice)
	
	// Legacy WhatsApp routes (for backward compatibility)
	app.Get("/app/login", AppLoginView)
	app.Get("/app/devices", AppDevicesView)
}

// Analytics handlers
func getAnalytics(c *fiber.Ctx) error {
	days := c.Params("days", "7")
	
	// Mock analytics data
	analytics := fiber.Map{
		"metrics": fiber.Map{
			"totalSent": 1234,
			"totalReceived": 987,
			"activeChats": 42,
			"replyRate": 85,
		},
		"daily": []fiber.Map{
			// Daily data would be generated here
		},
	}
	
	return c.JSON(analytics)
}

func getDevices(c *fiber.Ctx) error {
	// Mock device data
	devices := []fiber.Map{
		{
			"id": 1,
			"name": "iPhone 12 Pro",
			"phone": "+62 812-3456-7890",
			"status": "online",
			"lastSeen": "Active now",
		},
	}
	
	return c.JSON(devices)
}

func addDevice(c *fiber.Ctx) error {
	var device struct {
		Name string `json:"name"`
	}
	
	if err := c.BodyParser(&device); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	
	// Add device logic here
	
	return c.JSON(fiber.Map{
		"id": 2,
		"name": device.Name,
		"status": "offline",
	})
}

func deleteDevice(c *fiber.Ctx) error {
	deviceID := c.Params("id")
	
	// Delete device logic here
	
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Device %s deleted", deviceID),
	})
}

func getDeviceQR(c *fiber.Ctx) error {
	deviceID := c.Params("id")
	
	// Generate QR code for device
	// This would integrate with the existing WhatsApp QR generation
	
	return c.Redirect(fmt.Sprintf("/app/qr/%s", deviceID))
}

func logoutDevice(c *fiber.Ctx) error {
	deviceID := c.Params("id")
	
	// Logout device logic here
	
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Device %s logged out", deviceID),
	})
}

// Basic Auth Middleware
func BasicAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get auth credentials from config
		validUsers := map[string]string{
			"admin": "changeme123",
		}
		
		// Check basic auth
		username, password, ok := c.Request().BasicAuth()
		if !ok {
			c.Set("WWW-Authenticate", `Basic realm="Restricted"`)
			return c.SendStatus(401)
		}
		
		// Validate credentials
		if validPassword, exists := validUsers[username]; !exists || validPassword != password {
			c.Set("WWW-Authenticate", `Basic realm="Restricted"`)
			return c.SendStatus(401)
		}
		
		// Store username in context
		c.Locals("username", username)
		
		return c.Next()
	}
}