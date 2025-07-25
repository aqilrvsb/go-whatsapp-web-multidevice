package antipattern

import (
	"math/rand"
	"strings"
	"time"
	"unicode"
)

// MessageRandomizer handles message anti-pattern techniques
type MessageRandomizer struct {
	homoglyphs map[rune][]rune
	zeroWidthChars []string
}

// NewMessageRandomizer creates a new message randomizer
func NewMessageRandomizer() *MessageRandomizer {
	return &MessageRandomizer{
		homoglyphs: initHomoglyphs(),
		zeroWidthChars: []string{
			"\u200B", // Zero-width space
			"\u200C", // Zero-width non-joiner
			"\u200D", // Zero-width joiner
			"\uFEFF", // Zero-width no-break space
		},
	}
}

// RandomizeMessage applies all anti-pattern techniques
func (mr *MessageRandomizer) RandomizeMessage(message string) string {
	if message == "" {
		return message
	}
	
	// Apply techniques in order
	message = mr.applyHomoglyphs(message, 0.05) // 5% of characters for better readability
	message = mr.insertZeroWidthSpaces(message, 2) // 2 zero-width spaces
	message = mr.randomizePunctuation(message)
	
	return message
}

// applyHomoglyphs replaces a percentage of characters with look-alikes
func (mr *MessageRandomizer) applyHomoglyphs(text string, percentage float32) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	
	// Calculate how many characters to replace
	replaceCount := int(float32(len(runes)) * percentage)
	if replaceCount == 0 {
		replaceCount = 1 // At least one
	}
	
	// Get random positions to replace (only letters)
	positions := mr.getLetterPositions(runes)
	if len(positions) == 0 {
		return text
	}
	
	// Shuffle and take only needed positions
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})
	
	if len(positions) > replaceCount {
		positions = positions[:replaceCount]
	}
	
	// Replace characters at selected positions
	for _, pos := range positions {
		char := unicode.ToLower(runes[pos])
		if replacements, exists := mr.homoglyphs[char]; exists && len(replacements) > 0 {
			// Pick random homoglyph
			replacement := replacements[rand.Intn(len(replacements))]
			
			// Preserve original case
			if unicode.IsUpper(runes[pos]) {
				replacement = unicode.ToUpper(replacement)
			}
			
			runes[pos] = replacement
		}
	}
	
	return string(runes)
}

// insertZeroWidthSpaces adds invisible characters between words
func (mr *MessageRandomizer) insertZeroWidthSpaces(text string, count int) string {
	if count <= 0 || len(text) == 0 {
		return text
	}
	
	// CRITICAL FIX: Don't use strings.Fields as it removes line breaks!
	// Instead, work with the original text and find word boundaries
	
	// Find positions after words (but not after newlines)
	var positions []int
	runes := []rune(text)
	inWord := false
	
	for i := 0; i < len(runes)-1; i++ {
		current := runes[i]
		next := runes[i+1]
		
		// Check if we're at the end of a word (not newline)
		if inWord && !unicode.IsLetter(current) && current != '\n' && unicode.IsLetter(next) {
			positions = append(positions, i+1)
		}
		
		inWord = unicode.IsLetter(current)
	}
	
	if len(positions) == 0 {
		return text
	}
	
	// Limit count to available positions
	if count > len(positions) {
		count = len(positions)
	}
	
	// Shuffle positions
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})
	
	// Take only needed positions
	positions = positions[:count]
	
	// Sort positions in reverse order to insert from end
	for i := 0; i < len(positions)-1; i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[i] < positions[j] {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}
	
	// Insert zero-width characters at selected positions
	result := text
	for _, pos := range positions {
		zeroWidth := mr.zeroWidthChars[rand.Intn(len(mr.zeroWidthChars))]
		result = result[:pos] + zeroWidth + result[pos:]
	}
	
	return result
}

// randomizePunctuation adds subtle variations to punctuation
func (mr *MessageRandomizer) randomizePunctuation(text string) string {
	// Random variations
	variations := []struct{
		probability float32
		transform   func(string) string
	}{
		{0.1, func(s string) string { return strings.ReplaceAll(s, "!", "! ") }},
		{0.1, func(s string) string { return strings.ReplaceAll(s, "?", "? ") }},
		{0.1, func(s string) string { return strings.ReplaceAll(s, ".", ". ") }},
		{0.05, func(s string) string { return strings.ReplaceAll(s, ",", ", ") }},
	}
	
	for _, v := range variations {
		if rand.Float32() < v.probability {
			text = v.transform(text)
		}
	}
	
	// Clean up multiple spaces
	text = strings.ReplaceAll(text, "  ", " ")
	
	return strings.TrimSpace(text)
}

