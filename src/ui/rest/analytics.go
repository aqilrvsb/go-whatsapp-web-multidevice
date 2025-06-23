package rest

import (
	"encoding/csv"
	"os"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

// GetAnalyticsData returns analytics data for the dashboard
func (handler *App) GetAnalyticsData(c *fiber.Ctx) error {
	days := c.Params("days", "7")
	deviceFilter := c.Query("device", "all")
	
	// Get user email from session/auth (for now using a placeholder)
	userEmail := c.Get("X-User-Email", "admin@whatsapp.com")
	
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
	
	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -daysInt)
	
	// Get real analytics from message records
	analytics := utils.GetUserAnalytics(userEmail, startDate, endDate, deviceFilter)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Analytics data retrieved",
		Results: analytics,
	})
}

// GetCustomAnalyticsData returns analytics for custom date range
func (handler *App) GetCustomAnalyticsData(c *fiber.Ctx) error {
	startStr := c.Query("start")
	endStr := c.Query("end")
	deviceFilter := c.Query("device", "all")
	
	// Get user email from session/auth
	userEmail := c.Get("X-User-Email", "admin@whatsapp.com")
	
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
	
	// Get real analytics from message records
	analytics := utils.GetUserAnalytics(userEmail, startDate, endDate, deviceFilter)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Custom analytics data retrieved",
		Results: analytics,
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
	// Lead metrics
	leadsSent := 0
	leadsReceived := 0
	leadsRead := 0
	leadsReplied := 0
	uniqueChats := make(map[string]bool)
	
	// Initialize daily map
	dailyMap := make(map[string]fiber.Map)
	for d := startDate; d.Before(endDate.Add(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		dailyMap[dateStr] = fiber.Map{
			"date":     d.Format("Jan 2"),
			"sent":     0,
			"received": 0,
			"read":     0,
			"replied":  0,
		}
	}
	
	// Read chat storage file
	file, err := os.Open(config.PathChatStorage)
	if err != nil {
		// Return empty data if file doesn't exist
		metrics := fiber.Map{
			"activeDevices":     0,
			"inactiveDevices":   0,
			"leadsSent":         0,
			"leadsReceived":     0,
			"leadsNotReceived":  0,
			"leadsRead":         0,
			"leadsNotRead":      0,
			"leadsReplied":      0,
		}
		return metrics, convertDailyMapToSlice(dailyMap, startDate, endDate)
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
			
			// Simulate lead metrics (in real implementation, parse actual message data)
			if strings.Contains(jid, "@s.whatsapp.net") || strings.Contains(jid, "@g.us") {
				leadsReceived++
				
				// Simulate read status (70% of received are read)
				if leadsReceived%10 < 7 {
					leadsRead++
				}
				
				// Simulate reply status (50% of read are replied)
				if leadsRead%10 < 5 {
					leadsReplied++
				}
				
				// Add to daily count
				today := time.Now().Format("2006-01-02")
				if daily, exists := dailyMap[today]; exists {
					received := daily["received"].(int)
					read := daily["read"].(int)
					replied := daily["replied"].(int)
					
					daily["received"] = received + 1
					if leadsReceived%10 < 7 {
						daily["read"] = read + 1
					}
					if leadsRead%10 < 5 {
						daily["replied"] = replied + 1
					}
					dailyMap[today] = daily
				}
			}
		}
	}
	
	// Calculate derived metrics
	leadsSent = len(records) // Assume all records are sent messages for now
	leadsNotReceived := leadsSent - leadsReceived
	leadsNotRead := leadsReceived - leadsRead
	
	metrics := fiber.Map{
		"activeDevices":     1, // Will be updated from device list
		"inactiveDevices":   0,
		"leadsSent":         leadsSent,
		"leadsReceived":     leadsReceived,
		"leadsNotReceived":  leadsNotReceived,
		"leadsRead":         leadsRead,
		"leadsNotRead":      leadsNotRead,
		"leadsReplied":      leadsReplied,
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
	
	// Get devices from the service
	devicesList, err := handler.Service.FetchDevices(c.UserContext())
	if err == nil && len(devicesList) > 0 {
		for i, device := range devicesList {
			devices = append(devices, fiber.Map{
				"id":       i + 1,
				"name":     device.Name,
				"phone":    device.Device,
				"status":   "online",
				"lastSeen": "Active now",
				"pushName": device.Name,
			})
		}
	} else {
		// Try to get first device
		firstDevice, err := handler.Service.FirstDevice(c.UserContext())
		if err == nil {
			devices = append(devices, fiber.Map{
				"id":       1,
				"name":     firstDevice.Name,
				"phone":    firstDevice.Device,
				"status":   "online",
				"lastSeen": "Active now",
				"pushName": firstDevice.Name,
			})
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Devices retrieved",
		Results: devices,
	})
}