package middleware

import (
	"fmt"
	"strings"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// PublicRoutes that don't require authentication
var PublicRoutes = []string{
	"/",
	"/login",
	"/register",
	"/logout",
	"/dashboard",         // Allow dashboard access
	"/health",
	"/statics",
	"/assets",
	"/components",
	"/favicon.ico",
	"/robots.txt",
}

// CustomAuth middleware for session-based authentication
func CustomAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if route is public
		path := c.Path()
		
		// Check exact matches and prefix matches for truly public routes
		for _, publicRoute := range PublicRoutes {
			if path == publicRoute || strings.HasPrefix(path, publicRoute + "/") {
				return c.Next()
			}
		}
		
		// Allow all OPTIONS requests (for CORS)
		if c.Method() == "OPTIONS" {
			return c.Next()
		}
		
		// For API and APP routes, we need to check authentication
		// Check session token from cookie
		token := c.Cookies("session_token")
		
		// If no cookie, check headers (for API compatibility)
		if token == "" {
			authHeader := c.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		
		// Also check X-Auth-Token header
		if token == "" {
			token = c.Get("X-Auth-Token")
		}
		
		// Debug logging (remove in production)
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/app/") {
			fmt.Printf("API Auth Debug - Path: %s, Token: %s, Method: %s, Cookie: %s\n", 
				path, token, c.Method(), c.Cookies("session_token"))
		}
		
		// If no token found
		if token == "" {
			// For API routes, return JSON error
			if strings.HasPrefix(path, "/api/") {
				return c.Status(401).JSON(fiber.Map{
					"status": 401,
					"code": "UNAUTHORIZED",
					"message": "Authentication required - no token provided",
				})
			}
			// For web routes, redirect to login
			return c.Redirect("/login")
		}
		
		// Validate token in database
		userRepo := repository.GetUserRepository()
		session, err := userRepo.GetSession(token)
		
		if err != nil {
			fmt.Printf("Session validation error for token %s: %v\n", token, err)
			
			// For API routes, return JSON error
			if strings.HasPrefix(path, "/api/") {
				return c.Status(401).JSON(fiber.Map{
					"status": 401,
					"code": "UNAUTHORIZED", 
					"message": "Invalid session - token not found or expired",
				})
			}
			// For web routes, redirect to login
			return c.Redirect("/login")
		}
		
		// Session is valid - set user context
		fmt.Printf("Session validated for user: %s on path: %s\n", session.UserID, path)
		
		// Store user info in context for use in handlers
		c.Locals("userID", session.UserID)
		c.Locals("userEmail", session.UserID) // Assuming userID is email
		c.Locals("email", session.UserID) // Add this for backward compatibility
		c.Locals("session", session)
		
		return c.Next()
	}
}

// GetUserFromContext extracts user information from context
func GetUserFromContext(c *fiber.Ctx) (userID string, ok bool) {
	userIDVal := c.Locals("userID")
	if userIDVal == nil {
		return "", false
	}
	userID, ok = userIDVal.(string)
	return userID, ok
}
