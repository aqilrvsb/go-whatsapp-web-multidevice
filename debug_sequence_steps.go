package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Get database URI from environment or use default
	dbURI := os.Getenv("DB_URI")
	if dbURI == "" {
		dbURI = "postgresql://postgres:password@localhost:5432/whatsapp"
	}
	
	fmt.Printf("Connecting to database: %s\n", dbURI)
	
	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	fmt.Println("Connected successfully!")
	
	// Check table structure
	fmt.Println("\n=== CHECKING TABLE STRUCTURE ===")
	checkTableStructure(db)
	
	// Check data
	fmt.Println("\n=== CHECKING SEQUENCE DATA ===")
	checkSequenceData(db)
	
	// Check specific sequence steps
	fmt.Println("\n=== CHECKING SEQUENCE STEPS ===")
	checkSequenceSteps(db, "394d567f-e5bd-476d-ae7c-c39f74819d70")
	checkSequenceSteps(db, "b7319119-4d21-43cc-97e2-9644ded0608c")
}

func checkTableStructure(db *sql.DB) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = 'sequence_steps'
		ORDER BY ordinal_position
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error checking table structure: %v", err)
		return
	}
	defer rows.Close()
	
	fmt.Println("Columns in sequence_steps table:")
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault sql.NullString
		
		err := rows.Scan(&colName, &dataType, &isNullable, &colDefault)
		if err != nil {
			log.Printf("Error scanning column info: %v", err)
			continue
		}
		
		defaultVal := "NULL"
		if colDefault.Valid {
			defaultVal = colDefault.String
		}
		
		fmt.Printf("  - %s: %s (nullable: %s, default: %s)\n", 
			colName, dataType, isNullable, defaultVal)
	}
}

func checkSequenceData(db *sql.DB) {
	query := `
		SELECT id, name, step_count, 
		       (SELECT COUNT(*) FROM sequence_steps WHERE sequence_id = s.id) as actual_steps
		FROM sequences s
		WHERE user_id = 'de078f16-3266-4ab3-8153-a248b015228f'
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error checking sequences: %v", err)
		return
	}
	defer rows.Close()
	
	fmt.Println("Sequences:")
	for rows.Next() {
		var id, name string
		var stepCount, actualSteps int
		
		err := rows.Scan(&id, &name, &stepCount, &actualSteps)
		if err != nil {
			log.Printf("Error scanning sequence: %v", err)
			continue
		}
		
		fmt.Printf("  - %s: %s (step_count: %d, actual_steps: %d)\n", 
			id, name, stepCount, actualSteps)
	}
}

func checkSequenceSteps(db *sql.DB, sequenceID string) {
	fmt.Printf("\nSteps for sequence %s:\n", sequenceID)
	
	// First, count the steps
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sequence_steps WHERE sequence_id = $1", sequenceID).Scan(&count)
	if err != nil {
		log.Printf("Error counting steps: %v", err)
		return
	}
	
	fmt.Printf("Total steps in database: %d\n", count)
	
	if count == 0 {
		fmt.Println("No steps found!")
		return
	}
	
	// Try to query the steps
	query := `
		SELECT 
			id,
			COALESCE(day_number, 0) as day_number,
			COALESCE(message_type, '') as message_type,
			COALESCE(content, '') as content,
			COALESCE(trigger, '') as trigger,
			COALESCE(next_trigger, '') as next_trigger
		FROM sequence_steps 
		WHERE sequence_id = $1
		ORDER BY day_number
	`
	
	rows, err := db.Query(query, sequenceID)
	if err != nil {
		log.Printf("Error querying steps: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var id, messageType, content, trigger, nextTrigger string
		var dayNumber int
		
		err := rows.Scan(&id, &dayNumber, &messageType, &content, &trigger, &nextTrigger)
		if err != nil {
			log.Printf("Error scanning step: %v", err)
			continue
		}
		
		fmt.Printf("  Step %d: %s (type: %s, trigger: %s -> %s)\n", 
			dayNumber, id[:8], messageType, trigger, nextTrigger)
		fmt.Printf("    Content: %.50s...\n", content)
	}
}
