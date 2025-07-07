package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection from environment
	dbURI := os.Getenv("DATABASE_URL")
	if dbURI == "" {
		dbURI = os.Getenv("DB_URI")
	}
	if dbURI == "" {
		log.Fatal("DATABASE_URL or DB_URI environment variable required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	log.Println("🔧 Fixing sequence steps...")

	// Fix the "Invalid Date" issue
	result, err := db.Exec(`
		UPDATE sequence_steps 
		SET send_time = '10:00', time_schedule = '10:00'
		WHERE sequence_id = '394d567f-e5bd-476d-ae7c-c39f74819d70' 
		AND send_time = 'Invalid Date'
	`)
	if err != nil {
		log.Fatal("Failed to fix send_time:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ Fixed send_time for %d rows", rowsAffected)

	// Add missing columns if they don't exist
	log.Println("🔧 Adding missing columns...")
	
	columns := []string{
		"ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS image_url TEXT",
		"ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS min_delay_seconds INTEGER DEFAULT 10",
		"ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS max_delay_seconds INTEGER DEFAULT 30",
	}

	for _, sql := range columns {
		_, err := db.Exec(sql)
		if err != nil {
			log.Printf("Warning: Failed to add column: %v", err)
		}
	}

	// Update missing values
	_, err = db.Exec(`
		UPDATE sequence_steps 
		SET 
			min_delay_seconds = COALESCE(min_delay_seconds, 10),
			max_delay_seconds = COALESCE(max_delay_seconds, 30)
		WHERE sequence_id = '394d567f-e5bd-476d-ae7c-c39f74819d70'
	`)
	if err != nil {
		log.Printf("Warning: Failed to update delay values: %v", err)
	}

	// Update sequence step count
	_, err = db.Exec(`
		UPDATE sequences 
		SET step_count = (SELECT COUNT(*) FROM sequence_steps WHERE sequence_id = sequences.id)
		WHERE id = '394d567f-e5bd-476d-ae7c-c39f74819d70'
	`)
	if err != nil {
		log.Printf("Warning: Failed to update step count: %v", err)
	}

	// Verify the fix
	var content, sendTime string
	var stepCount int
	err = db.QueryRow(`
		SELECT s.step_count, ss.content, ss.send_time 
		FROM sequences s 
		LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id 
		WHERE s.id = '394d567f-e5bd-476d-ae7c-c39f74819d70'
	`).Scan(&stepCount, &content, &sendTime)
	
	if err != nil {
		log.Printf("Verification failed: %v", err)
	} else {
		log.Printf("✅ VERIFICATION SUCCESS:")
		log.Printf("   Step Count: %d", stepCount)
		log.Printf("   Content: %s", content)
		log.Printf("   Send Time: %s", sendTime)
	}

	log.Println("🎉 Fix completed! Restart your application and test.")
}
