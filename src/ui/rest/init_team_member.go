package rest

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
)

// InitRestTeamMember initializes team member routes
func InitRestTeamMember(app *fiber.App, db *sql.DB) {
	// Create repository and handlers
	repo := repository.NewTeamMemberRepository(db)
	handlers := NewTeamMemberHandlers(repo)
	
	// Public route for team login
	app.Post("/team-login", handlers.LoginTeamMember)
	
	// Team login page
	app.Get("/team-login", func(c *fiber.Ctx) error {
		return c.Render("views/team_login", fiber.Map{
			"Title": "Team Member Login",
		})
	})
	
	// Team dashboard (protected by middleware)
	app.Get("/team-dashboard", handlers.TeamMemberAuthMiddleware, func(c *fiber.Ctx) error {
		return c.Render("views/team_dashboard", fiber.Map{
			"Title": "Team Dashboard",
		})
	})
	
	// Team member API routes (protected)
	teamAPI := app.Group("/api", handlers.TeamMemberAuthMiddleware)
	teamAPI.Get("/team-member/info", handlers.GetTeamMemberInfo)
	
	// Protected routes (admin only)
	api := app.Group("/api")
	
	// Team member management (admin only)
	api.Get("/team-members", handlers.GetAllTeamMembers)
	api.Post("/team-members", handlers.CreateTeamMember)
	api.Put("/team-members/:id", handlers.UpdateTeamMember)
	api.Delete("/team-members/:id", handlers.DeleteTeamMember)
	
	// Team member logout
	api.Post("/team-logout", handlers.LogoutTeamMember)
}
