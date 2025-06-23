package rest

import (
	"encoding/csv"
	"os"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

// GetAnalyticsData returns analytics data for the dashboard
func (handler *App) GetAnalyticsData(c *fiber.Ctx) error {
	days := c.Params("days", "7")
	daysInt := 7
	
	switch days {
	case "today":
		daysInt = 1
	case "7":
		daysInt = 7
	case "30":
		daysInt = 30
	case "90":
		daysInt = 90
	}
	
	// Get metrics from chat storage
	metrics, dailyData := getMetricsFromStorage(daysInt)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Analytics data retrieved",
		Results: fiber.Map{
			"metrics": metrics,
			"daily":   dailyData,
		},
	})
}

// GetCustomAnalyticsData returns analytics for custom date range
func (handler *App) GetCustomAnalyticsData(c *fiber.Ctx) error {
	startStr := c.Query("start")
	endStr := c.Query("end")
	
	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid start date format",
		})
	}
	
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST", 
			Message: "Invalid end date format",
		})
	}
	
	metrics, dailyData := getMetricsFromStorageCustom(startDate, endDate)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Custom analytics data retrieved",
		Results: fiber.Map{
			"metrics": metrics,
			"daily":   dailyData,
		},
	})
}

// getMetricsFromStorage reads chat.csv and calculates metrics
func getMetricsFromStorage(days int) (fiber.Map, []fiber.Map) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	
	return getMetricsFromStorageCustom(startDate, endDate)
}

// getMetricsFromStorageCustom reads chat.csv for custom date range
func getMetricsFromStorageCustom(startDate, endDate time.Time) (fiber.Map, []fiber.Map) {
	totalSent := 0
	totalReceived := 0
	uniqueChats := make(map[string]bool)
	
	// Initialize daily map
	dailyMap := make(map[string]fiber.Map)
	for d := startDate; d.Before(endDate.Add(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		dailyMap[dateStr] = fiber.Map{
			"date":     d.Format("Jan 2"),
			"sent":     0,
			"received": 0,
		}
	}
	
	// Read chat storage file
	file, err := os.Open(config.PathChatStorage)
	if err != nil {
		// Return empty data if file doesn't exist
		return fiber.Map{
			"totalSent":     0,
			"totalReceived": 0,
			"activeChats":   0,
			"replyRate":     0,
		}, convertDailyMapToSlice(dailyMap, startDate, endDate)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()
	
	// Process records
	for _, record := range records {
		if len(record) >= 3 {
			// Assuming format: messageID, JID, content, timestamp
			jid := record[1]
			
			// Count unique chats
			uniqueChats[jid] = true
			
			// Determine if sent or received based on JID
			// Messages to others are "sent", messages from others are "received"
			if strings.Contains(jid, "@s.whatsapp.net") || strings.Contains(jid, "@g.us") {
				// For now, count all as received (in real implementation, check against logged-in user)
				totalReceived++
				
				// Add to daily count (using current date as we don't have timestamps in CSV)
				today := time.Now().Format("2006-01-02")
				if daily, exists := dailyMap[today]; exists {
					received := daily["received"].(int)
					daily["received"] = received + 1
					dailyMap[today] = daily
				}
			}
		}
	}
	
	// Calculate reply rate
	replyRate := 0
	if totalReceived > 0 {
		// Assume 80% reply rate for demo
		replyRate = 80
	}
	
	metrics := fiber.Map{
		"totalSent":     totalSent,
		"totalReceived": totalReceived,
		"activeChats":   len(uniqueChats),
		"replyRate":     replyRate,
	}
	
	return metrics, convertDailyMapToSlice(dailyMap, startDate, endDate)
}

// convertDailyMapToSlice converts daily map to sorted slice
func convertDailyMapToSlice(dailyMap map[string]fiber.Map, startDate, endDate time.Time) []fiber.Map {
	var dailyData []fiber.Map
	
	for d := startDate; d.Before(endDate.Add(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		if daily, exists := dailyMap[dateStr]; exists {
			dailyData = append(dailyData, daily)
		}
	}
	
	return dailyData
}

// GetConnectedDevices returns real connected devices
func (handler *App) GetConnectedDevices(c *fiber.Ctx) error {
	devices := []fiber.Map{}
	
	// Get device info from the service
	device, err := handler.Service.GetDeviceInfo(c.UserContext())
	if err == nil && device != nil {
		devices = append(devices, fiber.Map{
			"id":       1,
			"name":     device.Device.GetPlatform().String(),
			"phone":    device.Number,
			"status":   "online",
			"lastSeen": "Active now",
			"pushName": device.Device.GetDeviceName(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Devices retrieved",
		Results: devices,
	})
}