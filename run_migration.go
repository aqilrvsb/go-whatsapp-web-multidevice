package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment or use the one from your config
	dbURL := os.Getenv("DB_URI")
	if dbURL == "" {
		// Replace this with your actual Railway PostgreSQL URL
		log.Fatal("Please set DB_URI environment variable with your Railway PostgreSQL connection string")
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

	fmt.Println("Connected to Railway PostgreSQL successfully!")

	// Run migrations
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "Remove campaign unique constraint",
			sql:  `ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_user_id_campaign_date_key;`,
		},
		{
			name: "Add campaign index for performance",
			sql:  `CREATE INDEX IF NOT EXISTS idx_campaigns_user_date ON campaigns(user_id, campaign_date);`,
		},
		{
			name: "Fix campaign table columns",
			sql: `
				-- Ensure niche column can be NULL
				ALTER TABLE campaigns ALTER COLUMN niche DROP NOT NULL;
				
				-- Ensure image_url column can be NULL  
				ALTER TABLE campaigns ALTER COLUMN image_url DROP NOT NULL;
				
				-- Ensure scheduled_time column can be NULL
				ALTER TABLE campaigns ALTER COLUMN scheduled_time DROP NOT NULL;
			`,
		},
	}

	// Execute each migration
	for _, migration := range migrations {
		fmt.Printf("Running migration: %s\n", migration.name)
		_, err := db.Exec(migration.sql)
		if err != nil {
			fmt.Printf("Warning: %s failed: %v\n", migration.name, err)
		} else {
			fmt.Printf("✓ %s completed successfully\n", migration.name)
		}
	}

	// Verify the constraint was removed
	var constraintExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.table_constraints 
			WHERE constraint_name = 'campaigns_user_id_campaign_date_key'
			AND table_name = 'campaigns'
		);
	`).Scan(&constraintExists)

	if err != nil {
		log.Printf("Failed to check constraint: %v", err)
	} else if constraintExists {
		fmt.Println("⚠️  Warning: Unique constraint still exists!")
	} else {
		fmt.Println("✅ Unique constraint successfully removed!")
	}

	// Show current campaigns to verify multiple per date work
	rows, err := db.Query(`
		SELECT campaign_date, COUNT(*) as count 
		FROM campaigns 
		GROUP BY campaign_date 
		HAVING COUNT(*) > 1
		ORDER BY campaign_date DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		
		fmt.Println("\nDates with multiple campaigns:")
		hasMultiple := false
		for rows.Next() {
			var date string
			var count int
			rows.Scan(&date, &count)
			fmt.Printf("- %s: %d campaigns\n", date, count)
			hasMultiple = true
		}
		if !hasMultiple {
			fmt.Println("- No dates with multiple campaigns yet")
		}
	}

	fmt.Println("\n✅ All migrations completed!")
}
