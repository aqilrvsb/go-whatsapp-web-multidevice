package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("=== WhatsApp Multi-Device Redis Diagnostics ===")
	fmt.Println()
	
	// Check all Redis-related environment variables
	redisVars := []string{
		"REDIS_URL",
		"redis_url",
		"RedisURL",
		"REDIS_PASSWORD",
		"REDISHOST",
		"REDIS_HOST",
		"REDISPORT",
		"REDIS_PORT",
	}
	
	fmt.Println("1. Environment Variables:")
	fmt.Println("-------------------------")
	foundRedisURL := false
	var validRedisURL string
	
	for _, varName := range redisVars {
		value := os.Getenv(varName)
		if value != "" {
			// Mask password for security
			displayValue := value
			if strings.Contains(varName, "PASSWORD") && len(value) > 4 {
				displayValue = value[:4] + "****"
			} else if strings.Contains(value, "@") && strings.Contains(value, "redis://") {
				parts := strings.Split(value, "@")
				if len(parts) > 1 {
					displayValue = "redis://****@" + parts[1]
				}
			}
			fmt.Printf("✓ %s = %s\n", varName, displayValue)
			
			if strings.Contains(varName, "URL") && strings.Contains(value, "redis://") {
				foundRedisURL = true
				validRedisURL = value
			}
		} else {
			fmt.Printf("✗ %s = (not set)\n", varName)
		}
	}
	
	fmt.Println()
	fmt.Println("2. Redis URL Validation:")
	fmt.Println("------------------------")
	
	if !foundRedisURL {
		fmt.Println("❌ No Redis URL found in environment variables!")
		fmt.Println("   Please set REDIS_URL environment variable.")
		return
	}
	
	// Validate Redis URL
	fmt.Printf("Found Redis URL: %s\n", validRedisURL)
	
	// Check validation criteria
	checks := map[string]bool{
		"Not empty":                validRedisURL != "",
		"No template variables":    !strings.Contains(validRedisURL, "${{"),
		"Not localhost":            !strings.Contains(validRedisURL, "localhost") && !strings.Contains(validRedisURL, "[::1]"),
		"Has redis:// scheme":      strings.Contains(validRedisURL, "redis://") || strings.Contains(validRedisURL, "rediss://"),
		"Has internal domain":      strings.Contains(validRedisURL, ".internal"),
	}
	
	allPassed := true
	for check, passed := range checks {
		if passed {
			fmt.Printf("✓ %s\n", check)
		} else {
			fmt.Printf("✗ %s\n", check)
			allPassed = false
		}
	}
	
	fmt.Println()
	fmt.Println("3. Diagnosis:")
	fmt.Println("-------------")
	
	if allPassed {
		fmt.Println("✅ Redis configuration looks correct!")
		fmt.Println("   The application should use Redis for broadcasting.")
	} else {
		fmt.Println("❌ Redis configuration has issues!")
		fmt.Println("   The application will fall back to in-memory broadcasting.")
		fmt.Println()
		fmt.Println("   To fix:")
		fmt.Println("   1. Make sure REDIS_URL is set correctly")
		fmt.Println("   2. Use the internal Railway domain (e.g., redis.railway.internal)")
		fmt.Println("   3. Avoid using localhost in production")
	}
	
	fmt.Println()
	fmt.Println("4. Your Railway Redis Environment:")
	fmt.Println("----------------------------------")
	fmt.Println(`REDIS_URL="redis://default:zwSXYXzTBYBreTwZtPbDVQLJUTHGqYnL@redis.railway.internal:6379"`)
	fmt.Println()
	fmt.Println("Make sure this is set in your Railway environment variables!")
}
