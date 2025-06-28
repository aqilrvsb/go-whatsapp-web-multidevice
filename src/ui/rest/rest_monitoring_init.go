package rest

import (
	"github.com/gofiber/fiber/v2"
)

// InitRestMonitoring initializes monitoring endpoints
func InitRestMonitoring(app *fiber.App) {
	// Redis monitoring endpoints
	app.Get("/api/monitoring/redis", GetRedisMetrics)
	app.Get("/api/monitoring/queue/:queue", GetQueueMessages)
	app.Delete("/api/monitoring/queue/:queue", ClearQueue)
	app.Post("/api/monitoring/expire-messages", ExpireOldMessages)
	
	// Dashboard page for Redis monitoring
	app.Get("/monitoring/redis", func(c *fiber.Ctx) error {
		return c.Render("views/monitoring/redis", fiber.Map{
			"Title": "Redis Queue Monitoring",
		})
	})
}
