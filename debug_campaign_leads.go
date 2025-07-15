package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://user:password@localhost/whatsapp?sslmode=disable"
	}
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	fmt.Println("=== Campaign Lead Matching Debug ===")
	fmt.Println()
	
	userID := "c57309ce-0e4c-4c26-b6cd-092cc69f3806"
	
	// 1. Check all devices for this user
	fmt.Println("1. Checking devices for user:")
	rows, err := db.Query(`
		SELECT id, device_name, status, platform 
		FROM user_devices 
		WHERE user_id = $1
		ORDER BY device_name
	`, userID)
	if err != nil {
		log.Fatal("Failed to query devices:", err)
	}
	defer rows.Close()
	
	var deviceIDs []string
	fmt.Println("Devices:")
	for rows.Next() {
		var id, name, status string
		var platform sql.NullString
		rows.Scan(&id, &name, &status, &platform)
		deviceIDs = append(deviceIDs, id)
		platformStr := ""
		if platform.Valid && platform.String != "" {
			platformStr = fmt.Sprintf(" (platform: %s)", platform.String)
		}
		fmt.Printf("- %s: %s [%s]%s\n", id, name, status, platformStr)
	}
	
	// 2. Check leads for each device
	fmt.Println("\n2. Checking leads by device:")
	for _, deviceID := range deviceIDs {
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM leads 
			WHERE device_id = $1 AND niche = 'B1' AND target_status = 'prospect'
		`, deviceID).Scan(&count)
		if err == nil {
			fmt.Printf("- Device %s: %d leads\n", deviceID, count)
		}
	}
	
	// 3. Check total leads with criteria
	fmt.Println("\n3. Total leads matching campaign criteria:")
	var totalLeads int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM leads 
		WHERE user_id = $1 AND niche = 'B1' AND target_status = 'prospect'
	`, userID).Scan(&totalLeads)
	if err == nil {
		fmt.Printf("Total: %d leads\n", totalLeads)
	}
	
	// 4. Check which devices are online/connected
	fmt.Println("\n4. Online/Connected devices:")
	rows, err = db.Query(`
		SELECT id, device_name, status 
		FROM user_devices 
		WHERE user_id = $1 
		AND (status IN ('online', 'connected') OR platform IS NOT NULL AND platform != '')
		ORDER BY device_name
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, name, status string
			rows.Scan(&id, &name, &status)
			fmt.Printf("- %s: %s [%s]\n", id, name, status)
		}
	}
	
	// 5. Check campaign settings
	fmt.Println("\n5. Recent campaign settings:")
	rows, err = db.Query(`
		SELECT id, title, niche, target_status, status 
		FROM campaigns 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT 5
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var title, niche, targetStatus, status string
			rows.Scan(&id, &title, &niche, &targetStatus, &status)
			fmt.Printf("- Campaign %d: %s (niche: %s, target: %s, status: %s)\n", 
				id, title, niche, targetStatus, status)
		}
	}
	
	fmt.Println("\n=== Debug Complete ===")
}
