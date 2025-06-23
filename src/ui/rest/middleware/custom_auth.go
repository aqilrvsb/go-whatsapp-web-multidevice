package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"strings"
)

// PublicRoutes that don't require authentication
var PublicRoutes = []string{
	"/login",
	"/register", 
	"/api/login",
	"/api/register",
	"/health",
	"/api/health",
	"/statics",
	"/assets",
	"/components",
}

// CustomAuth middleware for session-based authentication
func CustomAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if route is public
		path := c.Path()
		for _, publicRoute := range PublicRoutes {
			if strings.HasPrefix(path, publicRoute) {
				return c.Next()
			}
		}
		
		// Check session token
		token := c.Get("Authorization")
		if token == "" {
			token = c.Get("X-Auth-Token")
		}
		
		if token == "" {
			// No token, redirect to login
			if strings.HasPrefix(path, "/api/") {
				return c.Status(401).JSON(fiber.Map{
					"status": 401,
					"code": "UNAUTHORIZED",
					"message": "Authentication required",
				})
			}
			return c.Redirect("/login")
		}
		
		// Validate token
		userRepo := repository.GetUserRepository()
		session, err := userRepo.GetSession(strings.TrimPrefix(token, "Bearer "))
		if err != nil {
			if strings.HasPrefix(path, "/api/") {
				return c.Status(401).JSON(fiber.Map{
					"status": 401,
					"code": "UNAUTHORIZED", 
					"message": "Invalid or expired session",
				})
			}
			return c.Redirect("/login")
		}
		
		// Set user context
		c.Locals("userID", session.UserID)
		c.Locals("session", session)
		
		return c.Next()
	}
}