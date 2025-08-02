package antipattern

import (
	"strings"
	"time"
	"math/rand"
	"unicode"
	"github.com/sirupsen/logrus"
)

// GreetingProcessor handles greeting for messages
type GreetingProcessor struct {
	// Malaysian greetings - NO SPINTAX, will randomly pick one
	greetings []string
	location  *time.Location
}

// NewGreetingProcessor creates a new greeting processor
func NewGreetingProcessor() *GreetingProcessor {
	// Always use Malaysia timezone
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		logrus.Warnf("Failed to load Malaysia timezone, using local time: %v", err)
		loc = time.Local
	}
	
	return &GreetingProcessor{
		greetings: []string{
			"Salam {name}",
			"Hi {name}",
			"Apa khabar {name}",
			"Maaf ganggu {name}",
			"Pinjam masa {name}",
		},
		location: loc,
	}
}

// cleanName removes all numbers from the name and returns cleaned name
func cleanName(name string) string {
	var result strings.Builder
	
	for _, r := range name {
		// Keep only letters and spaces
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	
	// Trim any extra spaces
	cleaned := strings.TrimSpace(result.String())
	
	logrus.Debugf("[GREETING] Name '%s' cleaned to '%s'", name, cleaned)
	
	return cleaned
}

// getSimpleGreeting returns a Malaysian greeting based on time or random
func (g *GreetingProcessor) getSimpleGreeting(name string) string {
	// Debug log the original name
	logrus.Debugf("[GREETING] Original name: '%s'", name)
	
	// Clean the name by removing all numbers
	cleanedName := cleanName(name)
	
	// If cleaned name is empty, use "Cik"
	if cleanedName == "" {
		logrus.Debugf("[GREETING] Cleaned name is empty, using 'Cik'")
		name = "Cik"
	} else {
		logrus.Debugf("[GREETING] Using cleaned name: '%s'", cleanedName)
		name = cleanedName
	}
	
	// Get current hour in Malaysia timezone
	now := time.Now().In(g.location)
	hour := now.Hour()
	
	logrus.Debugf("[GREETING] Malaysia time: %s (hour: %d)", now.Format("15:04:05"), hour)
	
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
		logrus.Debugf("[GREETING] Morning greeting selected")
	} else if hour >= 12 && hour < 15 {
		// Afternoon
		afternoonGreetings := []string{
			"Selamat tengahari {name}",
			"Salam {name}",
			"Hi {name}",
		}
		greeting = afternoonGreetings[rand.Intn(len(afternoonGreetings))]
		logrus.Debugf("[GREETING] Afternoon greeting selected")
	} else if hour >= 15 && hour < 19 {
		// Late afternoon/evening
		eveningGreetings := []string{
			"Selamat petang {name}",
			"Petang {name}",
			"Salam {name}",
		}
		greeting = eveningGreetings[rand.Intn(len(eveningGreetings))]
		logrus.Debugf("[GREETING] Evening greeting selected")
	} else {
		// Night (7pm - 5am)
		nightGreetings := []string{
			"Selamat malam {name}",
			"Malam {name}",
			"Maaf ganggu {name}",
			"Pinjam masa {name}",
		}
		greeting = nightGreetings[rand.Intn(len(nightGreetings))]
		logrus.Debugf("[GREETING] Night greeting selected")
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
	logrus.Infof("[GREETING] START - Processing message for recipient_name='%s', phone='%s'", name, recipientPhone)
	logrus.Infof("[GREETING] Name length: %d, Name bytes: %v", len(name), []byte(name))
	
	// STEP 1: Get the simple Malaysian greeting (NO SPINTAX!)
	greeting := g.getSimpleGreeting(name)
	logrus.Infof("[GREETING] Generated greeting: '%s'", greeting)
	
	// STEP 2: Process the message content
	// First replace {name} in the message if exists
	processedMessage := originalMessage
	if strings.Contains(processedMessage, "{name}") {
		// Use cleaned name for message content too
		properName := cleanName(name)
		if properName == "" {
			properName = "Cik"
		}
		processedMessage = strings.ReplaceAll(processedMessage, "{name}", properName)
	}
	
	// STEP 3: Fix line breaks for WhatsApp
	// Log original message to see what we're getting
	logrus.Infof("[GREETING] Original message: '%s'", originalMessage)
	logrus.Infof("[GREETING] Original message bytes: %v", []byte(originalMessage))
	
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
	logrus.Infof("[GREETING] Final message bytes (first 100): %v", []byte(finalMessage)[:min(100, len([]byte(finalMessage)))])
	
	// Return the message with real newlines for WhatsApp
	return finalMessage
}

// isPhoneNumber - NOT USED ANYMORE but kept for compatibility
func isPhoneNumber(name string) bool {
	// This function is no longer used
	// We now use cleanName() instead
	return false
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