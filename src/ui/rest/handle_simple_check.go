package rest

import (
	"github.com/gofiber/fiber/v2"
)

// SimpleCheckConnection - Quick fix for the 404 error
func SimpleCheckConnection(c *fiber.Ctx) error {
	// For now, just return a simple success response
	// This proves the endpoint is working
	return c.JSON(fiber.Map{
		"code":    200,
		"message": "Device check endpoint is working",
		"data": []map[string]interface{}{
			{
				"note": "Full device check will be available after complete deployment",
			},
		},
	})
}
