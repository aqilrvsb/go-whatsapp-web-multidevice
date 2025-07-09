package rest

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
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
			Message: "Invalid start date format: " + err.Error(),
		})
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid end date format: " + err.Error(),
		})
	}

	// Add 23:59:59 to end date to include the whole day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// Get database connection
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to connect to database: " + err.Error(),
		})
	}
	defer db.Close()

	// Initialize response data
	totalCampaigns := 0
	totalContactsShouldSend := 0
	contactsDoneSend := 0
	contactsFailedSend := 0
	contactsRemainingSend := 0
	
	labels := []string{}
	sentData := []int{}
	failedData := []int{}

	// Get total campaigns count with error handling
	campaignQuery := `SELECT COUNT(DISTINCT c.id) FROM campaigns c WHERE c.created_at BETWEEN $1 AND $2`
	args := []interface{}{start, end}

	if nicheFilter != "all" && nicheFilter != "" {
		campaignQuery += " AND c.niche = $3"
		args = append(args, nicheFilter)
	}

	err = db.QueryRow(campaignQuery, args...).Scan(&totalCampaigns)
	if err != nil && err != sql.ErrNoRows {
		// Log error but continue
		fmt.Printf("Error counting campaigns: %v\n", err)
	}

	// Get broadcast statistics with proper error handling
	broadcastQuery := `
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(COUNT(CASE WHEN status = 'sent' THEN 1 END), 0) as sent,
			COALESCE(COUNT(CASE WHEN status = 'failed' THEN 1 END), 0) as failed
		FROM broadcast_messages bm
		INNER JOIN campaigns c ON bm.campaign_id = c.id
		WHERE bm.created_at BETWEEN $1 AND $2`

	args = []interface{}{start, end}
	argCount := 3

	if deviceFilter != "all" && deviceFilter != "" {
		broadcastQuery += fmt.Sprintf(" AND bm.device_id = $%d", argCount)
		args = append(args, deviceFilter)
		argCount++
	}

	if nicheFilter != "all" && nicheFilter != "" {
		broadcastQuery += fmt.Sprintf(" AND c.niche = $%d", argCount)
		args = append(args, nicheFilter)
	}

	err = db.QueryRow(broadcastQuery, args...).Scan(&totalContactsShouldSend, &contactsDoneSend, &contactsFailedSend)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error getting broadcast stats: %v\n", err)
	}

	contactsRemainingSend = totalContactsShouldSend - contactsDoneSend - contactsFailedSend
	if contactsRemainingSend < 0 {
		contactsRemainingSend = 0
	}

	// Get chart data with error handling
	chartQuery := `
		SELECT 
			DATE(bm.created_at) as date,
			COALESCE(COUNT(CASE WHEN status = 'sent' THEN 1 END), 0) as sent,
			COALESCE(COUNT(CASE WHEN status = 'failed' THEN 1 END), 0) as failed
		FROM broadcast_messages bm
		INNER JOIN campaigns c ON bm.campaign_id = c.id
		WHERE bm.created_at BETWEEN $1 AND $2`

	args = []interface{}{start, end}
	argCount = 3

	if deviceFilter != "all" && deviceFilter != "" {
		chartQuery += fmt.Sprintf(" AND bm.device_id = $%d", argCount)
		args = append(args, deviceFilter)
		argCount++
	}

	if nicheFilter != "all" && nicheFilter != "" {
		chartQuery += fmt.Sprintf(" AND c.niche = $%d", argCount)
		args = append(args, nicheFilter)
	}

	chartQuery += " GROUP BY DATE(bm.created_at) ORDER BY date"

	rows, err := db.Query(chartQuery, args...)
	if err != nil {
		fmt.Printf("Error getting chart data: %v\n", err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var date time.Time
			var sent, failed int

			err := rows.Scan(&date, &sent, &failed)
			if err != nil {
				fmt.Printf("Error scanning row: %v\n", err)
				continue
			}

			labels = append(labels, date.Format("Jan 02"))
			sentData = append(sentData, sent)
			failedData = append(failedData, failed)
		}
	}

	// If no data, create empty arrays to prevent null in JSON
	if len(labels) == 0 {
		// Generate dates for the range even if no data
		current := start
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Jan 02"))
			sentData = append(sentData, 0)
			failedData = append(failedData, 0)
			current = current.AddDate(0, 0, 1)
		}
	}

	return c.JSON(fiber.Map{
		"totalCampaigns":          totalCampaigns,
		"totalContactsShouldSend": totalContactsShouldSend,
		"contactsDoneSend":        contactsDoneSend,
		"contactsFailedSend":      contactsFailedSend,
		"contactsRemainingSend":   contactsRemainingSend,
		"chartData": fiber.Map{
			"labels": labels,
			"sent":   sentData,
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
			Message: "Invalid start date format: " + err.Error(),
		})
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid end date format: " + err.Error(),
		})
	}

	// Add 23:59:59 to end date to include the whole day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// Get database connection
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DATABASE_ERROR",
			Message: "Failed to connect to database: " + err.Error(),
		})
	}
	defer db.Close()

	// Initialize response data
	totalSequences := 0
	totalFlows := 0
	totalContactsShouldSend := 0
	contactsDoneSend := 0
	contactsFailedSend := 0
	contactsRemainingSend := 0
	
	labels := []string{}
	completedData := []int{}
	failedData := []int{}
	pendingData := []int{}

	// Get total sequences with error handling
	sequenceQuery := `SELECT COUNT(DISTINCT s.id) FROM sequences s WHERE s.created_at BETWEEN $1 AND $2`
	args := []interface{}{start, end}

	if nicheFilter != "all" && nicheFilter != "" {
		sequenceQuery += " AND s.niche = $3"
		args = append(args, nicheFilter)
	}

	err = db.QueryRow(sequenceQuery, args...).Scan(&totalSequences)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error counting sequences: %v\n", err)
	}

	// Get total flows
	flowQuery := `SELECT COUNT(*) FROM sequence_steps ss INNER JOIN sequences s ON ss.sequence_id = s.id WHERE s.created_at BETWEEN $1 AND $2`
	args = []interface{}{start, end}

	if nicheFilter != "all" && nicheFilter != "" {
		flowQuery += " AND s.niche = $3"
		args = append(args, nicheFilter)
	}

	err = db.QueryRow(flowQuery, args...).Scan(&totalFlows)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error counting flows: %v\n", err)
	}

	// Get sequence contact statistics
	contactQuery := `
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(COUNT(CASE WHEN status = 'sent' THEN 1 END), 0) as sent,
			COALESCE(COUNT(CASE WHEN status = 'failed' THEN 1 END), 0) as failed
		FROM sequence_contacts sc
		INNER JOIN sequences s ON sc.sequence_id = s.id
		WHERE sc.created_at BETWEEN $1 AND $2`

	args = []interface{}{start, end}
	argCount := 3

	if deviceFilter != "all" && deviceFilter != "" {
		contactQuery += fmt.Sprintf(" AND sc.processing_device_id = $%d", argCount)
		args = append(args, deviceFilter)
		argCount++
	}

	if nicheFilter != "all" && nicheFilter != "" {
		contactQuery += fmt.Sprintf(" AND s.niche = $%d", argCount)
		args = append(args, nicheFilter)
	}

	err = db.QueryRow(contactQuery, args...).Scan(&totalContactsShouldSend, &contactsDoneSend, &contactsFailedSend)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error getting sequence contact stats: %v\n", err)
	}

	contactsRemainingSend = totalContactsShouldSend - contactsDoneSend - contactsFailedSend
	if contactsRemainingSend < 0 {
		contactsRemainingSend = 0
	}

	// Get chart data
	chartQuery := `
		SELECT 
			DATE(COALESCE(sc.completed_at, sc.created_at)) as date,
			COALESCE(COUNT(CASE WHEN status = 'sent' THEN 1 END), 0) as completed,
			COALESCE(COUNT(CASE WHEN status = 'failed' THEN 1 END), 0) as failed,
			COALESCE(COUNT(CASE WHEN status IN ('pending', 'active') THEN 1 END), 0) as pending
		FROM sequence_contacts sc
		INNER JOIN sequences s ON sc.sequence_id = s.id
		WHERE sc.created_at BETWEEN $1 AND $2`

	args = []interface{}{start, end}
	argCount = 3

	if deviceFilter != "all" && deviceFilter != "" {
		chartQuery += fmt.Sprintf(" AND sc.processing_device_id = $%d", argCount)
		args = append(args, deviceFilter)
		argCount++
	}

	if nicheFilter != "all" && nicheFilter != "" {
		chartQuery += fmt.Sprintf(" AND s.niche = $%d", argCount)
		args = append(args, nicheFilter)
	}

	chartQuery += " GROUP BY date ORDER BY date"

	rows, err := db.Query(chartQuery, args...)
	if err != nil {
		fmt.Printf("Error getting sequence chart data: %v\n", err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var date time.Time
			var completed, failed, pending int

			err := rows.Scan(&date, &completed, &failed, &pending)
			if err != nil {
				fmt.Printf("Error scanning sequence row: %v\n", err)
				continue
			}

			labels = append(labels, date.Format("Jan 02"))
			completedData = append(completedData, completed)
			failedData = append(failedData, failed)
			pendingData = append(pendingData, pending)
		}
	}

	// If no data, create empty arrays to prevent null in JSON
	if len(labels) == 0 {
		// Generate dates for the range even if no data
		current := start
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Jan 02"))
			completedData = append(completedData, 0)
			failedData = append(failedData, 0)
			pendingData = append(pendingData, 0)
			current = current.AddDate(0, 0, 1)
		}
	}

	return c.JSON(fiber.Map{
		"totalSequences":          totalSequences,
		"totalFlows":              totalFlows,
		"totalContactsShouldSend": totalContactsShouldSend,
		"contactsDoneSend":        contactsDoneSend,
		"contactsFailedSend":      contactsFailedSend,
		"contactsRemainingSend":   contactsRemainingSend,
		"chartData": fiber.Map{
			"labels":    labels,
			"completed": completedData,
			"failed":    failedData,
			"pending":   pendingData,
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
			Message: "Failed to connect to database: " + err.Error(),
		})
	}
	defer db.Close()

	niches := []string{}

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
		fmt.Printf("Error getting niches: %v\n", err)
		// Return empty array instead of error
		return c.JSON(niches)
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

