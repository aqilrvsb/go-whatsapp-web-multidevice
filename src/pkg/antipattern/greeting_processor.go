package antipattern

import (
	"strings"
	"github.com/sirupsen/logrus"
)

// GreetingProcessor handles greeting for messages
type GreetingProcessor struct{}

// NewGreetingProcessor creates a new greeting processor
func NewGreetingProcessor() *GreetingProcessor {
	return &GreetingProcessor{}
}

// GetAntiSpamGreeting - NOT USED ANYMORE, keeping for compatibility
func (g *GreetingProcessor) GetAntiSpamGreeting(name string, deviceID string, recipientPhone string) string {
	// Just return simple greeting
	return g.getSimpleGreeting(name)
}

// getSimpleGreeting returns a simple greeting with the name
func (g *GreetingProcessor) getSimpleGreeting(name string) string {
	// Handle empty name or phone number as name
	if name == "" || isPhoneNumber(name) {
		name = "there" // Use "there" instead of "Cik" for more universal greeting
	}
	
	// Simple greeting - NO SPINTAX!
	return "Hi " + name + ","
}

// PrepareMessageWithGreeting adds greeting to message
func (g *GreetingProcessor) PrepareMessageWithGreeting(originalMessage string, name string, deviceID string, recipientPhone string) string {
	logrus.Infof("[GREETING] Processing message for name='%s', phone='%s'", name, recipientPhone)
	
	// STEP 1: Get the simple greeting (NO SPINTAX!)
	greeting := g.getSimpleGreeting(name)
	logrus.Debugf("[GREETING] Simple greeting: %s", greeting)
	
	// STEP 2: Process the message content
	// First replace {name} in the message if exists
	processedMessage := originalMessage
	if strings.Contains(processedMessage, "{name}") {
		properName := name
		if name == "" || isPhoneNumber(name) {
			properName = "there"
		}
		processedMessage = strings.ReplaceAll(processedMessage, "{name}", properName)
	}
	
	// STEP 3: Fix line breaks in the message
	processedMessage = strings.ReplaceAll(processedMessage, "\\n", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0A", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0a", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br/>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br />", "\n")
	
	// STEP 4: Combine greeting + double line break + message
	// The spintax processing will happen LATER in the message randomizer
	finalMessage := greeting + "\n\n" + processedMessage
	
	logrus.Infof("[GREETING] Final message with greeting added (before spintax): %s", 
		strings.ReplaceAll(finalMessage[:min(100, len(finalMessage))], "\n", "\\n"))
	
	return finalMessage
}

// isPhoneNumber checks if the name looks like a phone number
func isPhoneNumber(name string) bool {
	// Remove spaces and check if mostly digits
	cleaned := strings.ReplaceAll(name, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	
	if len(cleaned) == 0 {
		return false
	}
	
	digitCount := 0
	for _, r := range cleaned {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	// If more than 70% digits, it's likely a phone number
	return float64(digitCount) > float64(len(cleaned))*0.7
}

// processSpintax - REMOVED! This should NOT be in greeting processor
// Spintax processing happens in message_randomizer.go AFTER greeting is added

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}