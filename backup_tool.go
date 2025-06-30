package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	_ "github.com/lib/pq"
)

func main() {
	// Database URL
	dbURL := "postgres://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway?sslmode=require"
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()
	
	// Create backup directory
	backupDir := "backups/2025-07-01_00-01-03_working_version"
	
	// Get table list and counts
	fmt.Println("Creating database backup...")
	fmt.Println("========================")
	
	// Create stats file
	statsFile, err := os.Create(backupDir + "/database_stats.json")
	if err != nil {
		log.Fatal("Failed to create stats file:", err)
	}
	defer statsFile.Close()
	
	stats := make(map[string]interface{})
	stats["backup_date"] = time.Now().Format("2006-01-02 15:04:05")
	stats["tables"] = make(map[string]int)
	
	// Tables to backup
	tables := []string{
		"users", "devices", "leads", "campaigns", "broadcast_messages",
		"sequences", "sequence_steps", "sequence_contacts",
		"whatsapp_chats", "whatsapp_messages",
	}
	
	// Get counts for each table
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			stats["tables"].(map[string]int)[table] = count
			fmt.Printf("%s: %d records\n", table, count)
		}
	}
	
	// Write stats
	encoder := json.NewEncoder(statsFile)
	encoder.SetIndent("", "  ")
	encoder.Encode(stats)
	
	// Create SQL backup for critical tables
	fmt.Println("\nExporting critical data...")
	
	// Export campaigns
	exportTable(db, "campaigns", backupDir)
	exportTable(db, "leads", backupDir)
	exportTable(db, "devices", backupDir)
	exportTable(db, "users", backupDir)
	exportTable(db, "sequences", backupDir)
	
	fmt.Println("\nBackup completed!")
	fmt.Printf("Files saved in: %s\n", backupDir)
}

func exportTable(db *sql.DB, tableName, backupDir string) {
	file, err := os.Create(fmt.Sprintf("%s/%s.json", backupDir, tableName))
	if err != nil {
		log.Printf("Failed to create file for %s: %v", tableName, err)
		return
	}
	defer file.Close()
	
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		log.Printf("Failed to query %s: %v", tableName, err)
		return
	}
	defer rows.Close()
	
	// Get column names
	columns, _ := rows.Columns()
	
	var results []map[string]interface{}
	
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range columns {
			valuePointers[i] = &values[i]
		}
		
		if err := rows.Scan(valuePointers...); err != nil {
			continue
		}
		
		// Create map for this row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}
		results = append(results, rowMap)
	}
	
	// Write to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(results)
	
	fmt.Printf("Exported %s: %d records\n", tableName, len(results))
}