// TestDatabaseConnection tests database tables and returns diagnostic info
func (handler *App) TestDatabaseConnection(c *fiber.Ctx) error {
	db, err := sql.Open("postgres", config.DBURI)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to connect to database: " + err.Error(),
		})
	}
	defer db.Close()

	diagnostics := fiber.Map{}

	// Test campaigns table
	var campaignCount int
	err = db.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&campaignCount)
	if err != nil {
		diagnostics["campaigns_error"] = err.Error()
	} else {
		diagnostics["campaigns_count"] = campaignCount
	}

	// Test broadcast_messages table
	var broadcastCount int
	err = db.QueryRow("SELECT COUNT(*) FROM broadcast_messages").Scan(&broadcastCount)
	if err != nil {
		diagnostics["broadcast_messages_error"] = err.Error()
	} else {
		diagnostics["broadcast_messages_count"] = broadcastCount
	}

	// Test sequences table
	var sequenceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sequences").Scan(&sequenceCount)
	if err != nil {
		diagnostics["sequences_error"] = err.Error()
	} else {
		diagnostics["sequences_count"] = sequenceCount
	}

	// Test sequence_contacts table
	var sequenceContactsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sequence_contacts").Scan(&sequenceContactsCount)
	if err != nil {
		diagnostics["sequence_contacts_error"] = err.Error()
	} else {
		diagnostics["sequence_contacts_count"] = sequenceContactsCount
	}

	// Check campaign columns
	rows, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'campaigns' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows.Close()
		columns := []string{}
		for rows.Next() {
			var colName, dataType string
			if err := rows.Scan(&colName, &dataType); err == nil {
				columns = append(columns, colName+" ("+dataType+")")
			}
		}
		diagnostics["campaigns_columns"] = columns
	}

	// Check broadcast_messages columns
	rows2, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'broadcast_messages' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows2.Close()
		columns := []string{}
		for rows2.Next() {
			var colName, dataType string
			if err := rows2.Scan(&colName, &dataType); err == nil {
				columns = append(columns, colName+" ("+dataType+")")
			}
		}
		diagnostics["broadcast_messages_columns"] = columns
	}

	return c.JSON(diagnostics)
}