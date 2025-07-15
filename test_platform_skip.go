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
	
	fmt.Println("=== Platform Skip Feature Test ===")
	fmt.Println()
	
	// 1. Check if platform column exists
	var columnExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'user_devices' 
			AND column_name = 'platform'
		)
	`).Scan(&columnExists)
	
	if err != nil {
		log.Fatal("Failed to check column:", err)
	}
	
	if !columnExists {
		fmt.Println("❌ Platform column does not exist. Please run the migration SQL first.")
		return
	}
	
	fmt.Println("✅ Platform column exists")
	
	// 2. Count devices with and without platform
	var totalDevices, platformDevices, onlineDevices, onlineNoPlatform int
	
	db.QueryRow("SELECT COUNT(*) FROM user_devices").Scan(&totalDevices)
	db.QueryRow("SELECT COUNT(*) FROM user_devices WHERE platform IS NOT NULL AND platform != ''").Scan(&platformDevices)
	db.QueryRow("SELECT COUNT(*) FROM user_devices WHERE status = 'online'").Scan(&onlineDevices)
	db.QueryRow("SELECT COUNT(*) FROM user_devices WHERE status = 'online' AND (platform IS NULL OR platform = '')").Scan(&onlineNoPlatform)
	
	fmt.Println()
	fmt.Println("Device Statistics:")
	fmt.Printf("- Total devices: %d\n", totalDevices)
	fmt.Printf("- Devices with platform: %d\n", platformDevices)
	fmt.Printf("- Online devices: %d\n", onlineDevices)
	fmt.Printf("- Online devices without platform: %d (would be checked)\n", onlineNoPlatform)
	fmt.Printf("- Devices that would be skipped: %d\n", platformDevices)
	
	// 3. Show devices with platform
	if platformDevices > 0 {
		fmt.Println()
		fmt.Println("Devices with platform set:")
		
		rows, err := db.Query(`
			SELECT id, device_name, status, platform 
			FROM user_devices 
			WHERE platform IS NOT NULL AND platform != ''
			ORDER BY device_name
		`)
		if err == nil {
			defer rows.Close()
			
			for rows.Next() {
				var id, name, status, platform string
				rows.Scan(&id, &name, &status, &platform)
				fmt.Printf("- %s (status: %s, platform: %s)\n", name, status, platform)
			}
		}
	}
	
	// 4. Test query that would be used by status normalizer
	fmt.Println()
	fmt.Println("Testing Status Normalizer Query:")
	
	rows, err := db.Query(`
		SELECT COUNT(*) 
		FROM user_devices 
		WHERE (platform IS NULL OR platform = '')
		AND status NOT IN ('online', 'offline')
	`)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
	} else {
		rows.Close()
		fmt.Println("✅ Query successful")
	}
	
	// 5. Test query that would be used by connection monitor
	fmt.Println()
	fmt.Println("Testing Connection Monitor Query:")
	
	rows, err = db.Query(`
		SELECT id, device_name, status 
		FROM user_devices 
		WHERE (platform IS NULL OR platform = '')
		ORDER BY device_name
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
	} else {
		defer rows.Close()
		fmt.Println("✅ Query successful")
		
		fmt.Println("Sample devices that would be checked:")
		count := 0
		for rows.Next() {
			var id, name, status string
			rows.Scan(&id, &name, &status)
			fmt.Printf("- %s (status: %s)\n", name, status)
			count++
		}
		if count == 0 {
			fmt.Println("(No devices without platform found)")
		}
	}
	
	fmt.Println()
	fmt.Println("=== Test Complete ===")
}
