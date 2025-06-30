package usecase

import (
	"time"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/sirupsen/logrus"
)

// StartCampaignStatusMonitor monitors campaign progress and updates status
func StartCampaignStatusMonitor() {
	logrus.Info("Starting campaign status monitor...")
	
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			updateCampaignStatuses()
		}
	}
}

func updateCampaignStatuses() {
	db := database.GetDB()
	
	// Find campaigns that are triggered or processing
	rows, err := db.Query(`
		SELECT DISTINCT c.id, c.title, c.status
		FROM campaigns c
		WHERE c.status IN ('triggered', 'processing')
		AND EXISTS (
			SELECT 1 FROM broadcast_messages bm 
			WHERE bm.campaign_id = c.id
		)
	`)
	if err != nil {
		logrus.Errorf("Failed to get campaigns for status update: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var campaignID int
		var title, currentStatus string
		if err := rows.Scan(&campaignID, &title, &currentStatus); err != nil {
			continue
		}
		
		// Get message statistics
		var total, pending, queued, sent, failed, skipped int
		err := db.QueryRow(`
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
				COUNT(CASE WHEN status = 'queued' THEN 1 END) as queued,
				COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
				COUNT(CASE WHEN status = 'skipped' THEN 1 END) as skipped
			FROM broadcast_messages
			WHERE campaign_id = $1
		`, campaignID).Scan(&total, &pending, &queued, &sent, &failed, &skipped)
		
		if err != nil {
			logrus.Errorf("Failed to get message stats for campaign %d: %v", campaignID, err)
			continue
		}
		
		// Get device count for this campaign
		var deviceCount int
		err = db.QueryRow(`
			SELECT COUNT(DISTINCT device_id) 
			FROM broadcast_messages 
			WHERE campaign_id = $1
		`, campaignID).Scan(&deviceCount)
		
		if err != nil {
			deviceCount = 1 // Default to 1 if error
		}
		
		// Determine new status
		var newStatus string
		
		if total == 0 {
			// No messages, keep current status
			continue
		} else if currentStatus == "triggered" && (sent > 0 || failed > 0) {
			// First worker has started processing (at least one message sent/failed)
			newStatus = "processing"
		} else if pending == 0 && queued == 0 {
			// All messages have been processed (no more pending or queued)
			if sent > 0 {
				newStatus = "finished"
			} else if failed == total || skipped == total {
				newStatus = "failed"
			} else {
				newStatus = "finished"
			}
		} else {
			// Still in progress, no status change needed
			continue
		}
		
		// Update status if changed
		if newStatus != "" && newStatus != currentStatus {
			_, err = db.Exec(`
				UPDATE campaigns 
				SET status = $1, updated_at = CURRENT_TIMESTAMP 
				WHERE id = $2
			`, newStatus, campaignID)
			
			if err != nil {
				logrus.Errorf("Failed to update campaign %d status to %s: %v", 
					campaignID, newStatus, err)
			} else {
				// Calculate progress percentage
				processed := sent + failed + skipped
				progress := 0
				if total > 0 {
					progress = (processed * 100) / total
				}
				
				logrus.Infof("Campaign '%s' (ID: %d) status: %s → %s | Progress: %d%% (%d/%d) | Devices: %d | Sent: %d, Failed: %d, Skipped: %d", 
					title, campaignID, currentStatus, newStatus, progress, processed, total, deviceCount, sent, failed, skipped)
			}
		}
	}
}
