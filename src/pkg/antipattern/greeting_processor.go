package antipattern

import (
	"strings"
	"time"
	"math/rand"
	"github.com/sirupsen/logrus"
)

// GreetingProcessor handles greeting for messages
type GreetingProcessor struct {
	// Malaysian greetings - NO SPINTAX, will randomly pick one
	greetings []string
}

// NewGreetingProcessor creates a new greeting processor
func NewGreetingProcessor() *GreetingProcessor {
	return &GreetingProcessor{
		greetings: []string{
			"Salam {name}",
			"Hi {name}",
			"Apa khabar {name}",
			"Maaf ganggu {name}",
			"Pinjam masa {name}",
		},
	}
}

// getSimpleGreeting returns a Malaysian greeting based on time or random
func (g *GreetingProcessor) getSimpleGreeting(name string) string {
	// Handle empty name or phone number as name
	if name == "" || isPhoneNumber(name) {
		// For Malaysian context, use more polite generic terms
		name = "tuan/puan" // or could use "saudara/i"
	}
	
	// Get current hour for time-based greetings
	hour := time.Now().Hour()
	var greeting string
	
	// Time-based Malaysian greetings
	if hour >= 5 && hour < 12 {
		// Morning greetings
		morningGreetings := []string{
			"Selamat pagi {name}",
			"Pagi {name}",
			"Assalamualaikum {name}",
		}
		greeting = morningGreetings[rand.Intn(len(morningGreetings))]
	} else if hour >= 12 && hour < 15 {
		// Afternoon
		afternoonGreetings := []string{
			"Selamat tengahari {name}",
			"Salam {name}",
			"Hi {name}",
		}
		greeting = afternoonGreetings[rand.Intn(len(afternoonGreetings))]
	} else if hour >= 15 && hour < 19 {
		// Late afternoon/evening
		eveningGreetings := []string{
			"Selamat petang {name}",
			"Petang {name}",
			"Salam {name}",
		}
		greeting = eveningGreetings[rand.Intn(len(eveningGreetings))]
	} else {
		// Night (7pm - 5am)
		nightGreetings := []string{
			"Selamat malam {name}",
			"Malam {name}",
			"Maaf ganggu {name}",
			"Pinjam masa {name}",
		}
		greeting = nightGreetings[rand.Intn(len(nightGreetings))]
	}
	
	// Replace {name} with actual name
	greeting = strings.ReplaceAll(greeting, "{name}", name)
	
	// Add comma at the end
	return greeting + ","
}

// GetAntiSpamGreeting - keeping for compatibility
func (g *GreetingProcessor) GetAntiSpamGreeting(name string, deviceID string, recipientPhone string) string {
	return g.getSimpleGreeting(name)
}

// PrepareMessageWithGreeting adds greeting to message
func (g *GreetingProcessor) PrepareMessageWithGreeting(originalMessage string, name string, deviceID string, recipientPhone string) string {
	logrus.Infof("[GREETING] Processing message for name='%s', phone='%s'", name, recipientPhone)
	
	// STEP 1: Get the simple Malaysian greeting (NO SPINTAX!)
	greeting := g.getSimpleGreeting(name)
	logrus.Debugf("[GREETING] Malaysian greeting: %s", greeting)
	
	// STEP 2: Process the message content
	// First replace {name} in the message if exists
	processedMessage := originalMessage
	if strings.Contains(processedMessage, "{name}") {
		properName := name
		if name == "" || isPhoneNumber(name) {
			properName = "tuan/puan"
		}
		processedMessage = strings.ReplaceAll(processedMessage, "{name}", properName)
	}
	
	// STEP 3: Fix line breaks for WhatsApp
	// WhatsApp uses actual newlines, not escaped ones
	// Convert all possible line break formats to proper newlines
	processedMessage = strings.ReplaceAll(processedMessage, "\\n", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "\\r\\n", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0A", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0a", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br/>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br />", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "[br]", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "{br}", "\n")
	
	// STEP 4: Combine greeting + proper WhatsApp line breaks + message
	// For WhatsApp, we need actual newline characters, not escaped ones
	// Using real newlines that WhatsApp will recognize
	finalMessage := greeting + "\n\n" + processedMessage
	
	// Log the message with visible line breaks for debugging
	debugMessage := strings.ReplaceAll(finalMessage, "\n", "\\n")
	logrus.Infof("[GREETING] Final message (debug view): %s", debugMessage[:min(200, len(debugMessage))])
	
	// Return the message with real newlines for WhatsApp
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// init initializes the random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}