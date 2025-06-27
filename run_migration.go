package main

import (
	"database/sql"
	"log"
	"os"
	_ "github.com/lib/pq"
)

func main() {
	// Get database URI from environment
	dbURI := os.Getenv("DB_URI")
	if dbURI == "" {
		log.Fatal("DB_URI environment variable not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	migrations := []string{
		`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'all'`,
		`ALTER TABLE sequences ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'all'`,
		`UPDATE campaigns SET target_status = 'all' WHERE target_status IS NULL`,
		`UPDATE sequences SET target_status = 'all' WHERE target_status IS NULL`,
	}

	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration)
		_, err := db.Exec(migration)
		if err != nil {
			log.Printf("Warning: Migration failed (may already exist): %v", err)
		} else {
			log.Println("Migration successful")
		}
	}

	log.Println("All migrations completed")
}
