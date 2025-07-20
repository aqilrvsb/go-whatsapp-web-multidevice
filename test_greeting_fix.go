package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Test the greeting processor fix
func main() {
	// Test the regex patterns
	oldRegex := regexp.MustCompile(`\{([^}]+)\}`)
	newRegex := regexp.MustCompile(`\{([^}]*\|[^}]*)\}`)
	
	templates := []string{
		"{Hi|Hello|Hai} {name}",
		"{name} {hi|hello}",
		"{Salam|Hi|Hello} {name}",
	}
	
	fmt.Println("=== TESTING GREETING PROCESSOR FIX ===")
	
	for _, template := range templates {
		fmt.Printf("\nTemplate: %s\n", template)
		
		// Old regex (buggy)
		oldResult := oldRegex.ReplaceAllStringFunc(template, func(match string) string {
			options := strings.Split(strings.Trim(match, "{}"), "|")
			if len(options) > 1 {
				return options[0] // Just return first option for testing
			}
			return strings.Trim(match, "{}")
		})
		fmt.Printf("Old regex result: %s\n", oldResult)
		
		// New regex (fixed)
		newResult := newRegex.ReplaceAllStringFunc(template, func(match string) string {
			options := strings.Split(strings.Trim(match, "{}"), "|")
			return options[0] // Just return first option for testing
		})
		fmt.Printf("New regex result: %s\n", newResult)
		
		// After name replacement
		finalResult := strings.ReplaceAll(newResult, "{name}", "Aqil")
		fmt.Printf("After name replacement: %s\n", finalResult)
	}
	
	// Test the full flow
	fmt.Println("\n=== FULL GREETING FLOW TEST ===")
	greeting := processGreeting("{Hi|Hello|Hai} {name}", "Aqil 1")
	message := greeting + "\n\n" + "Your message here"
	fmt.Printf("Final message:\n%s\n", message)
}

func processGreeting(template string, name string) string {
	// Process spintax (only with |)
	spintaxRegex := regexp.MustCompile(`\{([^}]*\|[^}]*)\}`)
	greeting := spintaxRegex.ReplaceAllStringFunc(template, func(match string) string {
		options := strings.Split(strings.Trim(match, "{}"), "|")
		return options[0] // Just return first for testing
	})
	
	// Replace name
	greeting = strings.ReplaceAll(greeting, "{name}", name)
	
	return greeting
}
