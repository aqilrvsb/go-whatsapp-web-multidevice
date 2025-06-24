package middleware

import (
	"fmt"
	"strings"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// PublicRoutes that don't require authentication
var PublicRoutes = []string{
	"/login",
	"/register",
	"/logout", 
	"/api/login",
	"/api/register",
	"/health",
	"/api/health",
	"/statics",
	"/assets",
	"/components",
	"/app",  // Allow all /app endpoints for WhatsApp functionality
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
		// Check for session cookie first
		token := c.Cookies("session_token")
		
		// If no cookie, check headers (for API compatibility)
		if token == "" {
			token = c.Get("Authorization")
			if token == "" {
				token = c.Get("X-Auth-Token")
			}
		}
		
		// Debug logging
		fmt.Printf("Auth Debug - Path: %s, Token: %s\n", path, token)
		
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
			fmt.Printf("Session validation error: %v\n", err)
			if strings.HasPrefix(path, "/api/") {
				return c.Status(401).JSON(fiber.Map{
					"status": 401,
					"code": "UNAUTHORIZED", 
					"message": "Invalid session",
				})
			}
			return c.Redirect("/login")
		}
		
		// Debug - session found
		fmt.Printf("Session found for user: %s\n", session.UserID)
		
		// Set user context
		c.Locals("userID", session.UserID)
		c.Locals("session", session)
		
		return c.Next()
	}
}