// getLetterPositions returns positions of letters in the text
func (mr *MessageRandomizer) getLetterPositions(runes []rune) []int {
	positions := []int{}
	for i, r := range runes {
		if unicode.IsLetter(r) {
			positions = append(positions, i)
		}
	}
	return positions
}

// initHomoglyphs initializes the homoglyph map
func initHomoglyphs() map[rune][]rune {
	return map[rune][]rune{
		'a': {'а', 'ɑ', 'α'}, // Cyrillic а, Latin alpha, Greek alpha
		'b': {'Ь', 'ƅ', 'ḃ'}, // Cyrillic soft sign, Latin b with stroke
		'c': {'с', 'ϲ', 'ć'}, // Cyrillic с, Greek lunate sigma
		'd': {'ԁ', 'ɗ', 'ḍ'}, // Cyrillic d, Latin d with hook
		'e': {'е', 'ė', 'ẹ'}, // Cyrillic е, Latin e with dot
		'f': {'ƒ', 'ḟ'},      // Latin f with hook, f with dot
		'g': {'ɡ', 'ġ', 'ǵ'}, // Latin script g, g with dot
		'h': {'һ', 'ḣ', 'ḥ'}, // Cyrillic һ, Latin h with dot
		'i': {'і', 'ı', 'ḭ'}, // Cyrillic і, Latin dotless i
		'j': {'ј', 'ĵ', 'ǰ'}, // Cyrillic ј, Latin j with circumflex
		'k': {'κ', 'ḳ', 'ķ'}, // Greek kappa, Latin k with dot
		'l': {'ⅼ', 'ḷ', 'ļ'}, // Small Roman numeral, Latin l with dot
		'm': {'м', 'ṁ', 'ḿ'}, // Cyrillic м, Latin m with dot
		'n': {'ո', 'ṅ', 'ń'}, // Armenian vo, Latin n with dot
		'o': {'о', 'ο', 'ȯ'}, // Cyrillic о, Greek omicron
		'p': {'р', 'ρ', 'ṗ'}, // Cyrillic р, Greek rho
		'q': {'ԛ', 'ɋ'},      // Cyrillic qa
		'r': {'г', 'ṙ', 'ŕ'}, // Cyrillic г, Latin r with dot
		's': {'ѕ', 'ṡ', 'ś'}, // Cyrillic ѕ, Latin s with dot
		't': {'τ', 'ṫ', 'ť'}, // Greek tau, Latin t with dot
		'u': {'υ', 'ս', 'ů'}, // Greek upsilon, Armenian
		'v': {'ν', 'ѵ', 'ṿ'}, // Greek nu, Cyrillic v
		'w': {'ԝ', 'ẇ', 'ẃ'}, // Cyrillic w, Latin w with dot
		'x': {'х', 'ẋ', 'ẍ'}, // Cyrillic х, Latin x with dot
		'y': {'у', 'ү', 'ẏ'}, // Cyrillic у, Latin y with dot
		'z': {'ᴢ', 'ż', 'ź'}, // Small capital z, Latin z with dot
	}
}

// GetRandomDelay returns a random delay between min and max seconds
func GetRandomDelay(minSeconds, maxSeconds int) time.Duration {
	if minSeconds >= maxSeconds {
		return time.Duration(minSeconds) * time.Second
	}
	
	delay := rand.Intn(maxSeconds-minSeconds) + minSeconds
	// Add random milliseconds for more variation
	milliseconds := rand.Intn(1000)
	
	return time.Duration(delay)*time.Second + time.Duration(milliseconds)*time.Millisecond
}

// AddTypingDelay returns a random typing duration based on message length
func AddTypingDelay(messageLength int) time.Duration {
	// Base: 50ms per character + random variation
	baseTime := messageLength * 50
	variation := rand.Intn(baseTime / 2) // 0-50% variation
	
	totalMs := baseTime + variation
	
	// Cap between 2-8 seconds
	if totalMs < 2000 {
		totalMs = 2000
	} else if totalMs > 8000 {
		totalMs = 8000
	}
	
	return time.Duration(totalMs) * time.Millisecond
}

// init initializes the random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
