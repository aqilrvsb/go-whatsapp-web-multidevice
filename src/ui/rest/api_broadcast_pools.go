package rest

import (
	"fmt"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
)

// GetBroadcastPoolStatus gets status of all broadcast pools
func (handler *App) GetBroadcastPoolStatus(c *fiber.Ctx) error {
	// Get session
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
	
	// Get all active broadcasts for user
	db := database.GetDB()
	
	// Get active campaigns
	campaignRows, err := db.Query(`
		SELECT id, title, status, 
		       (SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = c.id) as total_messages,
		       (SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = c.id AND status = 'sent') as sent_messages
		FROM campaigns c
		WHERE user_id = $1 
		AND status IN ('triggered', 'processing')
		ORDER BY created_at DESC
	`, session.UserID)
	
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get campaigns",
		})
	}
	defer campaignRows.Close()
	
	pools := []map[string]interface{}{}
	broadcastManager := broadcast.GetUltraScaleBroadcastManager()
	
	for campaignRows.Next() {
		var id int
		var title, status string
		var totalMessages, sentMessages int
		
		err := campaignRows.Scan(&id, &title, &status, &totalMessages, &sentMessages)
		if err != nil {
			continue
		}
		
		// Get pool status
		poolStatus := broadcastManager.GetPoolStatus("campaign", fmt.Sprintf("%d", id))
		poolStatus["title"] = title
		poolStatus["db_total"] = totalMessages
		poolStatus["db_sent"] = sentMessages
		
		pools = append(pools, poolStatus)
	}
	
	// TODO: Add sequence pools
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Broadcast pool status",
		Results: map[string]interface{}{
			"pools": pools,
			"total": len(pools),
		},
	})
}
