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
			"{Hi|Hello|Hai} {name},",
			"{name}, {hi|hello}",
			"{Salam|Hi|Hello} {name},",
			"{name}, {apa khabar|hi}",
			"{Selamat pagi|Hi} {name},", // Time-aware
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
	
	// Handle name FIRST - if phone number or empty, use "Cik"
	if isPhoneNumber(name) || name == "" {
		name = "Cik"
	}
	
	// Replace {name} placeholder BEFORE processing spintax
	template = strings.ReplaceAll(template, "{name}", name)
	
	// Process spintax AFTER name replacement
	greeting := g.processSpintax(template)
	
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
			"{Selamat pagi|Pagi} {name},",
			"{Hi|Hello|Hai} {name},",
			"{name}, {selamat pagi|hi}",
		}
		return morningTemplates[rand.Intn(len(morningTemplates))]
	}
	
	// Evening/Night - avoid "pagi"
	if hour >= 19 {
		eveningTemplates := []string{
			"{Hi|Hello|Hai} {name},",
			"{name}, {hi|hello}",
			"{Salam|Hi} {name},",
		}
		return eveningTemplates[rand.Intn(len(eveningTemplates))]
	}
	
	// Afternoon - use any template
	return g.templates[rand.Intn(len(g.templates))]
}

// processSpintax processes spintax notation {option1|option2|option3}
func (g *GreetingProcessor) processSpintax(template string) string {
	spintaxRegex := regexp.MustCompile(`\{([^}]+)\}`)
	
	return spintaxRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Remove braces and split by |
		options := strings.Split(strings.Trim(match, "{}"), "|")
		// Randomly SELECT one option
		return options[rand.Intn(len(options))]
	})
}

// applyMicroVariations adds subtle variations to greeting
func (g *GreetingProcessor) applyMicroVariations(greeting string) string {
	// For greetings that already end with comma, don't add more punctuation
	if strings.HasSuffix(greeting, ",") {
		return greeting
	}
	
	// Randomly (30% chance) add punctuation variations
	if rand.Float32() < 0.3 {
		variations := []string{
			greeting + "!",
			greeting + ".",
			greeting + ",",
			greeting + " 👋", // WhatsApp supports emojis
		}
		greeting = variations[rand.Intn(len(variations))]
	}
	
	// Randomly (20% chance) vary capitalization
	if rand.Float32() < 0.2 {
		// Lowercase first character for casual feel
		if len(greeting) > 0 {
			greeting = strings.ToLower(string(greeting[0])) + greeting[1:]
		}
	}
	
	return greeting
}

// hash generates a deterministic hash from a string
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// isPhoneNumber checks if the name looks like a phone number
func isPhoneNumber(name string) bool {
	// Remove all non-digits
	digitsOnly := regexp.MustCompile(`\D`).ReplaceAllString(name, "")
	// If more than 5 digits, probably a phone number
	return len(digitsOnly) > 5
}

// isSingleName checks if name is a single word (no surname)
func isSingleName(name string) bool {
	parts := strings.Fields(name)
	return len(parts) <= 1
}

// PrepareMessageWithGreeting adds greeting to original message
func (g *GreetingProcessor) PrepareMessageWithGreeting(originalMessage string, name string, deviceID string, recipientPhone string) string {
	// Get unique greeting
	greeting := g.GetAntiSpamGreeting(name, deviceID, recipientPhone)
	
	// Process original message to handle any spintax it might contain
	processedMessage := g.processSpintax(originalMessage)
	
	// Fix line breaks for WhatsApp
	// Replace common line break patterns with proper WhatsApp line breaks
	processedMessage = strings.ReplaceAll(processedMessage, "\\n", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0A", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "%0a", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br/>", "\n")
	processedMessage = strings.ReplaceAll(processedMessage, "<br />", "\n")
	
	// Combine with proper line breaks for WhatsApp
	// Using \n\n for double line break to create a blank line
	return greeting + "\n\n" + processedMessage
}
