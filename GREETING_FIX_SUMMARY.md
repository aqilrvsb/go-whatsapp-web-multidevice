# Greeting Processor Fix - January 20, 2025

## Issue Found
Messages were showing "Hi ոaṁe" instead of "Hi Aqil" - the {name} placeholder was not being replaced properly.

## Root Cause
The `processSpintax` function was using a regex that matched ALL curly braces `{...}`, including the `{name}` placeholder. This caused:

1. `{Hi|Hello|Hai}` → "Hi" ✓ (correct)
2. `{name}` → "name" ✗ (incorrect - removed the braces)
3. Later `ReplaceAll("{name}", "Aqil")` couldn't find `{name}` because it was already changed to just "name"

## The Fix
Changed the regex from:
```go
// OLD - matches everything in curly braces
regexp.MustCompile(`\{([^}]+)\}`)
```

To:
```go
// NEW - only matches spintax with | character
regexp.MustCompile(`\{([^}]*\|[^}]*)\}`)
```

This ensures:
- `{Hi|Hello|Hai}` is processed as spintax ✓
- `{name}` is left untouched for later replacement ✓

## Result
Messages now correctly show:
```
Hi Aqil 1, apa khabar

[Message content]
```

Instead of:
```
Hi ոaṁe [Message content with no line breaks]
```

## How Greetings Work

1. **Name Logic**:
   - If name is phone number (only digits) → "Cik"
   - If name is empty → "Cik"
   - Otherwise → Use actual name

2. **Format**:
   - Greeting line (e.g., "Hi Aqil, apa khabar")
   - Double line break (`\n\n`)
   - Original message content

3. **Variations**:
   - Different templates based on time of day
   - Random selection from options
   - Micro-variations (punctuation, spacing)

## Important Notes

- Greetings are applied at send time, not stored in database
- Each recipient gets a unique variation
- The `content` field in database shows original message only
- To verify greetings, check actual WhatsApp messages received

## Deployment Required
After pulling latest code:
1. Rebuild application
2. Deploy to server
3. Messages will have proper greetings with recipient names
