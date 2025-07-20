package main

import (
	"fmt"
	"strings"
)

func main() {
	// Test the name detection
	testNames := []string{
		"Aqil 1",
		"Aqil",
		"60123456789",
		"+60-123-456-789",
		"John Doe",
		"123",
		"",
	}
	
	fmt.Println("=== TESTING NAME DETECTION ===")
	for _, name := range testNames {
		isPhone := isPhoneNumber(name)
		fmt.Printf("Name: '%s' -> Phone number? %v\n", name, isPhone)
	}
	
	// Test greeting with line breaks
	fmt.Println("\n=== TESTING GREETING FORMAT ===")
	greeting := "Hi Aqil 1, apa khabar"
	message := "Lebih 90% perkembangan otak berlaku sebelum umur 12 tahun."
	fullMessage := greeting + "\n\n" + message
	
	fmt.Println("Full message:")
	fmt.Println(fullMessage)
	fmt.Println("\nMessage length:", len(fullMessage))
	fmt.Println("Contains \\n\\n?", strings.Contains(fullMessage, "\n\n"))
}

func isPhoneNumber(name string) bool {
	// Remove spaces and check if it's mostly digits
	cleaned := strings.ReplaceAll(name, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	
	// If empty after cleaning, it's not a valid name
	if cleaned == "" {
		return true
	}
	
	// Check if at least 80% are digits
	digitCount := 0
	for _, ch := range cleaned {
		if ch >= '0' && ch <= '9' {
			digitCount++
		}
	}
	
	// If more than 80% digits, it's likely a phone number
	return float64(digitCount) / float64(len(cleaned)) > 0.8
}
