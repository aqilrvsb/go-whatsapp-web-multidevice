package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Test database type detection logic
	fmt.Println("Testing Database Type Detection")
	fmt.Println("===============================")
	
	// Check MYSQL_URI
	mysqlURI := os.Getenv("MYSQL_URI")
	fmt.Printf("MYSQL_URI: %s\n", mysqlURI)
	
	// Check DB_URI
	dbURI := os.Getenv("DB_URI")
	fmt.Printf("DB_URI: %s\n", dbURI)
	
	// Determine database type
	dbType := "mysql"
	if mysqlURI == "" {
		if dbURI == "" || strings.Contains(dbURI, "postgres") {
			dbType = "postgres"
		}
	}
	
	fmt.Printf("\nDetected Database Type: %s\n", dbType)
	
	// Show appropriate syntax
	if dbType == "mysql" {
		fmt.Println("\nUsing MySQL syntax:")
		fmt.Println("- ON DUPLICATE KEY UPDATE")
		fmt.Println("- SET FOREIGN_KEY_CHECKS = 0/1")
		fmt.Println("- Placeholders: ?, ?, ?")
	} else {
		fmt.Println("\nUsing PostgreSQL syntax:")
		fmt.Println("- ON CONFLICT DO UPDATE")
		fmt.Println("- SET session_replication_role = 'replica'/'origin'")
		fmt.Println("- Placeholders: $1, $2, $3")
	}
}
