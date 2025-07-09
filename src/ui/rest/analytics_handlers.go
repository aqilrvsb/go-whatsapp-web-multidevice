package rest

import (
	"time"
	"database/sql"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

// GetCampaignAnalytics returns campaign analytics data
func (handler *App) GetCampaignAnalytics(c *fiber.Ctx) error {
	// Get query parameters
	startDate := c.Query("start")
	endDate := c.Query("end")
	deviceFilter := c.Query("device", "all")
	nicheFilter := c.Query("niche", "all")
	
	// Validate dates
	if startDate == "" || endDate == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Start date and end date are required",
		})
	}
	
	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid start date format",
		})
	}
	
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid end date format",
		})
	}
	
	// Add 23:59:59 to end date to include the whole day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	
	// Get campaign statistics
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to connect to database",
		})
	}
	defer db.Close()
	
	// Total campaigns
	var totalCampaigns int
	campaignQuery := `SELECT COUNT(DISTINCT c.id) FROM campaigns c WHERE c.created_at BETWEEN $1 AND $2`
	args := []interface{}{start, end}
	
	if nicheFilter != "all" {
		campaignQuery += " AND c.niche = $3"
		args = append(args, nicheFilter)
	}
	
	err = db.QueryRow(campaignQuery, args...).Scan(&totalCampaigns)
	if err != nil {
		totalCampaigns = 0
	}
	
	// Get broadcast statistics
	var totalContactsShouldSend, contactsDoneSend, contactsFailedSend int
	
	broadcastQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM broadcast_messages bm
		JOIN campaigns c ON bm.campaign_id = c.id
		WHERE bm.created_at BETWEEN $1 AND $2`
	
	args = []interface{}{start, end}
	argCount := 3
	
	if deviceFilter != "all" {
		broadcastQuery += " AND bm.device_id = $" + string(rune('0'+argCount))
		args = append(args, deviceFilter)
		argCount++
	}
	
	if nicheFilter != "all" {
		broadcastQuery += " AND c.niche = $" + string(rune('0'+argCount))
		args = append(args, nicheFilter)
	}
	
	err = db.QueryRow(broadcastQuery, args...).Scan(&totalContactsShouldSend, &contactsDoneSend, &contactsFailedSend)
	if err != nil {
		totalContactsShouldSend = 0
		contactsDoneSend = 0
		contactsFailedSend = 0
	}
	
	contactsRemainingSend := totalContactsShouldSend - contactsDoneSend - contactsFailedSend
	
	// Get chart data
	chartQuery := `
		SELECT 
			DATE(bm.created_at) as date,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM broadcast_messages bm
		JOIN campaigns c ON bm.campaign_id = c.id
		WHERE bm.created_at BETWEEN $1 AND $2`
	
	args = []interface{}{start, end}
	argCount = 3
	
	if deviceFilter != "all" {
		chartQuery += " AND bm.device_id = $" + string(rune('0'+argCount))
		args = append(args, deviceFilter)
		argCount++
	}
	
	if nicheFilter != "all" {
		chartQuery += " AND c.niche = $" + string(rune('0'+argCount))
		args = append(args, nicheFilter)
	}
	
	chartQuery += " GROUP BY DATE(bm.created_at) ORDER BY date"
	
	rows, err := db.Query(chartQuery, args...)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to get chart data",
		})
	}
	defer rows.Close()
	
	var labels []string
	var sentData []int
	var failedData []int
	
	for rows.Next() {
		var date time.Time
		var sent, failed int
		
		err := rows.Scan(&date, &sent, &failed)
		if err != nil {
			continue
		}
		
		labels = append(labels, date.Format("Jan 02"))
		sentData = append(sentData, sent)
		failedData = append(failedData, failed)
	}
	
	return c.JSON(fiber.Map{
		"totalCampaigns": totalCampaigns,
		"totalContactsShouldSend": totalContactsShouldSend,
		"contactsDoneSend": contactsDoneSend,
		"contactsFailedSend": contactsFailedSend,
		"contactsRemainingSend": contactsRemainingSend,
		"chartData": fiber.Map{
			"labels": labels,
			"sent": sentData,
			"failed": failedData,
		},
	})
}

// GetSequenceAnalytics returns sequence analytics data
func (handler *App) GetSequenceAnalytics(c *fiber.Ctx) error {
	// Get query parameters
	startDate := c.Query("start")
	endDate := c.Query("end")
	deviceFilter := c.Query("device", "all")
	nicheFilter := c.Query("niche", "all")
	
	// Validate dates
	if startDate == "" || endDate == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Start date and end date are required",
		})
	}
	
	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid start date format",
		})
	}
	
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid end date format",
		})
	}
	
	// Add 23:59:59 to end date to include the whole day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	
	// Get sequence statistics
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to connect to database",
		})
	}
	defer db.Close()
	
	// Total sequences
	var totalSequences int
	sequenceQuery := `SELECT COUNT(DISTINCT s.id) FROM sequences s WHERE s.created_at BETWEEN $1 AND $2`
	args := []interface{}{start, end}
	
	if nicheFilter != "all" {
		sequenceQuery += " AND s.niche = $3"
		args = append(args, nicheFilter)
	}
	
	err = db.QueryRow(sequenceQuery, args...).Scan(&totalSequences)
	if err != nil {
		totalSequences = 0
	}
	
	// Total flows
	var totalFlows int
	flowQuery := `SELECT COUNT(*) FROM sequence_steps ss JOIN sequences s ON ss.sequence_id = s.id WHERE s.created_at BETWEEN $1 AND $2`
	args = []interface{}{start, end}
	
	if nicheFilter != "all" {
		flowQuery += " AND s.niche = $3"
		args = append(args, nicheFilter)
	}
	
	err = db.QueryRow(flowQuery, args...).Scan(&totalFlows)
	if err != nil {
		totalFlows = 0
	}
	
	// Get sequence contact statistics
	var totalContactsShouldSend, contactsDoneSend, contactsFailedSend int
	
	contactQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM sequence_contacts sc
		JOIN sequences s ON sc.sequence_id = s.id
		WHERE sc.created_at BETWEEN $1 AND $2`
	
	args = []interface{}{start, end}
	argCount := 3
	
	if deviceFilter != "all" {
		contactQuery += " AND sc.processing_device_id = $" + string(rune('0'+argCount))
		args = append(args, deviceFilter)
		argCount++
	}
	
	if nicheFilter != "all" {
		contactQuery += " AND s.niche = $" + string(rune('0'+argCount))
		args = append(args, nicheFilter)
	}
	
	err = db.QueryRow(contactQuery, args...).Scan(&totalContactsShouldSend, &contactsDoneSend, &contactsFailedSend)
	if err != nil {
		totalContactsShouldSend = 0
		contactsDoneSend = 0
		contactsFailedSend = 0
	}
	
	contactsRemainingSend := totalContactsShouldSend - contactsDoneSend - contactsFailedSend
	
	// Get chart data
	chartQuery := `
		SELECT 
			DATE(sc.completed_at) as date,
			COUNT(CASE WHEN status = 'sent' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status IN ('pending', 'active') THEN 1 END) as pending
		FROM sequence_contacts sc
		JOIN sequences s ON sc.sequence_id = s.id
		WHERE sc.created_at BETWEEN $1 AND $2`
	
	args = []interface{}{start, end}
	argCount = 3
	
	if deviceFilter != "all" {
		chartQuery += " AND sc.processing_device_id = $" + string(rune('0'+argCount))
		args = append(args, deviceFilter)
		argCount++
	}
	
	if nicheFilter != "all" {
		chartQuery += " AND s.niche = $" + string(rune('0'+argCount))
		args = append(args, nicheFilter)
	}
	
	chartQuery += " GROUP BY DATE(sc.completed_at) ORDER BY date"
	
	rows, err := db.Query(chartQuery, args...)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to get chart data",
		})
	}
	defer rows.Close()
	
	var labels []string
	var completedData []int
	var failedData []int
	var pendingData []int
	
	for rows.Next() {
		var date sql.NullTime
		var completed, failed, pending int
		
		err := rows.Scan(&date, &completed, &failed, &pending)
		if err != nil {
			continue
		}
		
		if date.Valid {
			labels = append(labels, date.Time.Format("Jan 02"))
			completedData = append(completedData, completed)
			failedData = append(failedData, failed)
			pendingData = append(pendingData, pending)
		}
	}
	
	return c.JSON(fiber.Map{
		"totalSequences": totalSequences,
		"totalFlows": totalFlows,
		"totalContactsShouldSend": totalContactsShouldSend,
		"contactsDoneSend": contactsDoneSend,
		"contactsFailedSend": contactsFailedSend,
		"contactsRemainingSend": contactsRemainingSend,
		"chartData": fiber.Map{
			"labels": labels,
			"completed": completedData,
			"failed": failedData,
			"pending": pendingData,
		},
	})
}

// GetNiches returns all unique niches
func (handler *App) GetNiches(c *fiber.Ctx) error {
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to connect to database",
		})
	}
	defer db.Close()
	
	var niches []string
	
	// Get niches from campaigns and sequences
	query := `
		SELECT DISTINCT niche FROM (
			SELECT niche FROM campaigns WHERE niche IS NOT NULL AND niche != ''
			UNION
			SELECT niche FROM sequences WHERE niche IS NOT NULL AND niche != ''
		) AS all_niches
		ORDER BY niche
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to get niches",
		})
	}
	defer rows.Close()
	
	for rows.Next() {
		var niche string
		if err := rows.Scan(&niche); err == nil {
			niches = append(niches, niche)
		}
	}
	
	return c.JSON(niches)
}