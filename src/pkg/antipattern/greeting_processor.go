package antipattern

import (
	"hash/fnv"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// GreetingProcessor handles Malaysian greeting variations for anti-spam
type GreetingProcessor struct {
	templates []string
}

// NewGreetingProcessor creates a new greeting processor with Malaysian templates
func NewGreetingProcessor() *GreetingProcessor {
	return &GreetingProcessor{
		templates: []string{
			"{Hi|Hello|Hai} {name}",
			"{Hi|Hello} {name}",
			"{Salam|Hi|Hello} {name}",
			"{Hai|Hi} {name}, {apa khabar}",
			"{Selamat pagi|Hi} {name}", // Time-aware
		},
	}
}

// GetAntiSpamGreeting generates a unique greeting for each recipient
func (g *GreetingProcessor) GetAntiSpamGreeting(name string, deviceID string, recipientPhone string) string {
	// 1. Device-specific seed (each device has different pattern)
	deviceSeed := hash(deviceID + time.Now().Format("2006-01-02"))
	
	// 2. Recipient-specific variation
	recipientSeed := hash(recipientPhone)
	
	// 3. Time-based variation (changes every 4 hours)
	timeSeed := time.Now().Hour() / 4
	
	// Combine seeds for unique selection
	rand.Seed(int64(deviceSeed + recipientSeed + uint32(timeSeed)))
	
	// Select template based on time if applicable
	template := g.selectTimeAppropriateTemplate()
	
	// Process spintax
	greeting := g.processSpintax(template)
	
	// Handle name - if phone number or empty, use "Cik"
	if isPhoneNumber(name) || name == "" {
		name = "Cik"
	}
	
	greeting = strings.ReplaceAll(greeting, "{name}", name)
	
	// Apply micro-variations
	return g.applyMicroVariations(greeting)
}

// selectTimeAppropriateTemplate chooses template based on current time
func (g *GreetingProcessor) selectTimeAppropriateTemplate() string {
	hour := time.Now().Hour()
	
	// Morning templates (5am - 12pm)
	if hour >= 5 && hour < 12 {
		// Prefer templates with "Selamat pagi"
		morningTemplates := []string{
			"{Selamat pagi|Pagi} {name}",
			"{Hi|Hello|Hai} {name}",
			"{Selamat pagi} {name}, {apa khabar}",
		}
		return morningTemplates[rand.Intn(len(morningTemplates))]
	}
	
	// Evening/Night - avoid "pagi"
	if hour >= 19 {
		eveningTemplates := []string{
			"{Hi|Hello|Hai} {name}",
			"{Selamat malam|Hi} {name}",
			"{Salam|Hi} {name}",
		}
		return eveningTemplates[rand.Intn(len(eveningTemplates))]
	}
	
	// Afternoon - use any template
	return g.templates[rand.Intn(len(g.templates))]
}

// processSpintax processes spintax notation {option1|option2|option3}
func (g *GreetingProcessor) processSpintax(template string) string {
	// Only process spintax that contains | (not placeholders like {name})
	spintaxRegex := regexp.MustCompile(`\{([^}]*\|[^}]*)\}`)
	
	return spintaxRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Remove braces and split by |
		options := strings.Split(strings.Trim(match, "{}"), "|")
		// Randomly select one option
		return options[rand.Intn(len(options))]
	})
}

// applyMicroVariations adds subtle variations to prevent pattern detection
func (g *GreetingProcessor) applyMicroVariations(text string) string {
	// Punctuation variations (50% chance)
	r := rand.Float32()
	if r < 0.3 {
		text = text + "."
	} else if r < 0.5 {
		text = text + ","
	}
	
	// Case variations (20% chance)
	if rand.Float32() < 0.2 && len(text) > 0 {
		// Lowercase first letter
		text = strings.ToLower(text[:1]) + text[1:]
	}
	
	// Spacing variations (invisible but different)
	if rand.Float32() < 0.4 {
		text = text + " " // Extra space at end
	}
	
	return text
}

// isPhoneNumber checks if the name looks like a phone number
func isPhoneNumber(name string) bool {
	// Remove spaces and check if it's mostly digits
	cleaned := strings.ReplaceAll(name, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	
	// If empty after cleaning, it's not a valid name
	if cleaned == "" {
		return true
	}
	
	// Check if at least 80% are digits (to handle cases like +60-123-456-789)
	digitCount := 0
	for _, ch := range cleaned {
		if ch >= '0' && ch <= '9' {
			digitCount++
		}
	}
	
	// If more than 80% digits, it's likely a phone number
	return float64(digitCount) / float64(len(cleaned)) > 0.8
}

// hash creates a simple hash from string
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// GetMessageDelay calculates delay to prevent burst sending
func GetMessageDelay(deviceID string, messageCount int) time.Duration {
	baseDelay := 5 * time.Second
	
	// Add random jitter (0-10 seconds)
	jitter := time.Duration(rand.Intn(10)) * time.Second
	
	// Increase delay after every 50 messages
	batchDelay := time.Duration(messageCount/50) * 30 * time.Second
	
	return baseDelay + jitter + batchDelay
}

// PrepareMessageWithGreeting adds greeting to original message
func (g *GreetingProcessor) PrepareMessageWithGreeting(originalMessage string, name string, deviceID string, recipientPhone string) string {
	// Get unique greeting
	greeting := g.GetAntiSpamGreeting(name, deviceID, recipientPhone)
	
	// Combine with double line break for WhatsApp formatting
	return greeting + "\n\n" + originalMessage
}
